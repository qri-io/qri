package dataset

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/datatypes"
)

func TestDatasetAssign(t *testing.T) {
	// TODO - expand test to check all fields
	expect := &Dataset{
		Title:       "Final Title",
		Description: "Final Description",
		AccessUrl:   "AccessUrl",
		Structure: &Structure{
			Schema: &Schema{
				Fields: []*Field{
					&Field{Type: datatypes.String, Name: "foo"},
					&Field{Type: datatypes.Integer, Name: "bar"},
					&Field{Description: "bat"},
				},
			},
		},
	}
	got := &Dataset{
		Title:       "Overwrite Me",
		Description: "Nope",
		Structure: &Structure{
			Schema: &Schema{
				Fields: []*Field{
					&Field{Type: datatypes.String},
					&Field{Type: datatypes.Integer},
				},
			},
		},
	}

	got.Assign(&Dataset{
		Title:       "Final Title",
		Description: "Final Description",
		AccessUrl:   "AccessUrl",
		Structure: &Structure{
			Schema: &Schema{
				Fields: []*Field{
					&Field{Name: "foo"},
					&Field{Name: "bar"},
					&Field{Description: "bat"},
				},
			},
		},
	})

	if err := CompareDatasets(expect, got); err != nil {
		t.Error(err)
	}

	got.Assign(nil, nil)
	if err := CompareDatasets(expect, got); err != nil {
		t.Error(err)
	}

	emptyDs := &Dataset{}
	emptyDs.Assign(expect)
	if err := CompareDatasets(expect, emptyDs); err != nil {
		t.Error(err)
	}
}

func TestDatasetMarshalJSON(t *testing.T) {
	cases := []struct {
		in  *Dataset
		out []byte
		err error
	}{
		{&Dataset{}, []byte(`{"data":"","length":0,"structure":null,"timestamp":"0001-01-01T00:00:00Z","title":""}`), nil},
		{AirportCodes, []byte(AirportCodesJSON), nil},
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

	strbytes, err := json.Marshal(&Dataset{path: datastore.NewKey("/path/to/dataset")})
	if err != nil {
		t.Errorf("unexpected string marshal error: %s", err.Error())
		return
	}

	if !bytes.Equal(strbytes, []byte("\"/path/to/dataset\"")) {
		t.Errorf("marshal strbyte interface byte mismatch: %s != %s", string(strbytes), "\"/path/to/dataset\"")
	}
}

func TestDatasetUnmarshalJSON(t *testing.T) {
	cases := []struct {
		FileName string
		result   *Dataset
		err      error
	}{
		{"testdata/datasets/airport-codes.json", AirportCodes, nil},
		{"testdata/datasets/continent-codes.json", ContinentCodes, nil},
		{"testdata/datasets/hours.json", Hours, nil},
	}

	for i, c := range cases {
		data, err := ioutil.ReadFile(c.FileName)
		if err != nil {
			t.Errorf("case %d couldn't read file: %s", i, err.Error())
		}

		ds := &Dataset{}
		if err := json.Unmarshal(data, ds); err != c.err {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if err = CompareDatasets(ds, c.result); err != nil {
			t.Errorf("case %d resource comparison error: %s", i, err)
			continue
		}
	}

	strds := &Dataset{}
	path := "/path/to/dataset"
	if err := json.Unmarshal([]byte(`"`+path+`"`), strds); err != nil {
		t.Errorf("unmarshal string path error: %s", err.Error())
		return
	}

	if strds.path.String() != path {
		t.Errorf("unmarshal didn't set proper path: %s != %s", path, strds.path)
		return
	}
}
