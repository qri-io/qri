package config

import (
	"reflect"
	"testing"
)

func TestRemoteServerValidate(t *testing.T) {
	rem := &RemoteServer{}
	err := rem.Validate()
	if err != nil {
		t.Errorf("error validating remote: %s", err)
	}
}

func TestRemoteServerCopy(t *testing.T) {
	cases := []struct {
		remote *RemoteServer
	}{
		{&RemoteServer{}},
	}
	for i, c := range cases {
		cpy := c.remote.Copy()
		if !reflect.DeepEqual(cpy, c.remote) {
			t.Errorf("RemoteServer Copy test case %v, remote structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.remote)
			continue
		}
		cpy.Enabled = true
		if reflect.DeepEqual(cpy, c.remote) {
			t.Errorf("RemoteServer Copy test case %v, editing one remote struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.remote)
			continue
		}
	}
}
