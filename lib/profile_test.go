package lib

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/profile"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestProfileRequestsGet(t *testing.T) {
	cases := []struct {
		in  bool
		res *profile.Profile
		err string
	}{
		// TODO (b5)- restore or destroy
		// {true, nil, ""},
		// {false, nil, ""},
	}

	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	req := NewProfileRequests(node, config.DefaultConfigForTesting(), "", nil)
	for i, c := range cases {
		got := &config.ProfilePod{}
		err := req.GetProfile(&c.in, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestProfileRequestsSave(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_profile_requests_save")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cases := []struct {
		p   *config.ProfilePod
		res *config.ProfilePod
		err string
	}{
		{nil, nil, "profile required for update"},
		{&config.ProfilePod{}, nil, ""},
		// TODO - moar tests
	}

	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	req := NewProfileRequests(node, config.DefaultConfigForTesting(), filepath.Join(dir, "config.yaml"), nil)
	for i, c := range cases {
		got := &config.ProfilePod{}
		err := req.SaveProfile(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

// func TestSaveProfile(t *testing.T) {
// 	// Mock data for the global Config's Profile, used to create new profile.
// 	// TODO: Remove the randomly built Profile that config.DefaultProfile creates.
// 	mockID := "QmWu3MKx2B1xxphkSNWxw8TYt41HnXD8R85Kt2UzKzpGH9"
// 	mockTime := time.Unix(1234567890, 0)
// 	Config.Profile.ID = mockID
// 	Config.Profile.Created = mockTime
// 	Config.Profile.Updated = mockTime
// 	Config.Profile.Peername = "test_mock_peer_name"

// 	// SaveConfig func replacement, so that it copies the global Config here.
// 	var savedConf config.Config
// 	prevSaver := SaveConfig
// 	SaveConfig = func() error {
// 		savedConf = *Config
// 		return nil
// 	}
// 	defer func() { SaveConfig = prevSaver }()

// 	// ProfilePod filled with test data.
// 	pro := config.ProfilePod{}
// 	pro.Name = "test_name"
// 	pro.Email = "test_email@example.com"
// 	pro.Description = "This is only a test profile"
// 	pro.HomeURL = "http://example.com"
// 	pro.Color = "default"
// 	pro.Twitter = "test_twitter"

// 	// Save the ProfilePod.
// 	mr, err := testrepo.NewTestRepo(nil)
// 	if err != nil {
// 		t.Fatalf("error allocating test repo: %s", err.Error())
// 	}
// 	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}

// 	req := NewProfileRequests(node, nil)
// 	got := config.ProfilePod{}
// 	err = req.SaveProfile(&pro, &got)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Saving adds a private key. Verify that it used to not exist, then copy the key.
// 	if got.PrivKey != "" {
// 		log.Errorf("Returned Profile should not have private key: %v", got.PrivKey)
// 	}
// 	got.PrivKey = savedConf.Profile.PrivKey

// 	savedConf.Profile.Online = savedConf.P2P.Enabled

// 	// Verify that the saved config matches the returned config (after private key is copied).
// 	if !reflect.DeepEqual(*savedConf.Profile, got) {
// 		dmp := diffmatchpatch.New()
// 		saved, _ := json.Marshal(*savedConf.Profile)
// 		g, _ := json.Marshal(got)
// 		diffs := dmp.DiffMain(string(saved), string(g), false)
// 		log.Errorf("Saved Profile does not match returned Profile: %s", dmp.DiffPrettyText(diffs))
// 	}

// 	// Validate that the returned Profile has all the proper individual fields.
// 	nullTime := "0001-01-01 00:00:00 +0000 UTC"
// 	if pro.ID != "" {
// 		log.Errorf("Profile should not have ID, has %v", pro.ID)
// 	}
// 	if got.ID != mockID {
// 		log.Errorf("Got ID %v", got.ID)
// 	}
// 	if pro.Created.String() != nullTime {
// 		log.Errorf("Profile should not have Created, has %v", pro.Created)
// 	}
// 	if got.Created != mockTime {
// 		log.Errorf("Got Created %v", got.Created)
// 	}
// 	if pro.Updated.String() != nullTime {
// 		log.Errorf("Profile should not have Updated, has %v", pro.Updated)
// 	}
// 	if got.Updated != mockTime {
// 		log.Errorf("Got Updated %v", got.Updated)
// 	}
// 	if got.Type != "peer" {
// 		log.Errorf("Got Type %v", got.Type)
// 	}
// 	if got.Peername != "test_mock_peer_name" {
// 		log.Errorf("Got Peername %v", got.Peername)
// 	}
// 	if got.Name != "test_name" {
// 		log.Errorf("Got Name %v", got.Name)
// 	}
// 	if got.Email != "test_email@example.com" {
// 		log.Errorf("Got Email %v", got.Email)
// 	}
// 	if got.Description != "This is only a test profile" {
// 		log.Errorf("Got Description %v", got.Description)
// 	}
// 	if got.HomeURL != "http://example.com" {
// 		log.Errorf("Got Type %v", got.HomeURL)
// 	}
// 	if got.Color != "default" {
// 		log.Errorf("Got Type %v", got.Color)
// 	}
// 	if got.Twitter != "test_twitter" {
// 		log.Errorf("Got Type %v", got.Twitter)
// 	}
// }

// func TestProfileRequestsSetPeername(t *testing.T) {
// 	prevSC := SaveConfig
// 	SaveConfig = func() error { return nil }
// 	defer func() {
// 		SaveConfig = prevSC
// 	}()

// 	reg := regmock.NewMemRegistry()
// 	regCli, _ := regmock.NewMockServerRegistry(reg)
// 	n := newTestQriNodeRegClient(t, regCli)

// 	pr := NewProfileRequests(n, nil)

// 	pro, err := n.Repo.Profile()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	pro.Peername = "keyboard_cat"
// 	pp, err := pro.Encode()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	res := &config.ProfilePod{}
// 	if err := pr.SaveProfile(pp, res); err != nil {
// 		t.Error(err)
// 	}

// 	if reg.Profiles.Len() != 1 {
// 		t.Errorf("expected a profile to be in the registry. got: %d", reg.Profiles.Len())
// 	}

// 	reg.Profiles.SortedRange(func(key string, pro *registry.Profile) bool {
// 		if pro.Handle != "keyboard_cat" {
// 			t.Errorf("expected handle to be %s. got: %s", "keyboard_cat", pro.Handle)
// 		}
// 		return true
// 	})
// }

// func TestProfileRequestsSetProfilePhoto(t *testing.T) {
// 	prev := SaveConfig
// 	SaveConfig = func() error {
// 		return nil
// 	}
// 	defer func() { SaveConfig = prev }()

// 	cases := []struct {
// 		infile  string
// 		respath string
// 		err     string
// 	}{
// 		{"", "", "file is required"},
// 		{"testdata/ink_big_photo.jpg", "", "file size too large. max size is 250kb"},
// 		{"testdata/q_bang.svg", "", "invalid file format. only .jpg images allowed"},
// 		{"testdata/rico_400x400.jpg", "/map/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", ""},
// 	}

// 	mr, err := testrepo.NewTestRepo(nil)
// 	if err != nil {
// 		t.Fatalf("error allocating test repo: %s", err.Error())
// 	}
// 	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}

// 	req := NewProfileRequests(node, nil)
// 	for i, c := range cases {
// 		p := &FileParams{}
// 		if c.infile != "" {
// 			p.Filename = filepath.Base(c.infile)
// 			r, err := os.Open(c.infile)
// 			if err != nil {
// 				t.Errorf("case %d error opening test file %s: %s ", i, c.infile, err.Error())
// 				continue
// 			}
// 			p.Data = r
// 		}

// 		res := &config.ProfilePod{}
// 		err := req.SetProfilePhoto(p, res)
// 		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
// 			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err.Error())
// 			continue
// 		}

// 		if c.respath != res.Photo {
// 			t.Errorf("case %d profile hash mismatch. expected: %s, got: %s", i, c.respath, res.Photo)
// 			continue
// 		}
// 	}
// }

// func TestProfileRequestsSetPosterPhoto(t *testing.T) {
// 	prev := SaveConfig
// 	SaveConfig = func() error {
// 		return nil
// 	}
// 	defer func() { SaveConfig = prev }()

// 	cases := []struct {
// 		infile  string
// 		respath string
// 		err     string
// 	}{
// 		{"", "", "file is required"},
// 		{"testdata/ink_big_photo.jpg", "", "file size too large. max size is 250kb"},
// 		{"testdata/q_bang.svg", "", "invalid file format. only .jpg images allowed"},
// 		{"testdata/rico_poster_1500x500.jpg", "/map/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", ""},
// 	}

// 	mr, err := testrepo.NewTestRepo(nil)
// 	if err != nil {
// 		t.Fatalf("error allocating test repo: %s", err.Error())
// 	}
// 	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}

// 	req := NewProfileRequests(node, nil)
// 	for i, c := range cases {
// 		p := &FileParams{}
// 		if c.infile != "" {
// 			p.Filename = filepath.Base(c.infile)
// 			r, err := os.Open(c.infile)
// 			if err != nil {
// 				t.Errorf("case %d error opening test file %s: %s ", i, c.infile, err.Error())
// 				continue
// 			}
// 			p.Data = r
// 		}

// 		res := &config.ProfilePod{}
// 		err := req.SetProfilePhoto(p, res)
// 		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
// 			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err.Error())
// 			continue
// 		}

// 		if c.respath != res.Photo {
// 			t.Errorf("case %d profile hash mismatch. expected: %s, got: %s", i, c.respath, res.Photo)
// 			continue
// 		}
// 	}
// }
