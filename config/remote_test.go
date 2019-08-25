package config

import (
	"reflect"
	"testing"
)

func TestRemoteValidate(t *testing.T) {
	rem := &Remote{}
	err := rem.Validate()
	if err != nil {
		t.Errorf("error validating default remote: %s", err)
	}
}

func TestRemoteCopy(t *testing.T) {
	cases := []struct {
		remote *Remote
	}{
		{&Remote{}},
	}
	for i, c := range cases {
		cpy := c.remote.Copy()
		if !reflect.DeepEqual(cpy, c.remote) {
			t.Errorf("Remote Copy test case %v, remote structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.remote)
			continue
		}
		cpy.Enabled = false
		if reflect.DeepEqual(cpy, c.remote) {
			t.Errorf("Remote Copy test case %v, editing one remote struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.remote)
			continue
		}
	}
}
