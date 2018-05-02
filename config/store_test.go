package config

import (
	"reflect"
	"testing"
)

func TestStoreValidate(t *testing.T) {
	err := DefaultStore().Validate()
	if err != nil {
		t.Errorf("error validating default store: %s", err)
	}
}

func TestStoreCopy(t *testing.T) {
	cases := []struct {
		store *Store
	}{
		{DefaultStore()},
	}
	for i, c := range cases {
		cpy := c.store.Copy()
		if !reflect.DeepEqual(cpy, c.store) {
			t.Errorf("Store Copy test case %v, store structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.store)
			continue
		}
		cpy.Type = ""
		if reflect.DeepEqual(cpy, c.store) {
			t.Errorf("Store Copy test case %v, editing one store struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.store)
			continue
		}
	}
}
