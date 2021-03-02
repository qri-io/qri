package lib

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/registry"
	regmock "github.com/qri-io/qri/registry/regserver"
	testrepo "github.com/qri-io/qri/repo/test"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestProfileRequestsGet(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cases := []struct {
		in  bool
		res *profile.Profile
		err string
	}{
		// {true, nil, ""},
		// {false, nil, ""},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}

	cfg := testcfg.DefaultConfigForTesting()
	node, err := p2p.NewQriNode(mr, cfg.P2P, event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	// TODO (b5) - hack until tests have better instance-generation primitives
	inst := NewInstanceFromConfigAndNode(ctx, cfg, node)
	m := NewProfileMethods(inst)

	for i, c := range cases {
		got := &config.ProfilePod{}
		err := m.GetProfile(ctx, &c.in, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestProfileRequestsSave(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cfg := testcfg.DefaultConfigForTesting()

	cases := []struct {
		p   *config.ProfilePod
		res *config.ProfilePod
		err string
	}{
		{nil, nil, "profile required for update"},
		{&config.ProfilePod{}, nil, ""},
		// TODO - moar tests
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, cfg.P2P, event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	// TODO (b5) - hack until tests have better instance-generation primitives
	inst := NewInstanceFromConfigAndNode(ctx, cfg, node)
	m := NewProfileMethods(inst)

	for i, c := range cases {
		got := &config.ProfilePod{}
		err := m.SaveProfile(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestSaveProfile(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cfg := testcfg.DefaultConfigForTesting()

	// Mock data for the global Config's Profile, used to create new profile.
	// TODO: Remove the randomly built Profile that config.DefaultProfile creates.
	mockID := "QmWu3MKx2B1xxphkSNWxw8TYt41HnXD8R85Kt2UzKzpGH9"
	mockTime := time.Unix(1234567890, 0)
	cfg.Profile.ID = mockID
	cfg.Profile.Created = mockTime
	cfg.Profile.Updated = mockTime
	cfg.Profile.Peername = "test_mock_peer_name"

	// ProfilePod filled with test data.
	pro := config.ProfilePod{}
	pro.Name = "test_name"
	pro.Email = "test_email@example.com"
	pro.Description = "This is only a test profile"
	pro.HomeURL = "http://example.com"
	pro.Color = "default"
	pro.Twitter = "test_twitter"

	// Save the ProfilePod.
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, cfg.P2P, event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	// TODO (b5) - hack until tests have better instance-generation primitives
	inst := NewInstanceFromConfigAndNode(ctx, cfg, node)
	m := NewProfileMethods(inst)

	got := config.ProfilePod{}
	err = m.SaveProfile(&pro, &got)
	if err != nil {
		log.Fatal(err)
	}

	// Saving adds a private key. Verify that it used to not exist, then copy the key.
	if got.PrivKey != "" {
		log.Errorf("Returned Profile should not have private key: %v", got.PrivKey)
	}
	got.PrivKey = cfg.Profile.PrivKey

	cfg.Profile.Online = cfg.P2P.Enabled

	// Verify that the saved config matches the returned config (after private key is copied).
	if !reflect.DeepEqual(*cfg.Profile, got) {
		dmp := diffmatchpatch.New()
		saved, _ := json.Marshal(*cfg.Profile)
		g, _ := json.Marshal(got)
		diffs := dmp.DiffMain(string(saved), string(g), false)
		log.Errorf("Saved Profile does not match returned Profile: %s", dmp.DiffPrettyText(diffs))
	}

	// Validate that the returned Profile has all the proper individual fields.
	nullTime := "0001-01-01 00:00:00 +0000 UTC"
	if pro.ID != "" {
		log.Errorf("Profile should not have ID, has %v", pro.ID)
	}
	if got.ID != mockID {
		log.Errorf("Got ID %v", got.ID)
	}
	if pro.Created.String() != nullTime {
		log.Errorf("Profile should not have Created, has %v", pro.Created)
	}
	if got.Created != mockTime {
		log.Errorf("Got Created %v", got.Created)
	}
	if pro.Updated.String() != nullTime {
		log.Errorf("Profile should not have Updated, has %v", pro.Updated)
	}
	if got.Updated != mockTime {
		log.Errorf("Got Updated %v", got.Updated)
	}
	if got.Type != "peer" {
		log.Errorf("Got Type %v", got.Type)
	}
	if got.Peername != "test_mock_peer_name" {
		log.Errorf("Got Peername %v", got.Peername)
	}
	if got.Name != "test_name" {
		log.Errorf("Got Name %v", got.Name)
	}
	if got.Email != "test_email@example.com" {
		log.Errorf("Got Email %v", got.Email)
	}
	if got.Description != "This is only a test profile" {
		log.Errorf("Got Description %v", got.Description)
	}
	if got.HomeURL != "http://example.com" {
		log.Errorf("Got Type %v", got.HomeURL)
	}
	if got.Color != "default" {
		log.Errorf("Got Type %v", got.Color)
	}
	if got.Twitter != "test_twitter" {
		log.Errorf("Got Type %v", got.Twitter)
	}
}

func TestProfileRequestsSetPeername(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cfg := testcfg.DefaultConfigForTesting()

	reg := regmock.NewMemRegistry(nil)
	node := newTestQriNode(t)

	// TODO (b5) - hack until tests have better instance-generation primitives
	inst := NewInstanceFromConfigAndNode(ctx, cfg, node)

	regCli, _ := regmock.NewMockServerRegistry(reg)
	inst.registry = regCli

	m := NewProfileMethods(inst)

	pro := node.Repo.Profiles().Owner()
	pro.Peername = "keyboard_cat"
	pp, err := pro.Encode()
	if err != nil {
		t.Fatal(err)
	}

	res := &config.ProfilePod{}
	if err := m.SaveProfile(pp, res); err != nil {
		t.Error(err)
	}

	length, err := reg.Profiles.Len()
	if err != nil {
		t.Fatal(err)
	}
	if length != 1 {
		t.Errorf("expected a profile to be in the registry. got: %d", length)
	}

	reg.Profiles.SortedRange(func(key string, pro *registry.Profile) (bool, error) {
		if pro.Username != "keyboard_cat" {
			t.Errorf("expected handle to be %s. got: %s", "keyboard_cat", pro.Username)
		}
		return false, nil
	})
}

func TestProfileRequestsSetProfilePhoto(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cfg := testcfg.DefaultConfigForTesting()

	cases := []struct {
		infile  string
		respath string
		err     string
	}{
		{"", "", "file is required"},
		{"testdata/ink_big_photo.jpg", "", "file size too large. max size is 250kb"},
		{"testdata/q_bang.svg", "", "invalid file format. only .jpg images allowed"},
		{"testdata/rico_400x400.jpg", "/mem/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", ""},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, cfg.P2P, event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	// TODO (b5) - hack until tests have better instance-generation primitives
	inst := NewInstanceFromConfigAndNode(ctx, cfg, node)
	m := NewProfileMethods(inst)

	for i, c := range cases {
		p := &FileParams{}
		if c.infile != "" {
			p.Filename = filepath.Base(c.infile)
			r, err := os.Open(c.infile)
			if err != nil {
				t.Errorf("case %d error opening test file %s: %s ", i, c.infile, err.Error())
				continue
			}
			p.Data = r
		}

		res := &config.ProfilePod{}
		err := m.SetProfilePhoto(p, res)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err.Error())
			continue
		}

		if c.respath != res.Photo {
			t.Errorf("case %d profile hash mismatch. expected: %s, got: %s", i, c.respath, res.Photo)
			continue
		}
	}
}

func TestProfileRequestsSetPosterPhoto(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cfg := testcfg.DefaultConfigForTesting()

	cases := []struct {
		infile  string
		respath string
		err     string
	}{
		{"", "", "file is required"},
		{"testdata/ink_big_photo.jpg", "", "file size too large. max size is 250kb"},
		{"testdata/q_bang.svg", "", "invalid file format. only .jpg images allowed"},
		{"testdata/rico_poster_1500x500.jpg", "/mem/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", ""},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, cfg.P2P, event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	// TODO (b5) - hack until tests have better instance-generation primitives
	inst := NewInstanceFromConfigAndNode(ctx, cfg, node)
	m := NewProfileMethods(inst)

	for i, c := range cases {
		p := &FileParams{}
		if c.infile != "" {
			p.Filename = filepath.Base(c.infile)
			r, err := os.Open(c.infile)
			if err != nil {
				t.Errorf("case %d error opening test file %s: %s ", i, c.infile, err.Error())
				continue
			}
			p.Data = r
		}

		res := &config.ProfilePod{}
		err := m.SetProfilePhoto(p, res)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err.Error())
			continue
		}

		if c.respath != res.Photo {
			t.Errorf("case %d profile hash mismatch. expected: %s, got: %s", i, c.respath, res.Photo)
			continue
		}
	}
}
