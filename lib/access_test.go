package lib

import (
	"context"
	"testing"

	"github.com/qri-io/qri/auth/token"
)

func TestAccessCreateAuthToken(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inst, cleanup := NewMemTestInstance(ctx, t)
	defer cleanup()

	// create an authentication token using the owner profile
	p := &CreateAuthTokenParams{
		GranteeUsername: inst.cfg.Profile.Peername,
	}
	s, err := inst.Access().CreateAuthToken(ctx, p)
	if err != nil {
		t.Fatal(err)
	}

	// prove we can parse & validate that token
	_, err = token.ParseAuthToken(s, inst.keystore)
	if err != nil {
		t.Fatal(err)
	}
}
