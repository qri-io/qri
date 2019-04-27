package config

import (
	"reflect"
	"testing"
)

func TestUpdateValidate(t *testing.T) {
	err := DefaultUpdate().Validate()
	if err != nil {
		t.Errorf("error validating default update: %s", err)
	}
}

func TestUpdateCopy(t *testing.T) {
	cases := []struct {
		rpc *Update
	}{
		{DefaultUpdate()},
	}
	for i, c := range cases {
		cpy := c.rpc.Copy()
		if !reflect.DeepEqual(cpy, c.rpc) {
			t.Errorf("Update Copy test case %v, rpc structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.rpc)
			continue
		}
		cpy.Daemonize = false
		if reflect.DeepEqual(cpy, c.rpc) {
			t.Errorf("Update Copy test case %v, editing one rpc struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.rpc)
			continue
		}
	}
}
