package fill

import (
	"testing"
)

func TestFillPathValue(t *testing.T) {
	c := Collection{}
	err := SetPathValue("name", "Alice", &c)
	if err != nil {
		panic(err)
	}
	if c.Name != "Alice" {
		t.Errorf("expected: s.Name should be \"Alice\"")
	}

	c = Collection{}
	err = SetPathValue("age", 42, &c)
	if err != nil {
		panic(err)
	}
	if c.Age != 42 {
		t.Errorf("expected: s.Age should be 42")
	}

	c = Collection{}
	err = SetPathValue("age", "56", &c)
	if err != nil {
		panic(err)
	}
	if c.Age != 56 {
		t.Errorf("expected: s.Age should be 56")
	}

	c = Collection{}
	err = SetPathValue("ison", "true", &c)
	if err != nil {
		panic(err)
	}
	if !c.IsOn {
		t.Errorf("expected: s.IsOn should be true")
	}

	c = Collection{}
	err = SetPathValue("ison", true, &c)
	if err != nil {
		panic(err)
	}
	if !c.IsOn {
		t.Errorf("expected: s.IsOn should be true")
	}

	c = Collection{}
	err = SetPathValue("ptr", 123, &c)
	if err != nil {
		panic(err)
	}
	if *(c.Ptr) != 123 {
		t.Errorf("expected: s.Ptr should be 123")
	}

	c = Collection{}
	err = SetPathValue("not_found", "missing", &c)
	expect := "path: \"not_found\" not found"
	if err == nil {
		t.Fatalf("expected: error \"%s\", got no error", expect)
	}
	if err.Error() != expect {
		t.Errorf("expected: error should be \"%s\", got \"%s\"", expect, err.Error())
	}

	c = Collection{}
	err = SetPathValue("sub.num", 7, &c)
	if err != nil {
		panic(err)
	}
	if c.Sub.Num != 7 {
		t.Errorf("expected: s.Sub.Num should be 7")
	}

	c = Collection{}
	err = SetPathValue("dict.cat", "meow", &c)
	if err != nil {
		panic(err)
	}
	if c.Dict["cat"] != "meow" {
		t.Errorf("expected: s.Dict[\"cat\"] should be \"meow\"")
	}

	// Don't allocate a new map.
	err = SetPathValue("dict.dog", "bark", &c)
	if err != nil {
		panic(err)
	}
	if c.Dict["cat"] != "meow" {
		t.Errorf("expected: s.Dict[\"cat\"] should be \"meow\"")
	}
	if c.Dict["dog"] != "bark" {
		t.Errorf("expected: s.Dict[\"dog\"] should be \"bark\"")
	}

	s := &SubElement{}
	err = SetPathValue("things.eel", "zap", &s)
	if err != nil {
		panic(err)
	}
	if (*s.Things)["eel"] != "zap" {
		t.Errorf("expected: c.Things[\"eel\"] should be \"zap\"")
	}

	// Don't allocate a new map.
	err = SetPathValue("things.frog", "ribbit", &s)
	if err != nil {
		panic(err)
	}
	if (*s.Things)["eel"] != "zap" {
		t.Errorf("expected: c.Things[\"eel\"] should be \"zap\"")
	}
	if (*s.Things)["frog"] != "ribbit" {
		t.Errorf("expected: c.Things[\"frog\"] should be \"ribbit\"")
	}
}
