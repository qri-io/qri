package access

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/profile"
)

var (
	// Timestamp is a replacable function for getting the current time,
	// can be overridden for tests
	Timestamp = func() time.Time { return time.Now() }
	// ErrTokenNotFound is returned by stores that cannot find an access token
	// for a given key
	ErrTokenNotFound = errors.New("access token not found")
	// ErrInvalidToken indicates an access token is invalid
	ErrInvalidToken = errors.New("invalid access token")
	// DefaultTokenTTL is the default
	DefaultTokenTTL = time.Hour * 24 * 14
)

// Token abstracts a json web token
type Token = jwt.Token

// TokenClaims is a JWT Claims object
type TokenClaims struct {
	*jwt.StandardClaims
	Username string `json:"username"`
}

// ParseToken will parse, validate and return a token
func ParseToken(tokenString string, tokens TokenSource) (*Token, error) {
	return jwt.Parse(tokenString, tokens.VerificationKey)
}

// TokenSource creates tokens, and provides a verification key for all tokens
// it creates
//
// implementations of TokenSource must conform to the assertion test defined
// in the spec subpackage
type TokenSource interface {
	CreateToken(pro *profile.Profile, ttl time.Duration) (string, error)
	CreateTokenWithClaims(claims jwt.MapClaims, ttl time.Duration) (string, error)
	// VerifyKey returns the verification key for a given token
	VerificationKey(t *Token) (interface{}, error)
}

type pkTokenSource struct {
	pk            crypto.PrivKey
	signingMethod jwt.SigningMethod
	verifyKey     *rsa.PublicKey
	signKey       *rsa.PrivateKey
}

// assert pkTokenSource implements TokenSource at compile time
var _ TokenSource = (*pkTokenSource)(nil)

// NewPrivKeyTokenSource creates an authentication interface backed by a single
// private key. Intended for a node running as remote, or providing a public API
func NewPrivKeyTokenSource(privKey crypto.PrivKey) (TokenSource, error) {
	methodStr := ""
	keyType := privKey.Type().String()
	switch keyType {
	case "RSA":
		methodStr = "RS256"
	default:
		return nil, fmt.Errorf("unsupported key type for token creation: %q", keyType)
	}

	signingMethod := jwt.GetSigningMethod(methodStr)

	rawPrivBytes, err := privKey.Raw()
	if err != nil {
		return nil, err
	}
	signKey, err := x509.ParsePKCS1PrivateKey(rawPrivBytes)
	if err != nil {
		return nil, err
	}

	rawPubBytes, err := privKey.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	verifyKeyiface, err := x509.ParsePKIXPublicKey(rawPubBytes)
	if err != nil {
		return nil, err
	}

	verifyKey, ok := verifyKeyiface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not an RSA key. got type: %T", verifyKeyiface)
	}

	return &pkTokenSource{
		pk:            privKey,
		signingMethod: signingMethod,
		verifyKey:     verifyKey,
		signKey:       signKey,
	}, nil
}

// CreateToken returns a new JWT token
func (a *pkTokenSource) CreateToken(pro *profile.Profile, ttl time.Duration) (string, error) {
	// create a signer for rsa 256
	t := jwt.New(a.signingMethod)

	var exp int64
	if ttl != time.Duration(0) {
		exp = Timestamp().Add(ttl).In(time.UTC).Unix()
	}

	// set our claims
	t.Claims = &TokenClaims{
		StandardClaims: &jwt.StandardClaims{
			Subject: pro.ID.String(),
			// set the expire time
			// see http://tools.ietf.org/html/draft-ietf-oauth-json-web-token-20#section-4.1.4
			ExpiresAt: exp,
		},
		Username: pro.Peername,
	}

	// Creat token string
	return t.SignedString(a.signKey)
}

// CreateToken returns a new JWT token from provided claims
func (a *pkTokenSource) CreateTokenWithClaims(claims jwt.MapClaims, ttl time.Duration) (string, error) {
	// create a signer for rsa 256
	t := jwt.New(a.signingMethod)

	var exp int64
	if ttl != time.Duration(0) {
		exp = Timestamp().Add(ttl).In(time.UTC).Unix()
	}
	claims["exp"] = exp
	t.Claims = claims

	// Creat token string
	return t.SignedString(a.signKey)
}

// VerifyKey returns the verification key
// its packaged as an interface for easy extensibility in the future
func (a *pkTokenSource) VerificationKey(t *Token) (interface{}, error) {
	if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
	}
	return a.verifyKey, nil
}

// TokenStore is a store intended for clients, who need to persist secret jwts
// given to them by other remotes for API access. It deals in raw,
// string-formatted json web tokens, which are more useful when working with
// APIs, but validates the tokens are well-formed when placed in the store
//
// implementations of TokenStore must conform to the assertion test defined
// in the spec subpackage
type TokenStore interface {
	PutToken(ctx context.Context, key, rawToken string) error
	RawToken(ctx context.Context, key string) (rawToken string, err error)
	DeleteToken(ctx context.Context, key string) (err error)
	ListTokens(ctx context.Context, offset, limit int) (results []RawToken, err error)
}

// RawToken is a struct that binds a key to a raw token string
type RawToken struct {
	Key string
	Raw string
}

// RawTokens is a list of tokens that implements sorting by keys
type RawTokens []RawToken

func (rts RawTokens) Len() int           { return len(rts) }
func (rts RawTokens) Less(a, b int) bool { return rts[a].Key < rts[b].Key }
func (rts RawTokens) Swap(i, j int)      { rts[i], rts[j] = rts[j], rts[i] }

type qfsTokenStore struct {
	path string
	fs   qfs.Filesystem

	toksLk sync.Mutex
	toks   map[string]string
}

var _ TokenStore = (*qfsTokenStore)(nil)

// NewTokenStore creates a token store with a qfs.Filesystem
func NewTokenStore(filepath string, fs qfs.Filesystem) (TokenStore, error) {
	toks := map[string]string{}
	if f, err := fs.Get(context.Background(), filepath); err == nil {
		rawToks := []RawToken{}
		if err := json.NewDecoder(f).Decode(&rawToks); err != nil {
			return nil, fmt.Errorf("invalid token store file: %w", err)
		}
		for _, t := range rawToks {
			toks[t.Key] = t.Raw
		}
	} else {
		if err.Error() == "path not found" {
			log.Errorf("path not found")
			// TODO(arqu): handle Not Found
		} else {
			return nil, fmt.Errorf("error creating token store: %w", err)
		}
	}

	return &qfsTokenStore{
		path: filepath,
		fs:   fs,
		toks: toks,
	}, nil
}

func (st *qfsTokenStore) PutToken(ctx context.Context, key string, raw string) error {
	p := &jwt.Parser{
		UseJSONNumber:        true,
		SkipClaimsValidation: false,
	}
	if _, _, err := p.ParseUnverified(raw, &TokenClaims{}); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidToken, err)
	}

	st.toksLk.Lock()
	defer st.toksLk.Unlock()

	st.toks[key] = raw
	return st.save(ctx)
}

func (st *qfsTokenStore) RawToken(ctx context.Context, key string) (rawToken string, err error) {
	t, ok := st.toks[key]
	if !ok {
		return "", ErrTokenNotFound
	}
	return t, nil
}

func (st *qfsTokenStore) DeleteToken(ctx context.Context, key string) (err error) {
	st.toksLk.Lock()
	defer st.toksLk.Unlock()

	if _, ok := st.toks[key]; !ok {
		return ErrTokenNotFound
	}
	delete(st.toks, key)
	return st.save(ctx)
}

func (st *qfsTokenStore) ListTokens(ctx context.Context, offset, limit int) ([]RawToken, error) {
	results := make([]RawToken, 0, limit+1)

	toks := st.toRawTokens()
	for i := 0; i < len(toks); i++ {
		if offset > 0 {
			offset--
			continue
		}
		results = append(results, toks[i])
		if limit > 0 && len(results) == limit {
			break
		}
	}

	return results, nil
}

func (st *qfsTokenStore) toRawTokens() RawTokens {
	toks := make(RawTokens, len(st.toks))
	i := 0
	for key, t := range st.toks {
		toks[i] = RawToken{
			Key: key,
			Raw: t,
		}
		i++
	}
	sort.Sort(toks)
	return toks
}

func (st *qfsTokenStore) save(ctx context.Context) error {
	data, err := json.MarshalIndent(st.toRawTokens(), "", "  ")
	if err != nil {
		return err
	}
	f := qfs.NewMemfileBytes(st.path, data)
	path, err := st.fs.Put(ctx, f)
	if err != nil {
		return err
	}
	st.path = path
	return nil
}
