package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qri-io/qri/auth/key"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/profile"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestAuthAndGetProfile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tr, err := repotest.NewTempRepo("test_auth_profile", "TestAuthAndGetProfile", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Delete()

	kd0 := testkeys.GetKeyData(0)
	kd1 := testkeys.GetKeyData(1)

	ks, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(os.TempDir(), "profile_keys")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating tmp directory: %s", err.Error())
	}

	owner := &profile.Profile{
		ID:       profile.IDFromPeerID(kd0.PeerID),
		Peername: "user",
		PrivKey:  kd0.PrivKey,
	}
	ps, err := profile.NewLocalStore(ctx, filepath.Join(path, "profiles.json"), owner, ks)
	if err != nil {
		t.Fatal(err)
	}

	pp := &config.ProfilePod{
		ID:       kd0.PeerID.String(),
		Peername: "p0",
		Created:  time.Unix(1234567890, 0).In(time.UTC),
		Updated:  time.Unix(1234567890, 0).In(time.UTC),
	}
	pro, err := profile.NewProfile(pp)
	if err != nil {
		t.Fatal(err)
	}
	pro.PrivKey = kd0.PrivKey
	pro.PubKey = kd0.PrivKey.GetPublic()
	err = ps.PutProfile(ctx, pro)
	if err != nil {
		t.Fatal(err)
	}

	pp = &config.ProfilePod{
		ID:       kd1.PeerID.String(),
		Peername: "p1",
		Created:  time.Unix(1234567890, 0).In(time.UTC),
		Updated:  time.Unix(1234567890, 0).In(time.UTC),
	}
	pro, err = profile.NewProfile(pp)
	if err != nil {
		t.Fatal(err)
	}
	pro.PrivKey = kd1.PrivKey
	pro.PubKey = kd1.PrivKey.GetPublic()
	err = ps.PutProfile(ctx, pro)
	if err != nil {
		t.Fatal(err)
	}

	inst, err := lib.NewInstance(ctx, tr.QriPath,
		lib.OptKeyStore(ks),
		lib.OptProfiles(ps),
	)
	if err != nil {
		t.Fatal(err)
	}

	tok0, err := inst.TokenProvider().Token(ctx, &token.Request{GrantType: token.PasswordCredentials, Username: "p0"})
	if err != nil {
		t.Fatal(err)
	}

	tok1, err := inst.TokenProvider().Token(ctx, &token.Request{GrantType: token.PasswordCredentials, Username: "p1"})
	if err != nil {
		t.Fatal(err)
	}

	ctx = token.AddToContext(ctx, tok0.AccessToken)

	res, _, err := inst.Dispatch(ctx, "profile.getprofile", &lib.ProfileParams{})
	if err != nil {
		t.Fatal(err)
	}

	if respro, ok := res.(*config.ProfilePod); ok {
		if respro.Peername != "p0" {
			t.Errorf("wrong profile returned: expected 'p0' got '%s'", respro.Peername)
		}
	} else {
		t.Errorf("failed to get profile")
	}

	ctx = token.AddToContext(ctx, tok1.AccessToken)

	res, _, err = inst.Dispatch(ctx, "profile.getprofile", &lib.ProfileParams{})
	if err != nil {
		t.Fatal(err)
	}

	if respro, ok := res.(*config.ProfilePod); ok {
		if respro.Peername != "p1" {
			t.Errorf("wrong profile returned: expected 'p1' got '%s'", respro.Peername)
		}
	} else {
		t.Errorf("failed to get profile")
	}
}
