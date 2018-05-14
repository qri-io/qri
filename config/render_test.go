package config

import (
	"reflect"
	"testing"
)

func TestRenderValidate(t *testing.T) {
	err := DefaultRender().Validate()
	if err != nil {
		t.Errorf("error validating default render: %s", err)
	}
}

func TestRenderCopy(t *testing.T) {
	cases := []struct {
		render *Render
	}{
		{DefaultRender()},
	}
	for i, c := range cases {
		cpy := c.render.Copy()
		if !reflect.DeepEqual(cpy, c.render) {
			t.Errorf("Render Copy test case %v, render structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.render)
			continue
		}
		cpy.DefaultTemplateHash = "foo"
		if reflect.DeepEqual(cpy, c.render) {
			t.Errorf("Render Copy test case %v, editing one render struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.render)
			continue
		}
	}
}
