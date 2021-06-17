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

	expect := `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJRbWVMMm1kVmthMWVhaEtFTmplaEs2dEJ4a2twazVkTlExcU1jZ1dpN0hyYjRCIiwic3ViIjoiUW1lTDJtZFZrYTFlYWhLRU5qZWhLNnRCeGtrcGs1ZE5RMXFNY2dXaTdIcmI0QiIsImNsaWVudFR5cGUiOiJ1c2VyIn0.gHsjIzo3FAumMKt4epHYyw7AbhGRh-jTJbp661Uxobb23kaMX_CEWb5ZWnt5Bll8-bn0JhELtcDkirDM30Jc-E3qpwned1-NdwAO2jUIpeA-accrM-q2AFRTPVN3r8uBqLFbU7AXcF9m_HKvywTb99KySHqS1hx9MlmlFd3S5EC58aCdL7fYG38m1Rmzrksd9v196BG1VnYFD8Mg8tee06tE7u_8MDhDZ_CsBRXpoilYadEIg1PR1ElcVR7OQGA1yjw2cUibG1K8fAcfVjOC0aecbVFLbxs9wPbiqXdgOkJVpTKTFxdqE2CfI7fEeMLOcS9R5oJSDxftMS2vtl4HbA`
	if expect != tokenString {
		t.Errorf("token mismatch. expected: %q.\ngot: %q", expect, tokenString)
	}

	tokenWithExpiryString, err := tokens.CreateToken(pro, time.Hour)
	expect = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOi02MjEzNTU5MzIwMCwiaXNzIjoiUW1lTDJtZFZrYTFlYWhLRU5qZWhLNnRCeGtrcGs1ZE5RMXFNY2dXaTdIcmI0QiIsInN1YiI6IlFtZUwybWRWa2ExZWFoS0VOamVoSzZ0Qnhra3BrNWROUTFxTWNnV2k3SHJiNEIiLCJjbGllbnRUeXBlIjoidXNlciJ9.IRzNQWFVUZ4VxpcUFmcwRTfMpl85AVIlmdDkGb8qixYaO7gXBSGKL4_XG5yKjqQS5VTSWI1fuYeihs9CbK8sx8luUJXTzTatDNQFAMO4L-76QG7fsZgcaLF4ATGQ0HrxHmDoYlJkRB2tg8TlQ0XlyyRyDWrLxuMtmAUeoSUA97nU7F2IcvloPZEKsV9DU-vxTSLNzV29PhI2oSHhOWHPST9g7gQ39ZH1SX05iNgE-SD55IPZDTgkGZwbeACBrINfZIrAC5_UmCJ-YaUhiPRmCwGY0NoDZTQxPhhc6fJRd8Dgu9gdC1E5-ccOcCADSbXzdcBpoELXguwPlpjRvcqZuw`
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
	str, err := token.NewPrivKeyAuthToken(kd.PrivKey, kd.KeyID.String(), 0)
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
