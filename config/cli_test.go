package config

import (
	"reflect"
	"testing"
)

func TestCLIValidate(t *testing.T) {
	err := DefaultCLI().Validate()
	if err != nil {
		t.Errorf("error validating default cli: %s", err)
	}
}

func TestCLICopy(t *testing.T) {
	cases := []struct {
		cli *CLI
	}{
		{DefaultCLI()},
	}
	for i, c := range cases {
		cpy := c.cli.Copy()
		if !reflect.DeepEqual(cpy, c.cli) {
			t.Errorf("CLI Copy test case %v, cli structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.cli)
			continue
		}
	}
}
