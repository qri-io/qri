package stats

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qfs"
)

type TestCase struct {
	Description string
	JSONSchema  string
	JSONInput   string
	Expect      interface{}
}

func TestStrings(t *testing.T) {
	strings := TestCase{
		"an array of strings",
		`{"type":"array"}`,
		`["a","a","bb","ccc","dddd"]`,
		[]map[string]interface{}{
			{
				"type":        "string",
				"count":       5,
				"minLength":   1,
				"maxLength":   4,
				"unique":      4,
				"frequencies": map[string]int{"a": 2, "bb": 1, "ccc": 1, "dddd": 1},
			},
		},
	}

	runTestCases(t, strings)
}

func TestAllTypesIdentitySchemaArray(t *testing.T) {
	allTypesIdentitySchemaArray := TestCase{
		"all types identity schema array of object entries",
		`{"type":"array"}`,
		`[
			{"int": 1, "float": 1.1, "nil": null, "bool": false, "string": "a"},
			{"int": 1, "float": 1.1, "nil": null, "bool": true, "string": "aa"},
			{"int": 3, "float": 3.3, "nil": null, "bool": false, "string": "aaa"},
			{"int": 4, "float": 4.4, "nil": null, "bool": true, "string": "aaa"},
			{"int": 5, "float": 5.5, "nil": null, "bool": false, "string": "aaaaa"}
		]`,
		[]map[string]interface{}{
			{
				"key":        "bool",
				"count":      5,
				"trueCount":  2,
				"falseCount": 3,
				"type":       "boolean",
			},
			{
				"key":    "float",
				"count":  5,
				"min":    float64(1.1),
				"max":    float64(5.5),
				"mean":   float64(3.08),
				"median": float64(3.3),
				"type":   "numeric",
				"histogram": map[string][]float64{
					"bins":        {1.1, 3.3, 4.4, 5.5, 6.5},
					"frequencies": {2, 1, 1, 1},
				},
			},
			{
				"key":    "int",
				"count":  5,
				"min":    float64(1),
				"max":    float64(5),
				"mean":   float64(2.8),
				"median": float64(3),
				"type":   "numeric",
				"histogram": map[string][]float64{
					"bins":        {1, 3, 4, 5, 6},
					"frequencies": {2, 1, 1, 1},
				},
			},
			{
				"key":   "nil",
				"count": 5,
				"type":  "null",
			},
			{
				"key":         "string",
				"count":       5,
				"minLength":   1,
				"maxLength":   5,
				"type":        "string",
				"unique":      4,
				"frequencies": map[string]int{"aaa": 2, "aa": 1, "a": 1, "aaaaa": 1},
			},
		},
	}

	medianTestCase := TestCase{
		"median test cast",
		`{"type":"array"}`,
		`[
			[2],
			[1],
		]`,
		[]map[string]interface{}{
			{
				"count":  2,
				"min":    float64(1),
				"max":    float64(2),
				"mean":   float64(1.5),
				"median": float64(2),
				"type":   "numeric",
				"histogram": map[string][]float64{
					"bins":        {1, 2, 3},
					"frequencies": {1, 1},
				},
			},
		},
	}

	runTestCases(t, allTypesIdentitySchemaArray, medianTestCase)
}

func TestAllTypesIdentitySchemaObject(t *testing.T) {
	allTypesIdentitySchemaObject := TestCase{
		"all types identity schema object of array entries",
		`{"type":"object"}`,
		`{
			"a" : [5,1.1,null,false,"a"],
			"b" : [4,2.2,null,true,"aa"],
			"c" : [3,2.2,null,false,"aaa"],
			"d" : [1,4.4,null,true,"aaa"],
			"e" : [1,5.5,null,false,"aaaaa"]
		}`,
		[]map[string]interface{}{
			{
				"count":  5,
				"min":    float64(1),
				"max":    float64(5),
				"mean":   float64(2.8),
				"median": float64(3),
				"type":   "numeric",
				"histogram": map[string][]float64{
					"bins":        {1, 3, 4, 5, 6},
					"frequencies": {2, 1, 1, 1},
				},
			},
			{
				"count":  5,
				"min":    float64(1.1),
				"max":    float64(5.5),
				"mean":   float64(3.08),
				"median": float64(4.4), // median calculation is probabalistic and leans to the right
				"type":   "numeric",
				"histogram": map[string][]float64{
					"bins":        {1.1, 2.2, 4.4, 5.5, 6.5},
					"frequencies": {1, 2, 1, 1},
				},
			},
			{
				"count": 5,
				"type":  "null",
			},
			{
				"count":      5,
				"trueCount":  2,
				"falseCount": 3,
				"type":       "boolean",
			},
			{
				"count":       5,
				"minLength":   1,
				"maxLength":   5,
				"type":        "string",
				"unique":      4,
				"frequencies": map[string]int{"aaa": 2, "a": 1, "aa": 1, "aaaaa": 1},
			},
		},
	}

	runTestCases(t, allTypesIdentitySchemaObject)
}

func TestFreqThreshold(t *testing.T) {
	prev := StopFreqCountThreshold
	StopFreqCountThreshold = 2
	defer func() { StopFreqCountThreshold = prev }()

	less := TestCase{
		"fewer unique values than threhold",
		`{"type":"array"}`,
		`[
			["abcdefghijk",1],
			["abcdefghijk",1],
			["abcdefghijk",1],
			["abcdefghijk",1],
			["abcdefghijk",1]
		]`,
		[]map[string]interface{}{
			{
				"count":       5,
				"minLength":   11,
				"maxLength":   11,
				"type":        "string",
				"frequencies": map[string]int{"abcdefghijk": 5},
				"unique":      1,
			},
			{
				"count":  5,
				"min":    float64(1),
				"max":    float64(1),
				"mean":   float64(1),
				"median": float64(1),
				// currently we're calculating historams at 100x the stop threshold, so this shows up
				"histogram": map[string][]float64{
					"bins":        {1, 2},
					"frequencies": {5},
				},
				"type": "numeric",
			},
		},
	}

	more := TestCase{
		"more unique values than threhold",
		`{"type":"array"}`,
		`[
			["a",1],
			["b",2],
			["c",3],
			["d",4],
			["e",5]
		]`,
		[]map[string]interface{}{
			{
				"count":       5,
				"minLength":   1,
				"maxLength":   1,
				"frequencies": map[string]int{"b": 1, "e": 2}, //frequency counts are probabalistic and can overestimate
				"unique":      5,
				"type":        "string",
			},
			{
				"count":  5,
				"min":    float64(1),
				"max":    float64(5),
				"mean":   float64(3),
				"median": float64(3),
				// currently we're calculating historams at 100x the stop threshold, so this shows up
				"histogram": map[string][]float64{
					"bins":        {1, 2, 3, 4, 5, 6},
					"frequencies": {1, 1, 1, 1, 1},
				},
				"type": "numeric",
			},
		},
	}

	runTestCases(t, less, more)
}

func TestDepth3(t *testing.T) {
	t.SkipNow()

	depth3 := TestCase{
		"array of object of array of strings",
		`{"type":"array"}`,
		`[
			{"ids": ["a","b","c"], "is_great": true },
			{"ids": [1,2,3,4,5,6] },
			{"ids": ["b",20,"c"] }
		]`,
		[]map[string]interface{}{
			{
				"key":  "ids",
				"type": "array",
				"values": []map[string]interface{}{
					{"count": 2, "maxLength": 1, "minLength": 1},
					{"count": 2, "maxLength": 1, "minLength": 1},
				},
			},
		},
	}

	runTestCases(t, depth3)
}

func runTestCases(t *testing.T, cases ...TestCase) {
	for i, c := range cases {
		var sch map[string]interface{}
		if err := json.Unmarshal([]byte(c.JSONSchema), &sch); err != nil {
			t.Errorf("%d. %s error decoding schema: %s", i, c.Description, err)
			continue
		}
		st := &dataset.Structure{
			Format: "json",
			Schema: sch,
		}
		if c.JSONInput[0] == '{' {
			st.Schema = dataset.BaseSchemaObject
		}
		r, err := dsio.NewJSONReader(st, strings.NewReader(c.JSONInput))
		if err != nil {
			t.Errorf("%d. %s error creating json reader: %s", i, c.Description, err)
			continue
		}
		acc := NewAccumulator(r)

		err = ReadAllDiscard(acc)
		got := ToMap(acc)
		if diff := cmp.Diff(c.Expect, got); diff != "" {
			t.Errorf("%d. '%s' result mismatch (-want +got):%s\n", i, c.Description, diff)
		}
	}
}

// ReadAllDiscard consumes all reader entries, discarding entries
func ReadAllDiscard(r dsio.EntryReader) (err error) {
	defer r.Close()
	for {
		_, err = r.ReadEntry()
		if err != nil {
			if err.Error() == "EOF" {
				err = nil
				break
			}
			return err
		}
	}
	return err
}
func TestJSON(t *testing.T) {
	ctx := context.Background()
	bodyFile := qfs.NewMemfileBytes("bodyfile", []byte("[bad body file]"))
	structure := &dataset.Structure{
		Format: "json",
		Schema: map[string]interface{}{
			"type": "array",
		},
	}
	dsWithBody := &dataset.Dataset{Path: "path"}
	dsWithBody.SetBodyFile(bodyFile)
	dsWithStructure := &dataset.Dataset{Path: "path", Structure: structure}
	dsWithStructure.SetBodyFile(bodyFile)
	badCases := []struct {
		description string
		dataset     *dataset.Dataset
		err         string
	}{
		{"no body", &dataset.Dataset{Path: "path"}, "stats: dataset has no body file"},
		{"no structure", dsWithBody, "stats: dataset is missing structure"},
		{"reader error", dsWithStructure, "Expected: separator ','"},
	}

	for _, c := range badCases {
		s := New(nil)
		_, err := s.JSON(ctx, c.dataset)
		if c.err != err.Error() {
			t.Errorf("case '%s', error mismatch, expected: '%s', got: '%s'", c.description, c.err, err.Error())
		}
	}

	goodCases := []struct {
		Description string
		Format      string
		Schema      string
		Input       string
		Expect      []byte
	}{
		{
			"json: an array of strings",
			"json",
			`{"type":"array"}`,
			`["a","a","bb","ccc","dddd"]`,
			[]byte(`[{"count":5,"frequencies":{"a":2,"bb":1,"ccc":1,"dddd":1},"maxLength":4,"minLength":1,"type":"string","unique":4}]`),
		}, {
			"json: all types identity schema array of object entries",
			"json",
			`{"type":"array"}`,
			`[
				{"int": 1, "float": 1.1, "nil": null, "bool": false, "string": "a"},
				{"int": 1, "float": 1.1, "nil": null, "bool": true, "string": "aa"},
				{"int": 3, "float": 3.3, "nil": null, "bool": false, "string": "aaa"},
				{"int": 4, "float": 4.4, "nil": null, "bool": true, "string": "aaa"},
				{"int": 5, "float": 5.5, "nil": null, "bool": false, "string": "aaaaa"}
			]`,
			[]byte(`[{"count":5,"falseCount":3,"key":"bool","trueCount":2,"type":"boolean"},{"count":5,"histogram":{"bins":[1.1,3.3,4.4,5.5,6.5],"frequencies":[2,1,1,1]},"key":"float","max":5.5,"mean":3.08,"median":3.3,"min":1.1,"type":"numeric"},{"count":5,"histogram":{"bins":[1,3,4,5,6],"frequencies":[2,1,1,1]},"key":"int","max":5,"mean":2.8,"median":3,"min":1,"type":"numeric"},{"count":5,"key":"nil","type":"null"},{"count":5,"frequencies":{"a":1,"aa":1,"aaa":2,"aaaaa":1},"key":"string","maxLength":5,"minLength":1,"type":"string","unique":4}]`),
		}, {
			"csv: an array of strings",
			"csv",
			`{"type":"array", "items": { "type": "array", "items": [{ "title": "str_col", "type": "string" }] }}`,
			"a\na\nbb\nccc\ndddd",
			[]byte(`[{"count":5,"frequencies":{"a":2,"bb":1,"ccc":1,"dddd":1},"maxLength":4,"minLength":1,"type":"string","unique":4}]`),
		}, {
			"csv: all types identity schema array of object entries",
			"csv",
			`{
				"items": {
				 "items": [
					{
					 "title": "int",
					 "type": "integer"
					},
					{
					 "title": "float",
					 "type": "number"
					},
					{
					 "title": "nil",
					 "type": "null"
					},
					{
					 "title": "bool",
					 "type": "boolean"
					},
					{
					 "title": "string",
					 "type": "string"
					}
				 ],
				 "type": "array"
				},
				"type": "array"
			 }`,
			"1,1.1,,false,a\n1,1.1,,true,aa\n3,3.3,,false,aaa\n4,4.4,,true,aaa\n5,5.5,,false,aaaaa",
			[]byte(`[{"count":5,"histogram":{"bins":[1,3,4,5,6],"frequencies":[2,1,1,1]},"max":5,"mean":2.8,"median":3,"min":1,"type":"numeric"},{"count":5,"histogram":{"bins":[1.1,3.3,4.4,5.5,6.5],"frequencies":[2,1,1,1]},"max":5.5,"mean":3.08,"median":3.3,"min":1.1,"type":"numeric"},{"count":5,"type":"null"},{"count":5,"falseCount":3,"trueCount":2,"type":"boolean"},{"count":5,"frequencies":{"a":1,"aa":1,"aaa":2,"aaaaa":1},"maxLength":5,"minLength":1,"type":"string","unique":4}]`),
		}, {
			"json: all types identity schema object of array entries",
			"json",
			`{"type":"object"}`,
			`{
					"a" : [1,1.1,null,false,"a"],
					"b" : [1,2.2,null,true,"aa"],
					"c" : [3,2.2,null,false,"aaa"],
					"d" : [4,4.4,null,true,"aaa"],
					"e" : [5,5.5,null,false,"aaaaa"]
				}`,
			[]byte(`[{"count":5,"histogram":{"bins":[1,3,4,5,6],"frequencies":[2,1,1,1]},"max":5,"mean":2.8,"median":3,"min":1,"type":"numeric"},{"count":5,"histogram":{"bins":[1.1,2.2,4.4,5.5,6.5],"frequencies":[1,2,1,1]},"max":5.5,"mean":3.08,"median":4.4,"min":1.1,"type":"numeric"},{"count":5,"type":"null"},{"count":5,"falseCount":3,"trueCount":2,"type":"boolean"},{"count":5,"frequencies":{"a":1,"aa":1,"aaa":2,"aaaaa":1},"maxLength":5,"minLength":1,"type":"string","unique":4}]`),
		}, {
			"json: array of object of array of strings",
			"json",
			`{"type":"array"}`,
			`[
					{"ids": ["a","b","c"], "is_great": true },
					{"ids": [1,2,3,4,5,6] },
					{"ids": ["b",20,"c"] }
				]`,
			[]byte(`[{"key":"ids","type":"array","values":[{"count":2,"frequencies":{"a":1,"b":1},"maxLength":1,"minLength":1,"unique":2},{"count":1,"frequencies":{"b":1},"maxLength":1,"minLength":1,"unique":1},{"count":2,"frequencies":{"c":2},"maxLength":1,"minLength":1,"unique":1},{"count":1,"histogram":{"bins":[4,5],"frequencies":[1]},"max":4,"mean":4,"median":4,"min":4},{"count":1,"histogram":{"bins":[5,6],"frequencies":[1]},"max":5,"mean":5,"median":5,"min":5},{"count":1,"histogram":{"bins":[6,7],"frequencies":[1]},"max":6,"mean":6,"median":6,"min":6}]},{"count":1,"falseCount":0,"key":"is_great","trueCount":1,"type":"boolean"}]`),
		},
	}
	for i, c := range goodCases {
		var sch map[string]interface{}
		if err := json.Unmarshal([]byte(c.Schema), &sch); err != nil {
			t.Errorf("%d. %s error decoding schema: %s", i, c.Description, err)
			continue
		}
		st := &dataset.Structure{
			Format: c.Format,
			Schema: sch,
		}
		ds := &dataset.Dataset{Path: "path", Structure: st}
		bodyFile := qfs.NewMemfileBytes("bodyfile", []byte(c.Input))
		ds.SetBodyFile(bodyFile)

		s := New(nil)
		r, err := s.JSON(ctx, ds)
		if err != nil {
			t.Errorf("%d. %s unexpected error: %s", i, c.Description, err)
		}
		got, err := ioutil.ReadAll(r)
		if err != nil {
			t.Errorf("%d. %s unexpected read error: %s", i, c.Description, err)
		}
		if diff := cmp.Diff(c.Expect, got); diff != "" {
			t.Errorf("%d. '%s' result mismatch (-want +got):%s\n", i, c.Description, diff)
		}
	}
}
