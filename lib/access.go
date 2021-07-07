package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/profile"
)

// AccessMethods is a group of methods for access control & user authentication
type AccessMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m AccessMethods) Name() string {
	return "access"
}

// Attributes defines attributes for each method
func (m AccessMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"createauthtoken": {Endpoint: AECreateAuthToken, HTTPVerb: "POST", DefaultSource: "local"},
	}
}

// CreateAuthTokenParams are input parameters for Access().CreateAuthToken
type CreateAuthTokenParams struct {
	// username to grant auth; e.g. "keyboard_cat"
	GranteeUsername string `json:"granteeUsername"`
	// profile Identifier to grant token for; e.g. "QmemJQrK7PTQvD3n8gmo9JhyaByyLmETiNR1Y8wS7hv4sP"
	GranteeProfileID string `json:"granteeProfileID"`
	// lifespan of token in nanoseconds; e.g. 2000000000000
	TTL time.Duration `json:"ttl"`
}

// SetNonZeroDefaults uses default token time-to-live if one isn't set
func (p *CreateAuthTokenParams) SetNonZeroDefaults() {
	if p.TTL == 0 {
		p.TTL = token.DefaultTokenTTL
	}
}

// Validate returns an error if input params are invalid
func (p *CreateAuthTokenParams) Validate() error {
	if p.GranteeUsername == "" && p.GranteeProfileID == "" {
		return fmt.Errorf("either grantee username or profile is required")
	}
	return nil
}

// CreateAuthToken constructs a JWT string token suitable for making OAuth
// requests as the grantee user. Creating an access token requires a stored
// private key for the grantee.
// Callers can provide either GranteeUsername OR GranteeProfileID
func (m AccessMethods) CreateAuthToken(ctx context.Context, p *CreateAuthTokenParams) (string, error) {
	res, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "createauthtoken"), p)
	if s, ok := res.(string); ok {
		return s, err
	}
	return "", err
}

// accessImpl is the backing implementation for AccessMethods
type accessImpl struct{}

func (accessImpl) CreateAuthToken(scp scope, p *CreateAuthTokenParams) (string, error) {
	var (
		grantee *profile.Profile
		err     error
	)

	if p.GranteeProfileID != "" {
		id, err := profile.IDB58Decode(p.GranteeProfileID)
		if err != nil {
			return "", err
		}
		if grantee, err = scp.Profiles().GetProfile(id); err != nil {
			return "", err
		}
	} else if p.GranteeUsername == "me" {
		// TODO(b5): this should be scp.ActiveUser()
		grantee = scp.Profiles().Owner()
	} else {
		if grantee, err = profile.ResolveUsername(scp.Profiles(), p.GranteeUsername); err != nil {
			return "", err
		}
	}

	pk := grantee.PrivKey
	if pk == nil {
		return "", fmt.Errorf("cannot create token for %q (id: %s), private key is required", grantee.Peername, grantee.ID.String())
	}

	return token.NewPrivKeyAuthToken(pk, grantee.ID.String(), p.TTL)
}
