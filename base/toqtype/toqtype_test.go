package toqtype

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type simpleObject struct {
	Fruit string
	Num   int
}

type invalidObject struct {
	Bad func()
	Num int
}

func TestStructToMap(t *testing.T) {
	s := simpleObject{Fruit: "apple", Num: 4}
	val, err := StructToMap(s)
	if err != nil {
		t.Fatal(err)
	}
	expect := map[string]interface{}{
		"Fruit": "apple",
		"Num":   4.0,
	}
	if diff := cmp.Diff(expect, val); diff != "" {
		t.Errorf("StructToMap (-want +got):\n%s", diff)
	}
}

func TestStructToMapInvalid(t *testing.T) {
	s := invalidObject{Bad: func() {}, Num: 4}
	_, err := StructToMap(s)
	if err == nil {
		t.Fatal("expected error, did not get one")
	}
	expect := "json: unsupported type: func()"
	if err.Error() != expect {
		t.Fatalf("error mismatch, expected: %s, got: %s", expect, err.Error())
	}
}

func TestMustParseJSONAsArray(t *testing.T) {
	val := MustParseJSONAsArray("[1,2,3]")
	expect := []interface{}{1.0, 2.0, 3.0}
	if diff := cmp.Diff(expect, val); diff != "" {
		t.Errorf("MustParseJsonAsArray (-want +got):\n%s", diff)
	}
}

func TestMustParseCsvAsArray(t *testing.T) {
	val := MustParseCsvAsArray("1,2\n3,4\n")
	expect := []interface{}{
		[]string{"1", "2"},
		[]string{"3", "4"},
	}
	if diff := cmp.Diff(expect, val); diff != "" {
		t.Errorf("MustParseCsvAsArray (-want +got):\n%s", diff)
	}
}
