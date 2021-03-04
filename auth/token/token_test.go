package token_test

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/auth/key"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/auth/token"
	token_spec "github.com/qri-io/qri/auth/token/spec"
	"github.com/qri-io/qri/profile"
)

func TestPrivKeyTokens(t *testing.T) {
	prevTs := token.Timestamp
	token.Timestamp = func() time.Time { return time.Time{} }
	defer func() { token.Timestamp = prevTs }()

	kd := testkeys.GetKeyData(0)
	tokens, err := token.NewPrivKeySource(kd.PrivKey)
	if err != nil {
		t.Fatal(err)
	}

	pro := &profile.Profile{
		ID:       profile.IDB58MustDecode(kd.EncodedPeerID),
		Peername: "doug",
	}

	tokenString, err := tokens.CreateToken(pro, 0)
	if err != nil {
		t.Fatal(err)
	}

	expect := `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJRbWVMMm1kVmthMWVhaEtFTmplaEs2dEJ4a2twazVkTlExcU1jZ1dpN0hyYjRCIiwicHJvZmlsZUlEIjoiUW1lTDJtZFZrYTFlYWhLRU5qZWhLNnRCeGtrcGs1ZE5RMXFNY2dXaTdIcmI0QiJ9.GG4qKXSUPCFS0a_xuU8NZcRyCPTKvZIObwZQY5bhwnS9hJaxekHOGfIrRsps2tMJJPK4dUSML7dkOs_norVcuhZ4fcmVcJDT_Jel-5DwgxojLS-7ci-tO7NyU1urv7TlfNCUBWiAIoUGj9mkXZYfxVNA0GSssBvKkK4gHbONqHyLc2afkox07-vVOXdwHtVMBMIN-sQGsMHuVze8UJPJRrL2LTRVaYKRaKYwLrt1IG2fFCIpt6xNG93DVkaFV8CezHHXp9rsGtx6FcZUxyONyhTNROQRcJ756DQDLcOup3w435oWzwdanQ-wqGAwhJuy49Pbf2s3ysujMxxITWya4g`
	if expect != tokenString {
		t.Errorf("token mismatch. expected: %q.\ngot: %q", expect, tokenString)
	}

	tokenWithExpiryString, err := tokens.CreateToken(pro, time.Hour)
	expect = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOi02MjEzNTU5MzIwMCwic3ViIjoiUW1lTDJtZFZrYTFlYWhLRU5qZWhLNnRCeGtrcGs1ZE5RMXFNY2dXaTdIcmI0QiIsInByb2ZpbGVJRCI6IlFtZUwybWRWa2ExZWFoS0VOamVoSzZ0Qnhra3BrNWROUTFxTWNnV2k3SHJiNEIifQ.JCiCgabd3cx8yoZcxD-N6ajyoLJ8wpZJjJ6EwWrP1QPvC9_CMRchxtMSLh0iudHLUIv8iFOykcjTCOtK2Mo9QlAF2k1EkV6Bvarxg-BaFhvU1cI1dll5tbDvDs5RVDWi7nSlGEe5nsQwjJXPVZjCKtVR2l-4_iI8FKDUdKI92TJUWiAJ7M1wuK4Do0mtkJxwzjCU_B_9Dxq4qvptAGTAydSQS6z3MPYOXa_I6x9MlRw6vVx6wMoU6Z3NH_pvctLVSvmDyZjst1kZxl__FBAqqjwRfjtijaO9dEDPcHbpN0f26e_MswOJDtPtD2_Yke5GpwfbeC-aUwaWtAvxnCnLqA`
	if expect != tokenWithExpiryString {
		t.Errorf("token mismatch. expected: %q.\ngot: %q", expect, tokenWithExpiryString)
	}

	token_spec.AssertTokenSourceSpec(t, func(ctx context.Context) token.Source {
		source, err := token.NewPrivKeySource(kd.PrivKey)
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

func TestNewPrivKeyAuthToken(t *testing.T) {
	// create a token from a private key
	kd := testkeys.GetKeyData(0)
	str, err := token.NewPrivKeyAuthToken(kd.PrivKey, 0)
	if err != nil {
		t.Fatal(err)
	}

	// prove we can parse a token with a store that only has a public key
	ks, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := ks.AddPubKey(kd.KeyID, kd.PrivKey.GetPublic()); err != nil {
		t.Fatal(err)
	}

	_, err = token.ParseAuthToken(str, ks)
	if err != nil {
		t.Fatal(err)
	}
}
