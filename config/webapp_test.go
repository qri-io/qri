package config

import (
	"reflect"
	"testing"
)

func TestWebappValidate(t *testing.T) {
	err := DefaultWebapp().Validate()
	if err != nil {
		t.Errorf("error validating default webapp: %s", err)
	}
}

func TestWebappCopy(t *testing.T) {
	cases := []struct {
		webapp *Webapp
	}{
		{DefaultWebapp()},
	}
	for i, c := range cases {
		cpy := c.webapp.Copy()
		if !reflect.DeepEqual(cpy, c.webapp) {
			t.Errorf("Webapp Copy test case %v, webapp structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.webapp)
			continue
		}
		cpy.Port = 0
		if reflect.DeepEqual(cpy, c.webapp) {
			t.Errorf("Webapp Copy test case %v, editing one webapp struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.webapp)
			continue
		}
	}
}
