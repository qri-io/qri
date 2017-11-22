package dataset

import (
	"github.com/qri-io/compare"
	"testing"
)

func TestSchemaFieldNames(t *testing.T) {
	cases := []struct {
		Schema *Schema
		names  []string
	}{
		{AirportCodesStructure.Schema, []string{"ident", "type", "name", "latitude_deg", "longitude_deg", "elevation_ft", "continent", "iso_country", "iso_region", "municipality", "gps_code", "iata_code", "local_code"}},
	}

	for i, c := range cases {
		got := c.Schema.FieldNames()
		if err := compare.StringSlice(c.names, got); err != nil {
			t.Errorf("case %d error: %s", i, err.Error())
			continue
		}
	}
}

func TestSchemaFieldForName(t *testing.T) {
	cases := []struct {
		schema *Schema
		name   string
		field  *Field
	}{
		{AirportCodesStructure.Schema, "name", AirportCodesStructure.Schema.Fields[2]},
	}

	for i, c := range cases {
		got := c.schema.FieldForName(c.name)
		if err := CompareFields(c.field, got); err != nil {
			t.Errorf("case %d error: %s", i, err.Error())
			continue
		}
	}
}
