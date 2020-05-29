package config

import (
	"reflect"
	"testing"
)

func TestFilesystemValidate(t *testing.T) {
	err := DefaultFilesystemLocal().Validate()
	if err != nil {
		t.Errorf("error validating default store: %s", err)
	}
}

func TestFilesystemCopy(t *testing.T) {
	cases := []struct {
		filesystem *Filesystem
	}{
		{DefaultFilesystemLocal()},
	}
	for i, c := range cases {
		cpy := c.filesystem.Copy()
		if !reflect.DeepEqual(cpy, c.filesystem) {
			t.Errorf("Filesystem Copy test case %v, filesystem structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.filesystem)
			continue
		}
		cpy.Type = "test"
		if reflect.DeepEqual(cpy, c.filesystem) {
			t.Errorf("Filesystem Copy test case %v, editing one filesystem struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.filesystem)
			continue
		}
	}
}
