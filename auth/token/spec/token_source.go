package spec

import (
	"context"
	"testing"

	"github.com/dgrijalva/jwt-go"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/profile"
)

// AssertTokenSourceSpec ensures a TokenSource implementation behaves as
// expected
func AssertTokenSourceSpec(t *testing.T, newTokenSource func(ctx context.Context) token.Source) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := newTokenSource(ctx)

	p1 := &profile.Profile{
		ID:       profile.IDB58DecodeOrEmpty(testkeys.GetKeyData(1).EncodedPeerID),
		Peername: "username",
	}

	raw, err := source.CreateToken(p1, 0)
	if err != nil {
		t.Errorf("source should allow creating key with valid profile & zero duration. got: %q", err)
	}

	p := &jwt.Parser{
		UseJSONNumber:        true,
		SkipClaimsValidation: false,
	}
	if _, _, err := p.ParseUnverified(raw, &token.Claims{}); err != nil {
		t.Errorf("created token must parse with token.Claims. got: %q", err)
	}

	if _, err := token.Parse(raw, source); err != nil {
		t.Errorf("source must create tokens that parse with it's own verification keys. error: %q", err)
	}
}
