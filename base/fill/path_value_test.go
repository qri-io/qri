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
	err = SetPathValue("ison", "false", &c)
	if err != nil {
		panic(err)
	}
	if c.IsOn {
		t.Errorf("expected: s.IsOn should be false")
	}

	c = Collection{}
	err = SetPathValue("ison", 123, &c)
	expect := `at "ison": need bool, got int: 123`
	if err == nil {
		t.Fatalf("expected: error \"%s\", got no error", expect)
	}
	if err.Error() != expect {
		t.Errorf("expected: error should be \"%s\", got \"%s\"", expect, err.Error())
	}

	c = Collection{}
	err = SetPathValue("ison", "abc", &c)
	expect = `at "ison": need bool, got string: "abc"`
	if err == nil {
		t.Fatalf("expected: error \"%s\", got no error", expect)
	}
	if err.Error() != expect {
		t.Errorf("expected: error should be \"%s\", got \"%s\"", expect, err.Error())
	}

	c = Collection{}
	err = SetPathValue("big", 1234567890123, &c)
	if err != nil {
		panic(err)
	}
	if c.Big != 1234567890123 {
		t.Errorf("expected: s.Big should be 1234567890123")
	}

	err = SetPathValue("big", "2345678901234", &c)
	if err != nil {
		panic(err)
	}
	if c.Big != 2345678901234 {
		t.Errorf("expected: s.Big should be 2345678901234")
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
	expect = `at "not_found": path not found`
	if err == nil {
		t.Fatalf("expected: error \"%s\", got no error", expect)
	}
	if err.Error() != expect {
		t.Errorf("expected: error should be \"%s\", got \"%s\"", expect, err.Error())
	}

	// TODO: path like `list.0` where the index == len(list), should extend the list by 1

	c = Collection{}
	err = SetPathValue("list.2", "abc", &c)
	expect = `at "list.2": index outside of range: 2, len is 0`
	if err == nil {
		t.Fatalf("expected: error \"%s\", got no error", expect)
	}
	if err.Error() != expect {
		t.Errorf("expected: error should be \"%s\", got \"%s\"", expect, err.Error())
	}

	c = Collection{}
	c.List = make([]string, 4)
	err = SetPathValue("list.2", 123, &c)
	expect = `at "list.2": need string, got int: 123`
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

	// Error
	c = Collection{}
	err = SetPathValue("sub.num", "abc", &c)
	expect = `at "sub.num": need int, got string: "abc"`
	if err == nil {
		t.Fatalf("expected: error \"%s\", got no error", expect)
	}
	if err.Error() != expect {
		t.Errorf("expected: error should be \"%s\", got \"%s\"", expect, err.Error())
	}
}

func TestGetPathValue(t *testing.T) {
	c := Collection{
		Name: "Alice",
		Dict: map[string]string{
			"extra": "misc",
		},
		List: []string{"cat", "dog", "eel"},
	}

	val, err := GetPathValue("name", &c)
	if err != nil {
		panic(err)
	}
	if val != "Alice" {
		t.Errorf("expected: val should be \"Alice\"")
	}

	val, err = GetPathValue("dict.extra", &c)
	if err != nil {
		panic(err)
	}
	if val != "misc" {
		t.Errorf("expected: val should be \"misc\"")
	}

	val, err = GetPathValue("not_found", &c)
	expect := `at "not_found": path not found`
	if err == nil {
		t.Fatalf("expected: error \"%s\", got no error", expect)
	}
	if err.Error() != expect {
		t.Errorf("expected: error should be \"%s\", got \"%s\"", expect, err.Error())
	}

	val, err = GetPathValue("dict.missing_key", &c)
	expect = `at "dict.missing_key": invalid path`
	if err == nil {
		t.Fatalf("expected: error \"%s\", got no error", expect)
	}
	if err.Error() != expect {
		t.Errorf("expected: error should be \"%s\", got \"%s\"", expect, err.Error())
	}

	val, err = GetPathValue("list.1", &c)
	if err != nil {
		panic(err)
	}
	if val != "dog" {
		t.Errorf("expected: val should be \"dog\"")
	}

	val, err = GetPathValue("list.invalid", &c)
	expect = `at "list.invalid": need int, got string: "invalid"`
	if err == nil {
		t.Fatalf("expected: error \"%s\", got no error", expect)
	}
	if err.Error() != expect {
		t.Errorf("expected: error should be \"%s\", got \"%s\"", expect, err.Error())
	}

}
