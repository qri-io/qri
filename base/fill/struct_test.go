package fill

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

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
		t.Errorf("expected: ds.ProfileID should be \"test_profile_id\", got: %s", ds.Name)
	}
	if ds.Qri != "qri:0" {
		t.Errorf("expected: ds.Qri should be \"qri:0\", got: %s", ds.Qri)
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
		t.Errorf("expected: ds.ProfileID should be \"test_profile_id\", got: %s", ds.Name)
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

	expect := "field \"Unknown\" not found in target"
	if err.Error() != expect {
		t.Errorf("expected: expect: \"%s\", got: \"%s\"", expect, err.Error())
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
		t.Errorf("expected: ds.ProfileID should be \"test_profile_id\", got: %s", ds.Name)
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
	Sub  SubElement
}

type SubElement struct {
	Num    int
	Things *map[string]string
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
		t.Errorf("expected: c.Keywords should expect: %s, got: %s", expect, meta.Keywords)
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
