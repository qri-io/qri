package dataset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/datatypes"
)

func TestStrucureHash(t *testing.T) {
	cases := []struct {
		r    *Structure
		hash string
		err  error
	}{
		{&Structure{Format: CsvDataFormat}, "QmQS5d6vtwMCiCgtjS5883oHMK44EMcuCtvyMhDXsha4wo", nil},
	}

	for i, c := range cases {
		hash, err := c.r.Hash()
		if err != c.err {
			t.Errorf("case %d error mismatch. expected %s, got %s", i, c.err, err)
			continue
		}

		if hash != c.hash {
			t.Errorf("case %d hash mismatch. expected %s, got %s", i, c.hash, hash)
			continue
		}
	}
}

func TestAbstractColumnName(t *testing.T) {
	if AbstractColumnName(0) != "a" {
		t.Errorf("expected 0 == a")
	}
	// I found the h button & pushed it twice.
	if AbstractColumnName(215) != "hh" {
		t.Errorf("expected 26 == hh, got: %s", AbstractColumnName(215))
	}
	if AbstractColumnName(30000) != "ariw" {
		t.Errorf("expected 300 == ariw, got: %s", AbstractColumnName(30000))
	}
}

func TestStructureAbstract(t *testing.T) {
	cases := []struct {
		in, out *Structure
	}{
		{AirportCodesStructure, AirportCodesStructureAbstract},
	}

	for i, c := range cases {
		if err := CompareStructures(c.in.Abstract(), c.out); err != nil {
			t.Errorf("case %d error: %s", i, err.Error())
			continue
		}
	}
}

func TestStructureAssign(t *testing.T) {
	expect := &Structure{
		Format: CsvDataFormat,
		Schema: &Schema{
			Fields: []*Field{
				&Field{Type: datatypes.String, Name: "foo"},
				&Field{Type: datatypes.Integer, Name: "bar"},
				&Field{Description: "bat"},
			},
		},
	}
	got := &Structure{
		Format: CsvDataFormat,
		Schema: &Schema{
			Fields: []*Field{
				&Field{Type: datatypes.String},
				&Field{Type: datatypes.Integer},
			},
		},
	}

	got.Assign(&Structure{
		Schema: &Schema{
			Fields: []*Field{
				&Field{Name: "foo"},
				&Field{Name: "bar"},
				&Field{Description: "bat"},
			},
		},
	})

	if err := CompareStructures(expect, got); err != nil {
		t.Error(err)
	}

	got.Assign(nil, nil)
	if err := CompareStructures(expect, got); err != nil {
		t.Error(err)
	}

	emptySt := &Structure{}
	emptySt.Assign(expect)
	if err := CompareStructures(expect, emptySt); err != nil {
		t.Error(err)
	}
}

func TestStructureUnmarshalJSON(t *testing.T) {
	cases := []struct {
		FileName string
		result   *Structure
		err      error
	}{
		{"testdata/structures/airport-codes.json", AirportCodesStructure, nil},
		{"testdata/structures/continent-codes.json", ContinentCodesStructure, nil},
		{"testdata/structures/hours.json", HoursStructure, nil},
	}

	for i, c := range cases {
		data, err := ioutil.ReadFile(c.FileName)
		if err != nil {
			t.Errorf("case %d couldn't read file: %s", i, err.Error())
		}

		ds := &Structure{}
		if err := json.Unmarshal(data, ds); err != c.err {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if err = CompareStructures(ds, c.result); err != nil {
			t.Errorf("case %d resource comparison error: %s", i, err)
			continue
		}
	}

	strq := &Structure{}
	path := "/path/to/structure"
	if err := json.Unmarshal([]byte(`"`+path+`"`), strq); err != nil {
		t.Errorf("unmarshal string path error: %s", err.Error())
		return
	}

	if strq.path.String() != path {
		t.Errorf("unmarshal didn't set proper path: %s != %s", path, strq.path)
		return
	}
}

func TestStructureMarshalJSON(t *testing.T) {
	cases := []struct {
		in  *Structure
		out []byte
		err error
	}{
		{&Structure{Format: CsvDataFormat}, []byte(`{"format":"csv"}`), nil},
		{AirportCodesStructure, []byte(`{"format":"csv","formatConfig":{"headerRow":true},"schema":{"fields":[{"name":"ident","type":"string"},{"name":"type","type":"string"},{"name":"name","type":"string"},{"name":"latitude_deg","type":"float"},{"name":"longitude_deg","type":"float"},{"name":"elevation_ft","type":"integer"},{"name":"continent","type":"string"},{"name":"iso_country","type":"string"},{"name":"iso_region","type":"string"},{"name":"municipality","type":"string"},{"name":"gps_code","type":"string"},{"name":"iata_code","type":"string"},{"name":"local_code","type":"string"}]}}`), nil},
	}

	for i, c := range cases {
		got, err := c.in.MarshalJSON()
		if err != c.err {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if !bytes.Equal(c.out, got) {
			t.Errorf("case %d error mismatch. %s != %s", i, string(c.out), string(got))
			continue
		}
	}

	strbytes, err := json.Marshal(&Structure{path: datastore.NewKey("/path/to/structure")})
	if err != nil {
		t.Errorf("unexpected string marshal error: %s", err.Error())
		return
	}

	if !bytes.Equal(strbytes, []byte("\"/path/to/structure\"")) {
		t.Errorf("marshal strbyte interface byte mismatch: %s != %s", string(strbytes), "\"/path/to/structure\"")
	}
}

func CompareStructures(a, b *Structure) error {
	if a == nil && b == nil {
		return nil
	} else if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("Structure mismatch: %s != %s", a, b)
	}

	if err := CompareSchemas(a.Schema, b.Schema); err != nil {
		return fmt.Errorf("Schema mismatch: %s", err.Error())
	}

	return nil
}
