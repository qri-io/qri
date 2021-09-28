package token

import (
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	golog "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/auth/key"
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

	log = golog.Logger("token")
)

func init() {
	golog.SetLogLevel("token", "error")
}

// Token abstracts a json web token
type Token = jwt.Token

// Claims is a JWT Claims object
type Claims struct {
	*jwt.StandardClaims
	ClientType ClientType `json:"clientType"`
}

// Parse will parse, validate and return a token
func Parse(tokenString string, tokens Source) (*Token, error) {
	return jwt.Parse(tokenString, tokens.VerificationKey)
}

// ParseWithClaims will parse, validate and return a token with claims
func ParseWithClaims(tokenString string, claims *Claims, tokens Source) (*Token, error) {
	return jwt.ParseWithClaims(tokenString, claims, tokens.VerificationKey)
}

// NewPrivKeyAuthToken creates a JWT token string suitable for making requests
// authenticated as the given private key
func NewPrivKeyAuthToken(pk crypto.PrivKey, profileID string, ttl time.Duration) (string, error) {
	signingMethod, err := jwtSigningMethod(pk)
	if err != nil {
		return "", err
	}

	t := jwt.New(signingMethod)

	id, err := key.IDFromPrivKey(pk)
	if err != nil {
		return "", err
	}

	rawPrivBytes, err := pk.Raw()
	if err != nil {
		return "", err
	}

	var signKey interface{}

	switch pk.Type() {
	case crypto.RSA:
		// TODO(b5) - detect if key is encoded as PEM block, here we're assuming it is
		signKey, err = x509.ParsePKCS1PrivateKey(rawPrivBytes)
		if err != nil {
			return "", err
		}
	case crypto.Ed25519:
		signKey = ed25519.PrivateKey(rawPrivBytes)
	default:
		return "", fmt.Errorf("unsupported key type for token creation: %q", pk.Type())
	}

	var exp int64
	if ttl != time.Duration(0) {
		exp = Timestamp().Add(ttl).In(time.UTC).Unix()
	}

	// set our claims
	t.Claims = &Claims{
		StandardClaims: &jwt.StandardClaims{
			Issuer:  id,
			Subject: profileID,
			// set the expire time
			// see http://tools.ietf.org/html/draft-ietf-oauth-json-web-token-20#section-4.1.4
			ExpiresAt: exp,
		},
		ClientType: UserClient,
	}

	return t.SignedString(signKey)
}

// ParseAuthToken will parse, validate and return a token
func ParseAuthToken(ctx context.Context, tokenString string, keystore key.Store) (*Token, error) {
	claims := &Claims{}
	return jwt.ParseWithClaims(tokenString, claims, func(t *Token) (interface{}, error) {
		pid, err := key.DecodeID(claims.Issuer)
		if err != nil {
			return nil, err
		}
		pubKey := keystore.PubKey(ctx, pid)
		if pubKey == nil {
			return nil, fmt.Errorf("cannot verify key. missing public key for id %s", claims.Issuer)
		}
		rawPubBytes, err := pubKey.Raw()
		if err != nil {
			return nil, err
		}

		switch pubKey.Type() {
		case crypto.RSA:
			verifyKeyiface, err := x509.ParsePKIXPublicKey(rawPubBytes)
			if err != nil {
				return nil, err
			}

			verifyKey, ok := verifyKeyiface.(*rsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("public key is not an RSA key. got type: %T", verifyKeyiface)
			}
			return verifyKey, nil
		case crypto.Ed25519:
			return ed25519.PublicKey(rawPubBytes), nil
		default:
			return nil, fmt.Errorf("unsupported key type: %q", pubKey.Type())
		}
	})
}

// Source creates tokens, and provides a verification key for all tokens
// it creates
//
// implementations of Source must conform to the assertion test defined
// in the spec subpackage
type Source interface {
	CreateToken(pro *profile.Profile, ttl time.Duration) (string, error)
	CreateTokenWithClaims(claims *Claims, ttl time.Duration) (string, error)
	// VerifyKey returns the verification key for a given token
	VerificationKey(t *Token) (interface{}, error)
}

type pkSource struct {
	pk            crypto.PrivKey
	signingMethod jwt.SigningMethod

	verifyKey interface{} // one of: *rsa.PublicKey, *edsa.PublicKey
	signKey   interface{} // one of: *rsa.PrivateKey,
}

// assert pkSource implements Source at compile time
var _ Source = (*pkSource)(nil)

// NewPrivKeySource creates an authentication interface backed by a single
// private key. Intended for a node running as remote, or providing a public API
func NewPrivKeySource(privKey crypto.PrivKey) (Source, error) {
	rawPrivBytes, err := privKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("getting private key bytes: %w", err)
	}

	var (
		methodStr = ""
		keyType   = privKey.Type()
		signKey   interface{}
		verifyKey interface{}
	)

	switch keyType {
	case crypto.RSA:
		methodStr = "RS256"
		// TODO(b5) - detect if key is encoded as PEM block, here we're assuming it is
		signKey, err = x509.ParsePKCS1PrivateKey(rawPrivBytes)
		if err != nil {
			return nil, err
		}
		rawPubBytes, err := privKey.GetPublic().Raw()
		if err != nil {
			return nil, fmt.Errorf("getting raw public key bytes: %w", err)
		}
		verifyKeyiface, err := x509.ParsePKIXPublicKey(rawPubBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing public key bytes: %w", err)
		}
		var ok bool
		verifyKey, ok = verifyKeyiface.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key is not an RSA key. got type: %T", verifyKeyiface)
		}
	case crypto.Ed25519:
		methodStr = "EdDSA"
		signKey = ed25519.PrivateKey(rawPrivBytes)
		rawPubBytes, err := privKey.GetPublic().Raw()
		if err != nil {
			return nil, fmt.Errorf("getting raw public key bytes: %w", err)
		}
		verifyKey = ed25519.PublicKey(rawPubBytes)
	default:
		return nil, fmt.Errorf("unsupported key type for token creation: %q", keyType)
	}

	return &pkSource{
		pk:            privKey,
		signingMethod: jwt.GetSigningMethod(methodStr),
		verifyKey:     verifyKey,
		signKey:       signKey,
	}, nil
}

// CreateToken returns a new JWT token
func (a *pkSource) CreateToken(pro *profile.Profile, ttl time.Duration) (string, error) {
	// set our claims
	claims := &Claims{
		StandardClaims: &jwt.StandardClaims{
			Subject: pro.ID.Encode(),
			Issuer:  pro.ID.Encode(),
		},
		ClientType: UserClient,
	}

	return a.CreateTokenWithClaims(claims, ttl)
}

// CreateToken returns a new JWT token from provided claims
func (a *pkSource) CreateTokenWithClaims(claims *Claims, ttl time.Duration) (string, error) {
	if claims == nil {
		return "", fmt.Errorf("empty token claims")
	}
	// create a signer for rsa 256
	t := jwt.New(a.signingMethod)

	var exp int64
	if ttl != time.Duration(0) {
		exp = Timestamp().Add(ttl).In(time.UTC).Unix()
	}
	// set the expire time
	// see http://tools.ietf.org/html/draft-ietf-oauth-json-web-token-20#section-4.1.4
	claims.StandardClaims.ExpiresAt = exp
	t.Claims = claims

	// Creat token string
	return t.SignedString(a.signKey)
}

// VerifyKey returns the verification key
// its packaged as an interface for easy extensibility in the future
func (a *pkSource) VerificationKey(t *Token) (interface{}, error) {
	if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
	}
	return a.verifyKey, nil
}

// Store is a store intended for clients, who need to persist secret jwts
// given to them by other remotes for API access. It deals in raw,
// string-formatted json web tokens, which are more useful when working with
// APIs, but validates the tokens are well-formed when placed in the store
//
// implementations of Store must conform to the assertion test defined
// in the spec subpackage
type Store interface {
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

type qfsStore struct {
	path string
	fs   qfs.Filesystem

	toksLk sync.Mutex
	toks   map[string]string
}

var _ Store = (*qfsStore)(nil)

// NewStore creates a token store with a qfs.Filesystem
func NewStore(filepath string, fs qfs.Filesystem) (Store, error) {
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
			// TODO(arqu): handle Not Found
		} else {
			return nil, fmt.Errorf("error creating token store: %w", err)
		}
	}

	return &qfsStore{
		path: filepath,
		fs:   fs,
		toks: toks,
	}, nil
}

func (st *qfsStore) PutToken(ctx context.Context, key string, raw string) error {
	p := &jwt.Parser{
		UseJSONNumber:        true,
		SkipClaimsValidation: false,
	}
	if _, _, err := p.ParseUnverified(raw, &Claims{}); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidToken, err)
	}

	st.toksLk.Lock()
	defer st.toksLk.Unlock()

	st.toks[key] = raw
	return st.save(ctx)
}

func (st *qfsStore) RawToken(ctx context.Context, key string) (rawToken string, err error) {
	t, ok := st.toks[key]
	if !ok {
		return "", ErrTokenNotFound
	}
	return t, nil
}

func (st *qfsStore) DeleteToken(ctx context.Context, key string) (err error) {
	st.toksLk.Lock()
	defer st.toksLk.Unlock()

	if _, ok := st.toks[key]; !ok {
		return ErrTokenNotFound
	}
	delete(st.toks, key)
	return st.save(ctx)
}

func (st *qfsStore) ListTokens(ctx context.Context, offset, limit int) ([]RawToken, error) {
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

func (st *qfsStore) toRawTokens() RawTokens {
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

func (st *qfsStore) save(ctx context.Context) error {
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

func jwtSigningMethod(pk crypto.PrivKey) (jwt.SigningMethod, error) {
	keyType := pk.Type().String()
	switch keyType {
	case "RSA":
		return jwt.GetSigningMethod("RS256"), nil
	case "Ed25519":
		return jwt.GetSigningMethod("EdDSA"), nil
	default:
		return nil, fmt.Errorf("unsupported key type for token creation: %q", keyType)
	}
}
