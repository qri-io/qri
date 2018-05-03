package config

import (
	"reflect"
	"testing"
)

func TestLoggingValidate(t *testing.T) {
	err := DefaultLogging().Validate()
	if err != nil {
		t.Errorf("error validating default logging: %s", err)
	}
}

func TestLoggingCopy(t *testing.T) {
	cases := []struct {
		logging *Logging
	}{
		{DefaultLogging()},
	}
	for i, c := range cases {
		cpy := c.logging.Copy()
		if !reflect.DeepEqual(cpy, c.logging) {
			t.Errorf("Logging Copy test case %v, logging structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.logging)
			continue
		}
		cpy.Levels["qriapi"] = "error"
		if reflect.DeepEqual(cpy, c.logging) {
			t.Errorf("Logging Copy test case %v, editing one logging struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.logging)
			continue
		}
	}
}
