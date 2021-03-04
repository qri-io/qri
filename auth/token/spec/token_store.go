package spec

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/profile"
)

// AssertTokenStoreSpec ensures an token.TokenStore implementation behaves as
// expected
func AssertTokenStoreSpec(t *testing.T, newTokenStore func(context.Context) token.Store) {
	prevTs := token.Timestamp
	token.Timestamp = func() time.Time { return time.Time{} }
	defer func() { token.Timestamp = prevTs }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pk := testkeys.GetKeyData(0).PrivKey
	tokens, err := token.NewPrivKeySource(pk)
	if err != nil {
		t.Fatalf("creating local tokens: %q", err)
	}
	store := newTokenStore(ctx)

	results, err := store.ListTokens(ctx, 0, -1)
	if err != nil {
		t.Errorf("listing all tokens of an empty store shouldn't error. got: %q ", err)
	}
	if len(results) > 0 {
		t.Errorf("new store should return no results. got: %d", len(results))
	}

	_, err = store.RawToken(ctx, "this doesn't exist")
	if !errors.Is(err, token.ErrTokenNotFound) {
		t.Errorf("expected store.RawToken(nonexistent key) to return a wrap of token.ErrTokenNotFound. got: %q", err)
	}
	err = store.DeleteToken(ctx, "this also doesn't exist")
	if !errors.Is(err, token.ErrTokenNotFound) {
		t.Errorf("expected store.D key to return a wrap of token.ErrTokenNotFound. got: %q", err)
	}
	if err := store.PutToken(ctx, "_bad_key", "not.a.key"); err == nil {
		t.Errorf("putting an invalid json web token should error. got nil")
	}

	p1 := &profile.Profile{
		ID:       profile.IDB58DecodeOrEmpty(testkeys.GetKeyData(1).EncodedPeerID),
		Peername: "local_user",
	}
	t1Raw, err := tokens.CreateToken(p1, 0)
	if err != nil {
		t.Fatalf("creating token: %q", err)
	}

	if err := store.PutToken(ctx, "_root", t1Raw); err != nil {
		t.Errorf("putting root key shouldn't error. got: %q", err)
	}

	results, err = store.ListTokens(ctx, 0, -1)
	if err != nil {
		t.Errorf("listing all tokens of an empty store shouldn't error. got: %q ", err)
	}
	if len(results) != 1 {
		t.Errorf("result length mismatch listing keys after adding `root` key. expected 1, got: %d", len(results))
	}

	p2 := &profile.Profile{
		ID:       profile.IDB58DecodeOrEmpty(testkeys.GetKeyData(2).EncodedPeerID),
		Peername: "user_2",
	}
	t2Raw, err := tokens.CreateToken(p2, time.Millisecond*10)
	if err != nil {
		t.Fatalf("creating token: %q", err)
	}

	secondKey := "http://registry.qri.cloud"
	if err := store.PutToken(ctx, secondKey, t2Raw); err != nil {
		t.Errorf("putting a second token with key=%q shouldn't error. got: %q", secondKey, err)
	}

	results, err = store.ListTokens(ctx, 0, -1)
	if err != nil {
		t.Errorf("listing all tokens of an empty store shouldn't error. got: %q ", err)
	}
	if len(results) != 2 {
		t.Errorf("result length mismatch listing keys after adding second key. expected 2, got: %d", len(results))
	}

	expect := []token.RawToken{
		{
			Key: "_root",
			Raw: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJRbVdZZ0Q0OXI5SG51WEVwcFFFcTFhN1NVVXJ5amE0UU5zOUU2WENIMlBheUNEIiwicHJvZmlsZUlEIjoiUW1XWWdENDlyOUhudVhFcHBRRXExYTdTVVVyeWphNFFOczlFNlhDSDJQYXlDRCJ9.ZGWi-Ek60jlARIQFEBx0QnFpq0zrdlXKUOhdJtcyZohaKyeXF4lWnFVlnIFCSa-SKJFMnrdECygVFVAHOyj2wj-lXFSAsLIgZrLcS2kWp4alvc9K448PTy89zcdxPMhhwbR89cc7-5ndOjaxCL1JyiihmeIJbMAJ94z3QaFnhkWrcSUwxYalgYDdFaQChQRtSP454wtC2elXbDXSaJJpOgG0LU_tHun0lUgLHbYI4fG3Xoz-t9ZzuAXSgOKzduKVLRsSvoVmL81wXL2QRnMAnijovglBdkWxDmt7ae005ymd6wPMY-3Oy_2nmTbhSWrx-rcyA3Ce26gGKMEuP-EP8w",
		},
		{
			Key: secondKey,
			Raw: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOi02MjEzNTU5NjgwMCwic3ViIjoiUW1QZUZUTkhjWkRyM1pGRWZGZmVweFM1UHFIQW1mQlJHUU5QSjM4OUN3aDFhcyIsInByb2ZpbGVJRCI6IlFtUGVGVE5IY1pEcjNaRkVmRmZlcHhTNVBxSEFtZkJSR1FOUEozODlDd2gxYXMifQ.YakUofrw4mY5kEx8DB2CtBWBiXB6LdNfFd2uvYHRMm2jAS-3GOFy13hOHVhWKDidL4ve3n2bz1mMnSlHpf67gvep1cG8AsAfFaZEJdJPaeovkOvS21y_63o9BVNfpkRkh9QCd65QePHMJfyfwgUAZq2-vLTloiEjB0MX2oUmnF-bIWTUdJ9dsrS8MyzR4CcvPDx-J63n94w6MzK8ddVlUiFRFHx5jkuVfXLrSTaSvQPOq4SUj9sa8kmlk1pazIlTBIUH6raLoDZLxR6nY7Z8Uz-sS4P7bsRkv0647rPva8CmzdC-9SltNNfLC-pxFH1_sGO_fmr30fKyd2-YxOddaQ",
		},
	}

	if diff := cmp.Diff(expect, results); diff != "" {
		t.Log(results)
		t.Errorf("mistmatched list keys results. (-want +got):\n%s", diff)
	}

	results, err = store.ListTokens(ctx, 1, 1)
	if err != nil {
		t.Errorf("listing all tokens of an empty store shouldn't error. got: %q ", err)
	}
	if len(results) != 1 {
		t.Errorf("result length mismatch listing keys after adding `root` key. expected 1, got: %d", len(results))
	}

	if diff := cmp.Diff(expect[1:], results); diff != "" {
		t.Errorf("mistmatched list keys with offset=1, limit=1. results. (-want +got):\n%s", diff)
	}

	if err := store.DeleteToken(ctx, secondKey); err != nil {
		t.Errorf("store.DeleteToken shouldn't error for existing key. got: %q", err)
	}

	_, err = store.RawToken(ctx, secondKey)
	if !errors.Is(err, token.ErrTokenNotFound) {
		t.Errorf("store.RawToken() for a just-deleted key must return a wrap of token.ErrTokenNotFound. got: %q", err)
	}
}
