package dataset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/compare"
	"github.com/qri-io/dataset/datatypes"
)

func TestDatasetMarshalJSON(t *testing.T) {
	cases := []struct {
		in  *Dataset
		out []byte
		err error
	}{
		{&Dataset{}, []byte(`{"data":"","length":0,"structure":null,"timestamp":"0001-01-01T00:00:00Z","title":""}`), nil},
		// {AirportCodes, []byte(`{"format":"csv","formatConfig":{"header_row":true},"path":"","query":"","schema":{"fields":[{"name":"ident","type":"string"},{"name":"type","type":"string"},{"name":"name","type":"string"},{"name":"latitude_deg","type":"float"},{"name":"longitude_deg","type":"float"},{"name":"elevation_ft","type":"integer"},{"name":"continent","type":"string"},{"name":"iso_country","type":"string"},{"name":"iso_region","type":"string"},{"name":"municipality","type":"string"},{"name":"gps_code","type":"string"},{"name":"iata_code","type":"string"},{"name":"local_code","type":"string"}]}}`), nil},
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

func CompareDatasets(a, b *Dataset) error {
	if a.Title != b.Title {
		return fmt.Errorf("Title mismatch: %s != %s", a.Title, b.Title)
	}

	if err := compare.MapStringInterface(a.Meta(), b.Meta()); err != nil {
		fmt.Println(a.Meta())
		fmt.Println(b.Meta())
		return fmt.Errorf("meta mismatch: %s", err.Error())
	}

	if a.AccessUrl != b.AccessUrl {
		return fmt.Errorf("accessUrl mismatch: %s != %s", a.AccessUrl, b.AccessUrl)
	}
	if a.Readme != b.Readme {
		return fmt.Errorf("Readme mismatch: %s != %s", a.Readme, b.Readme)
	}
	if a.Author != b.Author {
		return fmt.Errorf("Author mismatch: %s != %s", a.Author, b.Author)
	}
	if a.Image != b.Image {
		return fmt.Errorf("Image mismatch: %s != %s", a.Image, b.Image)
	}
	if a.Description != b.Description {
		return fmt.Errorf("Description mismatch: %s != %s", a.Description, b.Description)
	}
	if a.Homepage != b.Homepage {
		return fmt.Errorf("Homepage mismatch: %s != %s", a.Homepage, b.Homepage)
	}
	if a.IconImage != b.IconImage {
		return fmt.Errorf("IconImage mismatch: %s != %s", a.IconImage, b.IconImage)
	}
	if a.DownloadUrl != b.DownloadUrl {
		return fmt.Errorf("DownloadUrl mismatch: %s != %s", a.DownloadUrl, b.DownloadUrl)
	}
	if err := CompareLicense(a.License, b.License); err != nil {
		return err
	}
	if a.Version != b.Version {
		return fmt.Errorf("Version mismatch: %s != %s", a.Version, b.Version)
	}
	if len(a.Keywords) != len(b.Keywords) {
		return fmt.Errorf("Keyword length mismatch: %s != %s", len(a.Keywords), len(b.Keywords))
	}
	// if a.Contributors != b.Contributors {
	//  return fmt.Errorf("Contributors mismatch: %s != %s", a.Contributors, b.Contributors)
	// }
	return nil
}
