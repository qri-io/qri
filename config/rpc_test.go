package config

import (
	"reflect"
	"testing"
)

func TestRPCValidate(t *testing.T) {
	err := DefaultRPC().Validate()
	if err != nil {
		t.Errorf("error validating default rpc: %s", err)
	}
}

func TestRPCCopy(t *testing.T) {
	cases := []struct {
		rpc *RPC
	}{
		{DefaultRPC()},
	}
	for i, c := range cases {
		cpy := c.rpc.Copy()
		if !reflect.DeepEqual(cpy, c.rpc) {
			t.Errorf("RPC Copy test case %v, rpc structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.rpc)
			continue
		}
		cpy.Enabled = false
		if reflect.DeepEqual(cpy, c.rpc) {
			t.Errorf("RPC Copy test case %v, editing one rpc struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.rpc)
			continue
		}
	}
}
