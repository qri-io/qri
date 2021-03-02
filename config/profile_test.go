package config_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
)

func TestProfileValidate(t *testing.T) {
	err := testcfg.DefaultProfileForTesting().Validate()
	if err != nil {
		t.Errorf("error validating default profile: %s", err)
	}
}

func TestProfileCopy(t *testing.T) {
	cases := []struct {
		profile *config.ProfilePod
	}{
		{&config.ProfilePod{Type: "peer"}},
		{&config.ProfilePod{Name: "user", Color: "blue"}},
		{&config.ProfilePod{Type: "peer", Name: "user", Color: "blue"}},
	}
	for i, c := range cases {
		cpy := c.profile.Copy()
		if !reflect.DeepEqual(cpy, c.profile) {
			t.Errorf("ProfilePod Copy test case %v, profile structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.profile)
			continue
		}
	}
}

func TestProfileCopyPeerIDs(t *testing.T) {
	// build off DefaultProfile so we can test that the profile Copy
	// actually copies over correctly (ie, deeply)
	p := testcfg.DefaultProfileForTesting()
	p.PeerIDs = []string{"1", "2", "3"}

	cpy := p.Copy()
	if !reflect.DeepEqual(cpy, p) {
		t.Errorf("ProfilePodCopyPeerIDs, structs are not equal: \ncopy: %v, \noriginal: %v", cpy, p)
	}
	cpy.PeerIDs[0] = ""
	if reflect.DeepEqual(cpy, p) {
		t.Errorf("ProfilePodCopyPeerIDs, editing one profile struct should not affect the other: \ncopy: %v, \noriginal: %v", cpy, p)
	}
}

func TestProfileSetField(t *testing.T) {
	p := config.ProfilePod{
		Created: time.Now(),
		Updated: time.Now(),
		Type:    "peer",
	}
	p.SetField("email", "user@example.com")

	expect := config.ProfilePod{
		Created: p.Created,
		Updated: p.Updated,
		Type:    "peer",
		Email:   "user@example.com",
	}
	if !reflect.DeepEqual(p, expect) {
		t.Errorf("ProfilePod SetField email, structs are not equal: \nactual: %v, \nexpect: %v", p, expect)
	}

	p.SetField("name", "user")
	expect = config.ProfilePod{
		Created: p.Created,
		Updated: p.Updated,
		Type:    "peer",
		Email:   "user@example.com",
		Name:    "user",
	}
	if !reflect.DeepEqual(p, expect) {
		t.Errorf("ProfilePod SetField name, structs are not equal: \nactual: %v, \nexpect: %v", p, expect)
	}

	p.SetField("email", "me@example.com")
	expect = config.ProfilePod{
		Created: p.Created,
		Updated: p.Updated,
		Type:    "peer",
		Email:   "me@example.com",
		Name:    "user",
	}
	if !reflect.DeepEqual(p, expect) {
		t.Errorf("ProfilePod SetField email again, structs are not equal: \nactual: %v, \nexpect: %v", p, expect)
	}
}

func TestBadPeername(t *testing.T) {
	pro := config.ProfilePod{
		Type:     "peer",
		Peername: "bad=name",
	}

	err := pro.Validate()
	if err == nil {
		t.Fatal("expected error, did not get one")
	}
	expectErr := dsref.ErrDescribeValidUsername.Error()
	if err.Error() != expectErr {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err.Error())
	}
}
