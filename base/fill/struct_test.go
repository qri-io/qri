package fill

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"gopkg.in/yaml.v2"
)

func TestStruct(t *testing.T) {
	jsonData := `{
  "Name": "test_name",
  "ProfileID": "test_profile_id",
  "Qri": "qri:0"
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var ds dataset.Dataset
	err = Struct(data, &ds)
	if err != nil {
		panic(err)
	}

	if ds.Name != "test_name" {
		t.Errorf("expected: ds.Name should be \"test_name\", got: %s", ds.Name)
	}
	if ds.ProfileID != "test_profile_id" {
		t.Errorf("expected: ds.ProfileID should be \"test_profile_id\", got: %s", ds.ProfileID)
	}
	if ds.Qri != "qri:0" {
		t.Errorf("expected: ds.Qri should be \"qri:0\", got: %s", ds.Qri)
	}
}

func TestInvalidTarget(t *testing.T) {
	jsonData := `{
  "Name": "test_name",
  "ProfileID": "test_profile_id",
  "Qri": "qri:0"
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var n int
	err = Struct(data, &n)
	expect := `can only assign fields to a struct`
	if err == nil {
		t.Fatalf("expected: error for unknown field, but no error returned")
	}
	if err.Error() != expect {
		t.Errorf("expected: expect: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

func TestFillCommitTimestamp(t *testing.T) {
	jsonData := `{
  "Name": "test_commit_timestamp",
  "Commit": {
    "Timestamp": "1999-03-31T19:30:00.000Z"
  }
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var ds dataset.Dataset
	err = Struct(data, &ds)
	if err != nil {
		panic(err)
	}

	loc := time.FixedZone("UTC", 0)
	expect := time.Date(1999, 03, 31, 19, 30, 0, 0, loc)

	if ds.Name != "test_commit_timestamp" {
		t.Errorf("expected: ds.Name should be \"test_name\", got: %s", ds.Name)
	}
	if !ds.Commit.Timestamp.Equal(expect) {
		t.Errorf("expected: timestamp expected %s, got: %s", expect, ds.Commit.Timestamp)
	}
}

func TestFillInvalidTimestamp(t *testing.T) {
	jsonData := `{
  "Name": "test_commit_timestamp",
  "Commit": {
    "Timestamp": "1999-03__1T19:30:00.000Z"
  }
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var ds dataset.Dataset
	err = Struct(data, &ds)
	expect := `at "Commit.Timestamp": could not parse time: "1999-03__1T19:30:00.000Z"`
	if err == nil {
		t.Fatalf("expected: error for unknown field, but no error returned")
	}
	if err.Error() != expect {
		t.Errorf("expected: expect: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

func TestStructInsensitive(t *testing.T) {
	jsonData := `{
  "name": "test_name",
  "pRoFiLeId": "test_profile_id",
  "QRI": "qri:0"
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var ds dataset.Dataset
	err = Struct(data, &ds)
	if err != nil {
		panic(err)
	}

	if ds.Name != "test_name" {
		t.Errorf("expected: ds.Name should be \"test_name\", got: %s", ds.Name)
	}
	if ds.ProfileID != "test_profile_id" {
		t.Errorf("expected: ds.ProfileID should be \"test_profile_id\", got: %s", ds.ProfileID)
	}
	if ds.Qri != "qri:0" {
		t.Errorf("expected: ds.Qri should be \"qri:0\", got: %s", ds.Qri)
	}
}

func TestStructBlankValue(t *testing.T) {
	jsonData := `{
  "Name": "test_name",
  "ProfileID": null,
  "Qri": "qri:0"
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var ds dataset.Dataset
	err = Struct(data, &ds)
	if err != nil {
		panic(err)
	}

	if ds.Name != "test_name" {
		t.Errorf("expected: ds.Name should be \"test_name\", got: %s", ds.Name)
	}
	if ds.ProfileID != "" {
		t.Errorf("expected: ds.ProfileID should be \"\", got: %s", ds.ProfileID)
	}
	if ds.Qri != "qri:0" {
		t.Errorf("expected: ds.Qri should be \"qri:0\", got: %s", ds.Qri)
	}
}

func TestStructUnknownFields(t *testing.T) {
	jsonData := `{
  "Name": "test_name",
  "ProfileID": "test_profile_id",
  "Qri": "qri:0",
  "Unknown": "value"
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var ds dataset.Dataset
	err = Struct(data, &ds)
	if err == nil {
		t.Errorf("expected: error for unknown field, but no error returned")
	}

	expect := `at "Unknown": not found in struct dataset.Dataset`
	if err.Error() != expect {
		t.Errorf("expected: expect: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

type IgnoreStruct struct {
	A string
	B string
}

func (*IgnoreStruct) IgnoreFillField(field string) bool {
	if field == "a" {
		return false
	}
	return true
}

func TestFieldIgnorer(t *testing.T) {
	jsonData := `{
  "a": "test_name",
  "b": "test_profile_id",
  "Qri": "qri:0",
  "Unknown": "value"
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var got IgnoreStruct
	if err = Struct(data, &got); err != nil {
		t.Fatal(err)
	}

	expect := IgnoreStruct{
		A: "test_name",
		// field B must be ignored
	}

	if diff := cmp.Diff(expect, expect); diff != "" {
		t.Errorf("result mistmatch (-want +got):\n%s", diff)
	}
}

func TestStructYaml(t *testing.T) {
	yamlData := `name: test_name
profileID: test_profile_id
qri: qri:0
`

	data := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(yamlData), &data)
	if err != nil {
		panic(err)
	}

	var ds dataset.Dataset
	err = Struct(data, &ds)
	if err != nil {
		panic(err)
	}

	if ds.Name != "test_name" {
		t.Errorf("expected: ds.Name should be \"test_name\", got: %s", ds.Name)
	}
	if ds.ProfileID != "test_profile_id" {
		t.Errorf("expected: ds.ProfileID should be \"test_profile_id\", got: %s", ds.ProfileID)
	}
	if ds.Qri != "qri:0" {
		t.Errorf("expected: ds.Qri should be \"qri:0\", got: %s", ds.Qri)
	}
}

type Collection struct {
	More map[string]interface{}
	Name string
	Age  int
	IsOn bool
	Xpos float64
	Ptr  *int
	Dict map[string]string
	List []string
	Blob []byte
	Sub  SubElement
	Big  int64
	Ubig uint64
	Pair [2]int
	Cat  string `json:"kitten"`
	Dog  string `json:"puppy,omitempty"`
}

type SubElement struct {
	Num    int
	Things *map[string]string
	Any    interface{}
}

func (c *Collection) SetArbitrary(key string, val interface{}) error {
	if c.More == nil {
		c.More = make(map[string]interface{})
	}
	c.More[key] = val
	return nil
}

func TestFillArbitrarySetter(t *testing.T) {
	jsonData := `{
  "Name": "Alice",
  "Age": 42,
  "Unknown": "value"
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Name != "Alice" {
		t.Errorf("expected: c.Name should be \"Alice\", got: %s", c.Name)
	}
	if c.Age != 42 {
		t.Errorf("expected: c.Age should be 42, got: %d", c.Age)
	}
	if c.More["Unknown"] != "value" {
		t.Errorf("expected: c.More[\"Unknown\"] should be \"value\", got: %s", c.More["Unknown"])
	}
}

func TestFillBoolean(t *testing.T) {
	jsonData := `{
  "Name": "Bob",
  "IsOn": true
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Name != "Bob" {
		t.Errorf("expected: c.Name should be \"Bob\", got: %s", c.Name)
	}
	if c.IsOn != true {
		t.Errorf("expected: c.IsOn should be true, got: %v", c.IsOn)
	}
}

func TestFillFloatingPoint(t *testing.T) {
	jsonData := `{
  "Name": "Carol",
  "Xpos": 6.283
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Name != "Carol" {
		t.Errorf("expected: c.Name should be \"Carol\", got: %s", c.Name)
	}
	if c.Xpos != 6.283 {
		t.Errorf("expected: c.Xpos should be 6.283, got: %v", c.Xpos)
	}
}

func TestFillInt64(t *testing.T) {
	jsonData := `{
  "Big": 1234567890123456789,
  "Ubig": 9934567890123456789
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Big != 1234567890123456768 {
		t.Errorf("expected: c.Big should be 1234567890123456768, got: %d", c.Big)
	}
	if c.Ubig != 9934567890123456512 {
		t.Errorf("expected: c.Ubig should be 9934567890123456512, got: %d", c.Ubig)
	}
}

func TestFillArray(t *testing.T) {
	jsonData := `{
  "Pair": [3,4]
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if len(c.Pair) != 2 {
		t.Errorf("expected: c.Pair should have two elements, got: %d", len(c.Pair))
	}
	if c.Pair[0] != 3 {
		t.Errorf("expected: c.Pair[0] should be 3, got: %d", c.Pair[0])
	}
	if c.Pair[1] != 4 {
		t.Errorf("expected: c.Pair[1] should be 4, got: %d", c.Pair[1])
	}
}

func TestFillArrayLengthError(t *testing.T) {
	jsonData := `{
  "Pair": [3,4,5]
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err == nil {
		t.Errorf("expected: error for wrong length, but no error returned")
	}

	expect := `at "Pair": need array of size 2, got size 3`
	if err.Error() != expect {
		t.Errorf("expected: expect: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

func TestFillMetaKeywords(t *testing.T) {
	jsonData := `{
  "Keywords": [
    "Test0",
    "Test1",
    "Test2"
  ]
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var meta dataset.Meta
	err = Struct(data, &meta)
	if err != nil {
		panic(err)
	}

	expect := []string{"Test0", "Test1", "Test2"}
	if !reflect.DeepEqual(meta.Keywords, expect) {
		t.Errorf("expected: c.Keywords should expect: %s, got: %s", expect, meta.Keywords)
	}
}

func TestFillMetaCitations(t *testing.T) {
	jsonData := `{
  "Citations": [
    {
      "Name": "A website",
      "URL": "http://example.com",
      "Email": "me@example.com"
    }
  ]
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var meta dataset.Meta
	err = Struct(data, &meta)
	if err != nil {
		panic(err)
	}

	expect := dataset.Meta{
		Citations: []*dataset.Citation{
			&dataset.Citation{
				Name:  "A website",
				URL:   "http://example.com",
				Email: "me@example.com",
			},
		},
	}
	if !reflect.DeepEqual(meta, expect) {
		t.Errorf("expected: c.Keywords should expect: %s, got: %s", expect.Keywords, meta.Keywords)
	}
}

func TestFillMapStringToString(t *testing.T) {
	jsonData := `{
  "Dict": {
    "cat": "meow",
    "dog": "bark",
    "eel": "zap"
  }
}`
	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if len(c.Dict) != 3 {
		t.Error("expected 3 elements in Dict")
	}
	if c.Dict["cat"] != "meow" {
		t.Error("expected: Dict[\"cat\"] == \"meow\"")
	}
}

func TestStringSlice(t *testing.T) {
	jsonData := `{
  "List": ["a","b","c"]
}`
	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if len(c.List) != 3 {
		t.Error("expected 3 elements in List")
	}
	if c.List[0] != "a" {
		t.Error("expected: List[0] == \"a\"")
	}
}

func TestByteSlice(t *testing.T) {
	jsonData := `{
  "blob": "abcd"
}`
	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if len(c.Blob) != 4 {
		t.Error("expected size 4 for Blob")
	}
	if bytes.Compare(c.Blob, []byte("abcd")) != 0 {
		t.Error("expected: Blob == \"abcd\"")
	}

	// Binary data may also be assigned from a slice of bytes
	data = map[string]interface{}{
		"blob": []byte{0x01, 0x02, 0x03, 0x04, 0x05},
	}
	c = Collection{}
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if len(c.Blob) != 5 {
		t.Error("expected size 5 for Blob")
	}
	if bytes.Compare(c.Blob, []byte{0x01, 0x02, 0x03, 0x04, 0x05}) != 0 {
		t.Error("expected: Blob == \"\\x01, \\x02, \\x03, \\x04, \\x05\"")
	}

	// Binary data can't be assigned an integer, that's an error
	data = map[string]interface{}{
		"blob": 3456,
	}
	c = Collection{}
	err = Struct(data, &c)
	expect := `at "Blob": need type byte slice, value 3456`
	if err == nil {
		t.Fatalf("expected: error for wrong type, but no error returned")
	}
	if err.Error() != expect {
		t.Errorf("expected: expect: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

func TestNilStringSlice(t *testing.T) {
	jsonData := `{
  "List": null
}`
	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.List != nil {
		t.Error("expected null List")
	}
}

func TestNilMap(t *testing.T) {
	jsonData := `{
  "Dict": null
}`
	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Dict != nil {
		t.Error("expected null Dict")
	}
}

func TestNilPointer(t *testing.T) {
	jsonData := `{
  "Ptr": null
}`
	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Ptr != nil {
		t.Error("expected null Ptr")
	}
}

func TestFillErrorMessageOnWrongType(t *testing.T) {
	jsonData := `{
  "Age": "abc"
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err == nil {
		t.Errorf("expected error, did not get an error")
	}
	expect := `at "Age": need int, got string: "abc"`
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

func TestFillErrorMessageOnWrongSubfield(t *testing.T) {
	jsonData := `{
  "Sub": {
    "Num": false
  }
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err == nil {
		t.Errorf("expected error, did not get an error")
	}
	expect := `at "Sub.Num": need int, got bool: false`
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

func TestFillMulitpleErrors(t *testing.T) {
	jsonData := `{
  "Age": "abc",
  "Sub": {
    "Num": false
  }
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err == nil {
		t.Errorf("expected error, did not get an error")
	}
	expect := `at "Age": need int, got string: "abc"
at "Sub.Num": need int, got bool: false`
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

func TestFillAttrName(t *testing.T) {
	jsonData := `{
  "kitten": "meow",
  "puppy": "bark"
}`

	// json Unmarshal will look at field tags to get json keys
	var c Collection
	err := json.Unmarshal([]byte(jsonData), &c)
	if err != nil {
		panic(err)
	}

	if c.Cat != "meow" {
		t.Errorf("exepcted: c.Cat should be \"meow\", got %s", c.Cat)
	}
	if c.Dog != "bark" {
		t.Errorf("exepcted: c.Dog should be \"bark\", got %s", c.Dog)
	}

	// fill Struct should do the same thing, looking at field tags
	data := make(map[string]interface{})
	err = json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	c = Collection{}
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Cat != "meow" {
		t.Errorf("exepcted: c.Cat should be \"meow\", got %s", c.Cat)
	}
	if c.Dog != "bark" {
		t.Errorf("exepcted: c.Dog should be \"bark\", got %s", c.Dog)
	}
	if len(c.More) != 0 {
		t.Errorf("expected: no unused keys, got %v", c.More)
	}
}

func TestFillSubSection(t *testing.T) {
	jsonData := `{
  "Sub": {
    "Num": 7
  }
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Sub.Num != 7 {
		t.Errorf("expected: c.Sub.Num should be 7, got: %d", c.Sub.Num)
	}
}

func TestFillPointerToMap(t *testing.T) {
	jsonData := `{
  "Things": {
    "a": "apple",
    "b": "banana"
  }
}`

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var s SubElement
	err = Struct(data, &s)
	if err != nil {
		panic(err)
	}

	if s.Things == nil {
		t.Errorf("expected: s.Things should be non-nil")
	}
	if (*s.Things)["a"] != "apple" {
		t.Errorf("expected: s.Things[\"a\"] should be \"apple\"")
	}
	if (*s.Things)["b"] != "banana" {
		t.Errorf("expected: s.Things[\"b\"] should be \"banana\"")
	}
}

func TestFillInterface(t *testing.T) {
	// Any can be a float
	jsonData := `{
  "Any": 123.4
}`
	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	var s SubElement
	err = Struct(data, &s)
	if err != nil {
		panic(err)
	}

	if s.Any != 123.4 {
		t.Errorf("expected: s.Any should be 123, got %v of %s", s.Any, reflect.TypeOf(s.Any))
	}

	// Any can be a string
	jsonData = `{
  "Any": "abc"
}`
	data = make(map[string]interface{})
	err = json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		panic(err)
	}

	s = SubElement{}
	err = Struct(data, &s)
	if err != nil {
		panic(err)
	}

	if s.Any != "abc" {
		t.Errorf("expected: s.Any should be \"abc\", got %v of %s", s.Any, reflect.TypeOf(s.Any))
	}
}

func TestFillYamlFields(t *testing.T) {
	yamlData := `
name: abc
age: 42
xpos: 3.51
big: 1234567890123456
`
	data := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(yamlData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Name != "abc" {
		t.Errorf("expected: c.Name should be \"abc\", got: %s", c.Name)
	}
	if c.Age != 42 {
		t.Errorf("expected: c.Name should be 42, got: %d", c.Age)
	}
	if c.Xpos != 3.51 {
		t.Errorf("expected: c.Xpos should be 3.51, got: %f", c.Xpos)
	}
	if c.Big != 1234567890123456 {
		t.Errorf("expected: c.Big should be 1234567890123456, got: %d", c.Big)
	}
}

func TestFillYamlMap(t *testing.T) {
	yamlData := `
name: more
dict:
  a: apple
  b: banana
sub:
  num: 7
  things:
    c: cat
    d: dog
`
	data := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(yamlData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Name != "more" {
		t.Errorf("expected: c.Name should be \"abc\", got: %s", c.Name)
	}
	if len(c.Dict) != 2 {
		t.Errorf("expected: len(c.Dict) should be 2, got: %d", len(c.Dict))
	}
	if c.Dict["a"] != "apple" {
		t.Errorf("expected: c.Dict[\"a\"] should be \"apple\", got: %s", c.Dict["a"])
	}
	if c.Dict["b"] != "banana" {
		t.Errorf("expected: c.Dict[\"b\"] should be \"banana\", got: %s", c.Dict["b"])
	}

	if c.Sub.Num != 7 {
		t.Errorf("expected: c.Sub.Num should be 7, got: %d", c.Sub.Num)
	}
	if len(*c.Sub.Things) != 2 {
		t.Errorf("expected: len(c.Sub.Things) should be 2, got: %d", len(*c.Sub.Things))
	}
	if (*c.Sub.Things)["c"] != "cat" {
		t.Errorf("expected: c.Sub.Things[\"c\"] should be \"cat\", got: %s", (*c.Sub.Things)["c"])
	}
	if (*c.Sub.Things)["d"] != "dog" {
		t.Errorf("expected: c.Sub.Things[\"d\"] should be \"dog\", got: %s", (*c.Sub.Things)["d"])
	}
}

func TestFillYamlMapsHaveStringKeys(t *testing.T) {
	yamlData := `
name: schema
sub:
  any:
    type: object
    inner:
      e: eel
      f: frog
`
	data := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(yamlData), &data)
	if err != nil {
		panic(err)
	}

	var c Collection
	err = Struct(data, &c)
	if err != nil {
		panic(err)
	}

	if c.Name != "schema" {
		t.Errorf("expected: c.Name should be \"schema\", got: %s", c.Name)
	}
	m, ok := c.Sub.Any.(map[string]interface{})
	if !ok {
		t.Fatalf("expected: c.Sub.Any should be a map[string]interface{}, got: %s",
			reflect.TypeOf(c.Sub.Any))
	}
	if m["type"] != "object" {
		t.Errorf("expected: c.Sub.Any[\"type\"] should be \"object\", got: %d", m["type"])
	}

	// This is asserting that maps within data structures also have string keys, despite
	// the fact that YAML deserialized this, and YAML always uses interface{} keys.
	m, ok = m["inner"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected: Inner should be a map[string]interface{}, got: %s",
			reflect.TypeOf(m["inner"]))
	}
	if m["e"] != "eel" {
		t.Errorf("expected: c.Sub.Any[\"e\"] should be \"eel\", got: %d", m["e"])
	}
	if m["f"] != "frog" {
		t.Errorf("expected: c.Sub.Any[\"f\"] should be \"frog\", got: %d", m["f"])
	}
}
