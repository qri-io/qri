package config

import (
	"reflect"
	"testing"
)

func TestRegistryValidate(t *testing.T) {
	err := DefaultRegistry().Validate()
	if err != nil {
		t.Errorf("error validating default registry: %s", err)
	}
}

func TestRegistryCopy(t *testing.T) {
	cases := []struct {
		registry *Registry
	}{
		{DefaultRegistry()},
	}
	for i, c := range cases {
		cpy := c.registry.Copy()
		if !reflect.DeepEqual(cpy, c.registry) {
			t.Errorf("Registry Copy test case %v, registry structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.registry)
			continue
		}
		cpy.Location = "different/location"
		if reflect.DeepEqual(cpy, c.registry) {
			t.Errorf("Registry Copy test case %v, editing one registry struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.registry)
			continue
		}
	}
}
