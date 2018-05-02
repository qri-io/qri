package config

import (
	"reflect"
	"testing"
)

func TestProfileValidate(t *testing.T) {
	err := DefaultProfile().Validate()
	if err != nil {
		t.Errorf("error validating default profile: %s", err)
	}
}

func TestProfileCopy(t *testing.T) {
	// build off DefaultProfile so we can test that the profile Copy
	// actually copies over correctly (ie, deeply)
	p := DefaultProfile()
	p.Addresses = map[string][]string{}
	p.Addresses["test"] = []string{"1", "2", "3"}

	cases := []struct {
		profile *ProfilePod
	}{
		{p},
	}
	for i, c := range cases {
		cpy := c.profile.Copy()
		if !reflect.DeepEqual(cpy, c.profile) {
			t.Errorf("ProfilePod Copy test case %v, profile structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.profile)
			continue
		}
		cpy.Addresses["test"][0] = ""
		if reflect.DeepEqual(cpy, c.profile) {
			t.Errorf("ProfilePod Copy test case %v, editing one profile struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.profile)
			continue
		}
	}
}
