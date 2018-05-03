package config

import (
	"reflect"
	"testing"
)

func TestAPIValidate(t *testing.T) {
	err := DefaultAPI().Validate()
	if err != nil {
		t.Errorf("error validating default api: %s", err)
	}
}

func TestAPICopy(t *testing.T) {
	cases := []struct {
		api *API
	}{
		{DefaultAPI()},
	}
	for i, c := range cases {
		cpy := c.api.Copy()
		if !reflect.DeepEqual(cpy, c.api) {
			t.Errorf("API Copy test case %v, api structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.api)
			continue
		}
		cpy.AllowedOrigins[0] = ""
		if reflect.DeepEqual(cpy, c.api) {
			t.Errorf("API Copy test case %v, editing one api struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.api)
			continue
		}
	}
}
