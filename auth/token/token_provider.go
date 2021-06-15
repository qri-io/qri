package token

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/profile"
)

const (
	// RTCode signals the token response type is 'code'
	RTCode ResponseType = "code"
	// RTToken signals the token response type is 'token'
	RTToken ResponseType = "token"

	// AccessTokenTTL is the lifespan of an access token
	AccessTokenTTL = time.Hour * 2
	// RefreshTokenTTL is the lifespan of a refresh token
	RefreshTokenTTL = time.Hour * 24 * 30
)

var (
	// ErrInvalidRequest is returned on any parse or void output error
	ErrInvalidRequest = fmt.Errorf("invalid request")
	// ErrInvalidCredentials signals a bad username/password/key error
	ErrInvalidCredentials = fmt.Errorf("invalid user credentials")
	// ErrNotFound is returned when no matching results exist for the provided credentials
	ErrNotFound = fmt.Errorf("user not found")
)

// Provider is a service that generates access & refresh tokens
type Provider interface {
	// Token handles the auth token flow
	Token(ctx context.Context, req *Request) (*Response, error)
}

// Request is a wrapper for incoming token requests
type Request struct {
	GrantType    GrantType `json:"grant_type"`
	Code         string    `json:"code"`
	Username     string    `json:"username"`
	Password     string    `json:"password"`
	RefreshToken string    `json:"refresh_token"`
	RedirectURI  string    `json:"redirect_uri"`
}

// Response wraps the token response object
type Response struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// ResponseType the type of authorization request
type ResponseType string

func (rt ResponseType) String() string {
	return string(rt)
}

// GrantType authorization model
type GrantType string

// define authorization model
const (
	AuthorizationCode   GrantType = "authorization_code"
	PasswordCredentials GrantType = "password"
	ClientCredentials   GrantType = "client_credentials"
	Refreshing          GrantType = "refresh_token"
	Implicit            GrantType = "__implicit"
)

func (gt GrantType) String() string {
	if gt == AuthorizationCode ||
		gt == PasswordCredentials ||
		gt == ClientCredentials ||
		gt == Refreshing {
		return string(gt)
	}
	return ""
}

// LocalProvider implements the Provider interface and
// provides mechanics for generating tokens for a selected profile
type LocalProvider struct {
	profiles profile.Store
	keys     key.Store
}

// NewProvider instantiates a new LocalProvider
func NewProvider(p profile.Store, k key.Store) (*LocalProvider, error) {
	return &LocalProvider{
		profiles: p,
		keys:     k,
	}, nil
}

// compile-time assertion that LocalProvider is a token.Provider
var _ Provider = (*LocalProvider)(nil)

// Token handles the OAuth token flow
func (p *LocalProvider) Token(ctx context.Context, req *Request) (*Response, error) {
	log.Debugf("token.Provider got request: %+v", req)
	resp := &Response{TokenType: "jwt", ExpiresIn: int64(AccessTokenTTL.Seconds())}
	switch req.GrantType {
	case PasswordCredentials:
		if req.Username == "" {
			return nil, ErrInvalidCredentials
		}
		// TODO(arqu): this only selects the first returned profile for a given peername.
		// ideally we would use the profile.ID to fetch the exact profile
		// or otherwise validate the signatures
		pros, err := p.profiles.ProfilesForUsername(req.Username)
		if err != nil {
			log.Debugf("token.Provider failed to fetch profiles: %q", err.Error())
			return nil, ErrInvalidRequest
		}
		if len(pros) == 0 {
			log.Debugf("token.Provider no matching profiles found")
			return nil, ErrNotFound
		}
		if len(pros) > 1 {
			log.Infof("token.Provider found multiple profiles for the given username - selected the first one")
		}
		pro := pros[0]
		if pro.PrivKey == nil {
			log.Debugf("token.Provider private key is nil")
			return nil, ErrInvalidCredentials
		}
		accessToken, err := NewPrivKeyAuthToken(pro.PrivKey, pro.ID.String(), AccessTokenTTL)
		if err != nil {
			log.Debugf("token.Provider failed to generate access token: %q", err.Error())
			return nil, ErrInvalidRequest
		}
		refreshToken, err := NewPrivKeyAuthToken(pro.PrivKey, pro.ID.String(), RefreshTokenTTL)
		if err != nil {
			log.Debugf("token.Provider failed to generate refresh token: %q", err.Error())
			return nil, ErrInvalidRequest
		}
		resp.AccessToken = accessToken
		resp.RefreshToken = refreshToken
	case Refreshing:
		tok, err := ParseAuthToken(req.RefreshToken, p.keys)
		if err != nil {
			log.Debugf("token.Provider error parsing refresh token: %q", err.Error())
			return nil, ErrInvalidRequest
		}

		if claims, ok := tok.Claims.(*Claims); ok {
			pid, err := profile.IDB58Decode(claims.ProfileID)
			if err != nil {
				log.Debugf("token.Provider failed to parse profileID")
				return nil, ErrInvalidRequest
			}
			pro, err := p.profiles.GetProfile(pid)
			if errors.Is(err, profile.ErrNotFound) {
				log.Debugf("token.Provider profile not found")
				return nil, ErrNotFound
			}
			accessToken, err := NewPrivKeyAuthToken(pro.PrivKey, pro.ID.String(), AccessTokenTTL)
			if err != nil {
				log.Debugf("token.Provider failed to generate access token: %q", err.Error())
				return nil, ErrInvalidRequest
			}
			resp.AccessToken = accessToken
		}
	default:
		return nil, ErrInvalidRequest
	}
	return resp, nil
}
