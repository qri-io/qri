package dsutil

import (
	"testing"

	"github.com/qri-io/dataset"
)

const yamlData = `---
meta:
  title: EPA TRI Basic Summary
  description: A few key fields pulled from EPA TRI Basic data for 2016
transform:
  config:
    foo: bar
  secrets:
    a: b
    c: d
structure:
  format: json
  schema:
    type: array
    items:
      type: array
      items:
      - title: Year
        maxLength: 4
        type: string
        description: "The Reporting Year - Year the chemical was released or managed as waste"
      - title: "TRI Facility ID"
        maxLength: 15
        type: string
        description: "The TRI Facility Identification Number assigned by EPA/TRI"
      - title: Facility Name
        maxLength: 62
        type: string
        description: "Facility Name"
`

func TestUnmarshalYAMLDataset(t *testing.T) {
	ds := &dataset.Dataset{}
	if err := UnmarshalYAMLDataset([]byte(yamlData), ds); err != nil {
		t.Error(err.Error())
		return
	}

	if ds.Transform.Secrets["a"] != "b" {
		t.Error("expected transform.secrets.a to equal 'b'")
		return
	}
}
