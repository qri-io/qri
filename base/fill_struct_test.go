package base

import (
	"encoding/json"
	"testing"

	"github.com/qri-io/dataset"
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

// TODO (dlong): More tests
