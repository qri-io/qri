package dsio

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
)

func TestJsonReader(t *testing.T) {
	cases := []struct {
		structure *dataset.Structure
		filepath  string
		count     int
		err       string
	}{
		{&dataset.Structure{
			Format: dataset.JsonDataFormat,
			FormatConfig: &dataset.JsonOptions{
				ArrayEntries: false,
			}}, "testdata/city_data.json", 5, ""},
	}

	for i, c := range cases {
		f, err := os.Open(c.filepath)
		if err != nil {
			t.Errorf("case %d error opening data file: %s", i, err.Error())
			continue
		}

		r := NewJsonReader(c.structure, f)
		j := 0
		for {
			// TODO - inspect row output for well formed json
			_, err := r.ReadRow()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				t.Errorf("case %d error reading row %d: %s", i, j, err.Error())
				break
			}
			j++
		}

		if c.count != j {
			t.Errorf("case %d count mismatch. expected: %d, got: %d", i, c.count, j)
			continue
		}

		// for _, ent := range c.entries {
		// 	if err := r.ReadRow(ent); err != nil {
		// 		t.Errorf("case %d WriteRow error: %s", i, err.Error())
		// 		break
		// 	}
		// }
		// if err := w.Close(); err != nil {
		// 	t.Errorf("case %d Close error: %s", i, err.Error())
		// }

		// if string(buf.Bytes()) != c.out {
		// 	t.Errorf("case %d result mismatch. expected:\n%s\ngot:\n%s", i, c.out, string(buf.Bytes()))
		// }

		// var v interface{}
		// if cfg, ok := c.structure.FormatConfig.(*dataset.JsonOptions); ok && cfg.ArrayEntries {
		// 	v = []interface{}{}
		// } else {
		// 	v = map[string]interface{}{}
		// }

		// if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		// 	t.Errorf("unmarshal error: %s", err.Error())
		// }
	}
}

func TestJsonWriter(t *testing.T) {
	cases := []struct {
		structure *dataset.Structure
		entries   [][][]byte
		out       string
	}{
		{&dataset.Structure{Schema: &dataset.Schema{Fields: []*dataset.Field{&dataset.Field{Name: "a", Type: datatypes.String}}}}, [][][]byte{}, "[]"},
		{&dataset.Structure{Schema: &dataset.Schema{Fields: []*dataset.Field{&dataset.Field{Name: "a", Type: datatypes.String}}}}, [][][]byte{[][]byte{[]byte("hello")}}, "[\n{\"a\":\"hello\"}\n]"},
		{&dataset.Structure{Schema: &dataset.Schema{Fields: []*dataset.Field{&dataset.Field{Name: "a", Type: datatypes.String}}}, FormatConfig: &dataset.JsonOptions{ArrayEntries: true}}, [][][]byte{[][]byte{[]byte("hello")}}, "[\n[\"hello\"]\n]"},
		{&dataset.Structure{Schema: &dataset.Schema{Fields: []*dataset.Field{&dataset.Field{Name: "a", Type: datatypes.String}}}}, [][][]byte{
			[][]byte{[]byte("hello")},
			[][]byte{[]byte("world")},
		}, "[\n{\"a\":\"hello\"},\n{\"a\":\"world\"}\n]"},
		// {&dataset.Structure{ Schema: &dataset.Schema{ Fields: []*dataset.Field{&dataset.Field{Name: "a", Type: datatypes.String}}}}, [][][]byte{
		// 	[][]byte{[]byte("hello")},
		// 	[][]byte{[]byte("world")},
		// }, "[\n[\"hello\"],\n[\"world\"]\n]"},
		{&dataset.Structure{Schema: &dataset.Schema{Fields: []*dataset.Field{&dataset.Field{Name: "a", Type: datatypes.String}}}, FormatConfig: &dataset.JsonOptions{ArrayEntries: true}}, [][][]byte{
			[][]byte{[]byte("hello\n?")},
			[][]byte{[]byte("world")},
		}, "[\n[\"hello\\n?\"],\n[\"world\"]\n]"},
		{&dataset.Structure{Schema: &dataset.Schema{
			Fields: []*dataset.Field{
				&dataset.Field{Name: "ident", Type: datatypes.String},
				&dataset.Field{Name: "type", Type: datatypes.String},
				&dataset.Field{Name: "name", Type: datatypes.String},
				&dataset.Field{Name: "latitude_deg", Type: datatypes.Float},
				&dataset.Field{Name: "longitude_deg", Type: datatypes.Float},
				&dataset.Field{Name: "elevation_ft", Type: datatypes.Integer},
				&dataset.Field{Name: "continent", Type: datatypes.String},
				&dataset.Field{Name: "iso_country", Type: datatypes.String},
				&dataset.Field{Name: "iso_region", Type: datatypes.String},
				&dataset.Field{Name: "municipality", Type: datatypes.String},
				&dataset.Field{Name: "gps_code", Type: datatypes.String},
				&dataset.Field{Name: "iata_code", Type: datatypes.String},
				&dataset.Field{Name: "local_code", Type: datatypes.String},
				&dataset.Field{Name: "bool_teim", Type: datatypes.Boolean},
			}},
			FormatConfig: &dataset.JsonOptions{ArrayEntries: true}},
			[][][]byte{
				[][]byte{[]byte("00AR"), []byte("heliport"), []byte("Newport Hospital & Clinic Heliport"), []byte{}, []byte{}, []byte{}, []byte("NA"), []byte("US"), []byte("US-AR"), []byte("Newport"), []byte("00AR"), []byte{}, []byte("00AR"), []byte{}},
			},
			// "[\n[\"00AR\",\"heliport\",\"Newport Hospital & Clinic Heliport\",0,0,0,\"NA\",\"US\",\"US-AR\",\"Newport\",\"00AR\",\"\",\"00AR\",false]\n]",
			`[
["00AR","heliport","Newport Hospital & Clinic Heliport",null,null,null,"NA","US","US-AR","Newport","00AR",null,"00AR",null]
]`,
		},
		{&dataset.Structure{Schema: &dataset.Schema{
			Fields: []*dataset.Field{
				&dataset.Field{Name: "ident", Type: datatypes.String},
				&dataset.Field{Name: "type", Type: datatypes.String},
				&dataset.Field{Name: "name", Type: datatypes.String},
				&dataset.Field{Name: "latitude_deg", Type: datatypes.Float},
				&dataset.Field{Name: "longitude_deg", Type: datatypes.Float},
				&dataset.Field{Name: "elevation_ft", Type: datatypes.Integer},
				&dataset.Field{Name: "continent", Type: datatypes.String},
				&dataset.Field{Name: "iso_country", Type: datatypes.String},
				&dataset.Field{Name: "iso_region", Type: datatypes.String},
				&dataset.Field{Name: "municipality", Type: datatypes.String},
				&dataset.Field{Name: "gps_code", Type: datatypes.String},
				&dataset.Field{Name: "iata_code", Type: datatypes.String},
				&dataset.Field{Name: "local_code", Type: datatypes.String},
				&dataset.Field{Name: "bool_teim", Type: datatypes.Boolean},
			}}},
			[][][]byte{
				[][]byte{[]byte("00AR"), []byte("heliport"), []byte("Newport Hospital & Clinic Heliport"), []byte{}, []byte("0"), []byte{}, []byte("NA"), []byte("US"), []byte("US-AR"), []byte("Newport"), []byte("00AR"), []byte{}, []byte("00AR"), []byte{}},
			},
			`[
{"ident":"00AR","type":"heliport","name":"Newport Hospital & Clinic Heliport","latitude_deg":null,"longitude_deg":0,"elevation_ft":null,"continent":"NA","iso_country":"US","iso_region":"US-AR","municipality":"Newport","gps_code":"00AR","iata_code":null,"local_code":"00AR","bool_teim":null}
]`,
		},
		{&dataset.Structure{Schema: &dataset.Schema{
			Fields: []*dataset.Field{
				&dataset.Field{Name: "name", Type: datatypes.String},
				&dataset.Field{Name: "metadata", Type: datatypes.Json},
			}}},
			[][][]byte{
				[][]byte{[]byte("name_one"), []byte(`{ "data" : "stuff", "foo" : 5, "false" : true }`)},
				[][]byte{[]byte("name_two"), []byte(`["stuff",5,false,null,27.5]`)},
			},
			`[
{"name":"name_one","metadata":{ "data" : "stuff", "foo" : 5, "false" : true }},
{"name":"name_two","metadata":["stuff",5,false,null,27.5]}
]`,
		},
	}

	for i, c := range cases {
		buf := &bytes.Buffer{}
		w := NewJsonWriter(c.structure, buf)
		for _, ent := range c.entries {
			if err := w.WriteRow(ent); err != nil {
				t.Errorf("case %d WriteRow error: %s", i, err.Error())
				break
			}
		}
		if err := w.Close(); err != nil {
			t.Errorf("case %d Close error: %s", i, err.Error())
		}

		if string(buf.Bytes()) != c.out {
			t.Errorf("case %d result mismatch. expected:\n%s\ngot:\n%s", i, c.out, string(buf.Bytes()))
		}

		var v interface{}
		if cfg, ok := c.structure.FormatConfig.(*dataset.JsonOptions); ok && cfg.ArrayEntries {
			v = []interface{}{}
		} else {
			v = map[string]interface{}{}
		}

		if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
			t.Errorf("unmarshal error: %s", err.Error())
		}
	}
}
