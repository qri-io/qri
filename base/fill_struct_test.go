package base

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"gopkg.in/yaml.v2"
)

func TestFillStruct(t *testing.T) {
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
	err = FillStruct(data, &ds)
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
	err = FillStruct(data, &ds)
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

func TestFillStructInsensitive(t *testing.T) {
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
	err = FillStruct(data, &ds)
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

func TestFillStructUnknownFields(t *testing.T) {
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
	err = FillStruct(data, &ds)
	if err == nil {
		t.Errorf("expected: error for unknown field, but no error returned")
	}

	expect := "field \"Unknown\" not found in target"
	if err.Error() != expect {
		t.Errorf("expected: expect: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

func TestFillStructYaml(t *testing.T) {
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
	err = FillStruct(data, &ds)
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
}

func (c *Collection) SetKeyVal(key string, val interface{}) error {
	if c.More == nil {
		c.More = make(map[string]interface{})
	}
	c.More[key] = val
	return nil
}

func TestFillCollection(t *testing.T) {
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
	err = FillStruct(data, &c)
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
