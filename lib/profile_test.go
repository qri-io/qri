package lib

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/registry"
	regmock "github.com/qri-io/qri/registry/regserver"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestProfileRequestsGet(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cases := []struct {
		res *profile.Profile
		err string
	}{
		// {nil, ""},
		// {nil, ""},
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
	m := inst.Profile()

	for i, c := range cases {
		_, err := m.GetProfile(ctx, &ProfileParams{})

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
		p   *SetProfileParams
		res *config.ProfilePod
		err string
	}{
		{nil, nil, ErrDispatchNilParam.Error()},
		{&SetProfileParams{Pro: &config.ProfilePod{}}, nil, ""},
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
	m := inst.Profile()

	for i, c := range cases {
		_, err := m.SetProfile(ctx, c.p)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestSetProfile(t *testing.T) {
	tr := newTestRunner(t)
	defer tr.Delete()

	// ProfilePod filled with test data.
	pro := config.ProfilePod{}
	pro.Name = "test_name"
	pro.Email = "test_email@example.com"
	pro.Description = "This is only a test profile"
	pro.HomeURL = "http://example.com"
	pro.Color = "default"
	pro.Twitter = "test_twitter"

	// set up expected profile
	expect := tr.Instance.cfg.Profile.Copy()
	expect.Name = pro.Name
	expect.Email = pro.Email
	expect.Description = pro.Description
	expect.HomeURL = pro.HomeURL
	expect.Color = pro.Color
	expect.Twitter = pro.Twitter

	p := &SetProfileParams{Pro: &pro}
	got, err := tr.Instance.Profile().SetProfile(tr.Ctx, p)
	if err != nil {
		t.Fatal(err)
	}

	// Saving adds a private key. Verify that it used to not exist, then copy the key.
	if got.PrivKey != "" {
		t.Errorf("Returned Profile should not have private key: %v", got.PrivKey)
	}

	// Verify that the saved config is as expected
	if diff := cmp.Diff(expect, tr.Instance.cfg.Profile); diff != "" {
		t.Errorf("saved profile mismatch (-want +got):\n%s", diff)
	}

	// Verify that the returned config is as expected
	expect.PrivKey = ""
	expect.Online = tr.Instance.cfg.P2P.Enabled
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("saved profile response mismatch (-want +got):\n%s", diff)
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

	pro := node.Repo.Profiles().Owner()
	pro.Peername = "keyboard_cat"
	pp, err := pro.Encode()
	if err != nil {
		t.Fatal(err)
	}

	param := &SetProfileParams{Pro: pp}
	_, err = inst.Profile().SetProfile(ctx, param)
	if err != nil {
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
	m := inst.Profile()

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

		res, err := m.SetProfilePhoto(ctx, p)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %q, got: %q", i, c.err, err.Error())
			continue
		}

		if res == nil && c.respath != "" {
			t.Errorf("case %d profile hash mismatch. expected: %q, got nil", i, c.respath)
			continue
		}

		if res == nil {
			continue
		}

		if c.respath != res.Photo {
			t.Errorf("case %d profile hash mismatch. expected: %q, got: %q", i, c.respath, res.Photo)
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
		{"testdata/ink_big_photo.jpg", "", "file size too large. max size is 2Mb"},
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
	m := inst.Profile()

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

		res, err := m.SetPosterPhoto(ctx, p)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err.Error())
			continue
		}

		if res == nil && c.respath != "" {
			t.Errorf("case %d profile hash mismatch. expected: %q, got nil", i, c.respath)
			continue
		}

		if res == nil {
			continue
		}

		if c.respath != res.Poster {
			t.Errorf("case %d profile hash mismatch. expected: %s, got: %s", i, c.respath, res.Poster)
			continue
		}
	}
}
