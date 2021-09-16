package config_test

import (
	"reflect"
	"testing"

	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
)

func TestP2PValidate(t *testing.T) {
	err := testcfg.DefaultP2PForTesting().Validate()
	if err != nil {
		t.Errorf("error validating default p2p: %s", err)
	}
}

func TestP2PCopy(t *testing.T) {
	cases := []struct {
		p2p *config.P2P
	}{
		{testcfg.DefaultP2PForTesting()},
	}
	for i, c := range cases {
		cpy := c.p2p.Copy()
		if !reflect.DeepEqual(cpy, c.p2p) {
			t.Errorf("P2P Copy test case %v, p2p structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.p2p)
			continue
		}
		cpy.QriBootstrapAddrs[0] = ""
		if reflect.DeepEqual(cpy, c.p2p) {
			t.Errorf("P2P Copy test case %v, editing one p2p struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.p2p)
			continue
		}
	}
}
