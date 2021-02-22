package token_test

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/auth/token"
	token_spec "github.com/qri-io/qri/auth/token/spec"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/profile"
)

func TestPrivKeyTokens(t *testing.T) {
	prevTs := token.Timestamp
	token.Timestamp = func() time.Time { return time.Time{} }
	defer func() { token.Timestamp = prevTs }()

	peerInfo := cfgtest.GetTestPeerInfo(0)
	tokens, err := token.NewPrivKeySource(peerInfo.PrivKey)
	if err != nil {
		t.Fatal(err)
	}

	pro := &profile.Profile{
		ID:       profile.IDB58MustDecode(peerInfo.EncodedPeerID),
		Peername: "doug",
	}

	tokenString, err := tokens.CreateToken(pro, 0)
	if err != nil {
		t.Fatal(err)
	}

	expect := `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJRbWVMMm1kVmthMWVhaEtFTmplaEs2dEJ4a2twazVkTlExcU1jZ1dpN0hyYjRCIiwidXNlcm5hbWUiOiJkb3VnIn0.ZNVGEvqDvCsY1H8dsWJILCIrcOTlLxC_5F-in7jWyfmT4RDatk3-ygVCCH-tYqvXx3dzf-U7qOSR8aR3E5Irvax84WoT0nwR7m51R36WaLPt_dXvtb4jLpjuqUdj5hGdBl2OA-UUuIlI7EzBftlNi6AMDQkcYbX8JWT-Jk47cVxM9f9DWDZphQlgEGm6Czdk5SCfIX1oORkN58zwIaOqP29aba6gzTgl3BMaTAJUkzy-i8dD98xLQXdXIYHxUzsLPAD-WjIEf7lmMetz2ls8okYq8EGyHVYhko_b6t8b5_VZA-GnFnB8D2JkAlcWEIJ_jxuNHHK7g0MTF1GPUT4s1A`
	if expect != tokenString {
		t.Errorf("token mismatch. expected: %q.\ngot: %q", expect, tokenString)
	}

	tokenWithExpiryString, err := tokens.CreateToken(pro, time.Hour)
	expect = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOi02MjEzNTU5MzIwMCwic3ViIjoiUW1lTDJtZFZrYTFlYWhLRU5qZWhLNnRCeGtrcGs1ZE5RMXFNY2dXaTdIcmI0QiIsInVzZXJuYW1lIjoiZG91ZyJ9.d7XPhsj7hkyxg1JzC59hfu90RYem5q6Pie-ofJhdlGk_sY5bH8gcqG90LndMh4_LglEvtrwf_SVFcM1b78qhNon_Yo91kG_K_MmyExa-AlpY65Ji_kpRWcnI8hl-mxrZ2MzxPjvAEOa6c80DUWgTFKlkrgf9RnZlqq-nHnxHHXbVKYI3girsDgWynaIhR53yMBDIhbTCZaQ8XKtU_Pr0L1dJAW7YvOo2H01VM4LI_UQqhCmEbTnQX1Zee0tg88IMzLl7WsdNNOzUsf7dCYWGerLtzxGbxR0wweXbqVJBlzIl0Upke8-FBuZIbcdGSniy4DX643KrNnp_FnzQ8oBHTA`
	if expect != tokenWithExpiryString {
		t.Errorf("token mismatch. expected: %q.\ngot: %q", expect, tokenWithExpiryString)
	}

	token_spec.AssertTokenSourceSpec(t, func(ctx context.Context) token.Source {
		source, err := token.NewPrivKeySource(peerInfo.PrivKey)
		if err != nil {
			panic(err)
		}
		return source
	})
}

func TestTokenStore(t *testing.T) {
	fs := qfs.NewMemFS()

	token_spec.AssertTokenStoreSpec(t, func(ctx context.Context) token.Store {
		ts, err := token.NewStore("tokens.json", fs)
		if err != nil {
			panic(err)
		}
		return ts
	})
}
