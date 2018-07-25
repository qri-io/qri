package config

import (
	"reflect"
	"testing"
	"time"
)

func TestProfileValidate(t *testing.T) {
	err := DefaultProfile().Validate()
	if err != nil {
		t.Errorf("error validating default profile: %s", err)
	}
}

func TestProfileCopy(t *testing.T) {
	cases := []struct {
		profile *ProfilePod
	}{
		{&ProfilePod{Type: "peer"}},
		{&ProfilePod{Name: "user", Color: "blue"}},
		{&ProfilePod{Type: "peer", Name: "user", Color: "blue"}},
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
	p := DefaultProfile()
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
	p := ProfilePod{
		Created: time.Now(),
		Updated: time.Now(),
		Type:    "peer",
	}
	p.SetField("email", "user@example.com")

	expect := ProfilePod{
		Created: p.Created,
		Updated: p.Updated,
		Type:    "peer",
		Email:   "user@example.com",
	}
	if !reflect.DeepEqual(p, expect) {
		t.Errorf("ProfilePod SetField email, structs are not equal: \nactual: %v, \nexpect: %v", p, expect)
	}

	p.SetField("name", "user")
	expect = ProfilePod{
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
	expect = ProfilePod{
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
