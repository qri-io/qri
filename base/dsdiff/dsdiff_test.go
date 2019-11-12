package dsdiff

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/qri-io/dataset"
)

func loadTestData(path string) (*dataset.Dataset, error) {
	dataBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	d := &dataset.Dataset{}
	err = d.UnmarshalJSON(dataBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal dataset: %s", err.Error())
	}
	return d, nil
}

func TestDiffDataset(t *testing.T) {
	//test cases
	cases := []struct {
		dsLeftPath, dsRightPath string
		displayFormat           string
		expected                string
		err                     string
	}{
		{"testdata/orig.json", "testdata/newStructure.json", "simple", "Structure Changed. (3 changes)", ""},
		{"testdata/orig.json", "testdata/newStructure.json", "delta", `{
  "checksum": [
    "@@ -33,7 +33,7 @@\n y9Jc\n-ud9\n+aaa\n",
    0,
    2
  ],
  "entries": [
    33,
    35
  ],
  "schema": {
    "items": {
      "items": {
        "0": {
          "title": [
            "rank",
            "ranking"
          ]
        },
        "1": {
          "title": [
            "probability_of_automation",
            "prob_of_automation"
          ]
        },
        "_t": "a"
      }
    }
  }
}
`, ""},
		{"testdata/orig.json", "testdata/newTitle.json", "listKeys", "Transform: 2 changes\n\t- modified config\n\t- modified syntax", ""},
		{"testdata/orig.json", "testdata/newDescription.json", "plusMinusColor", ` {
[30;41m-  "description": "I am a dataset",[0m
[30;42m+  "description": "I am a new description",[0m
   "qri": "md:0",
   "title": "abc"
 }
`, ""},
		{"testdata/orig.json", "testdata/newVisConfig.json", "plusMinus", ` {
-  "format": "abc",
+  "format": "new thing",
   "qri": "vz:0"
 }
`, ""},
		{"testdata/orig.json", "testdata/newTransform.json", "simple", "Transform Changed. (2 changes)", ""},
		// {"testdata/orig.json", "testdata/newData.json", "simple", "Data Changed. (1 change)", ""},
	}
	// execute
	for i, c := range cases {
		//Load data
		dsLeft, err := loadTestData(c.dsLeftPath)
		if err != nil {
			t.Errorf("case %d error: error loading file '%s'", i, c.dsLeftPath)
			return
		}
		dsRight, err := loadTestData(c.dsRightPath)
		if err != nil {
			t.Errorf("case %d error: error loading file '%s'", i, c.dsRightPath)
			return
		}
		got, err := DiffDatasets(dsLeft, dsRight, nil)
		if err != nil {
			if err.Error() == c.err {
				continue
			} else {
				t.Errorf("case %d error mismatch: expected '%s', got '%s'", i, c.err, err.Error())
				return
			}
		}
		stringDiffs, err := MapDiffsToString(got, c.displayFormat)
		if err != nil {
			t.Errorf("case %d error: %s", i, err.Error())
			return
		}
		// if i == 0 {
		//  s, err := MapDiffsToFormattedString(got, dsLeft)
		//  if err != nil {
		//    t.Errorf("not today: %s", err.Error())
		//  }
		//  fmt.Println("--------------------------")
		//  fmt.Print(s)
		//  fmt.Println("--------------------------")
		// }

		if stringDiffs != c.expected {
			// texp := []byte(c.expected)
			tgot := []byte(stringDiffs)
			_ = ioutil.WriteFile("got0.txt", tgot, 0775)
			t.Errorf("case %d response mistmatch: expected '%s', got '%s'", i, c.expected, stringDiffs)
		}
	}
}

func TestDiffJSON(t *testing.T) {
	//test cases
	cases := []struct {
		dsLeftPath, dsRightPath string
		description             string
		expected                string
		err                     string
	}{
		{"testdata/orig.json", "testdata/newStructure.json", "abc", "3 diffs", ""},
	}
	// execute
	for i, c := range cases {
		//Load data
		a, err := loadTestData(c.dsLeftPath)
		if err != nil {
			t.Errorf("case %d error: error loading file '%s'", i, c.dsLeftPath)
			return
		}
		b, err := loadTestData(c.dsRightPath)
		if err != nil {
			t.Errorf("case %d error: error loading file '%s'", i, c.dsRightPath)
			return
		}
		aBytes, err := json.Marshal(a)
		if err != nil {
			t.Errorf("error marshalling structure a: %s", err.Error())
			return
		}
		bBytes, err := json.Marshal(b)
		if err != nil {
			t.Errorf("error marshalling structure b: %s", err.Error())
			return
		}
		got, err := DiffJSON(aBytes, bBytes, c.description)
		if err != nil {
			if err.Error() == c.err {
				continue
			} else {
				t.Errorf("case %d error mismatch: expected '%s', got '%s'", i, c.err, err.Error())
				return
			}
		}
		stringDiffs := fmt.Sprintf("%d diffs", len(got.Deltas()))
		if stringDiffs != c.expected {
			t.Errorf("case %d response mistmatch: expected '%s', got '%s'", i, c.expected, stringDiffs)
		}
	}
}

func TestDiffTransform(t *testing.T) {
	// Same structs
	diff, err := DiffTransform(&dataset.Transform{
		ScriptBytes: []byte("return [1,2]"),
		ScriptPath: "path/to/script.star",
	}, &dataset.Transform{
		ScriptBytes: []byte("return [1,2]"),
		ScriptPath: "path/to/script.star",
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff.Diff.Modified() {
		t.Errorf("error: expected not to have been modified")
	}

	// Different bytes
	diff, err = DiffTransform(&dataset.Transform{
		ScriptBytes: []byte("return [1,2]"),
		ScriptPath: "path/to/script.star",
	}, &dataset.Transform{
		ScriptBytes: []byte("return [1,2,3]"),
		ScriptPath: "path/to/script.star",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !diff.Diff.Modified() {
		t.Errorf("error: expected modification")
	}

	// A blank path, so different
	diff, err = DiffTransform(&dataset.Transform{
		ScriptBytes: []byte("return [1,2]"),
		ScriptPath: "",
	}, &dataset.Transform{
		ScriptBytes: []byte("return [1,2]"),
		ScriptPath: "path/to/script.star",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !diff.Diff.Modified() {
		t.Errorf("error: expected modification")
	}

	// A blank path, on the other one, so different
	diff, err = DiffTransform(&dataset.Transform{
		ScriptBytes: []byte("return [1,2]"),
		ScriptPath: "path/to/script.star",
	}, &dataset.Transform{
		ScriptBytes: []byte("return [1,2]"),
		ScriptPath: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !diff.Diff.Modified() {
		t.Errorf("error: expected modification")
	}

	// Paths are different, but both non-blank, so considered the same
	diff, err = DiffTransform(&dataset.Transform{
		ScriptBytes: []byte("return [1,2]"),
		ScriptPath: "path/to/script.star",
	}, &dataset.Transform{
		ScriptBytes: []byte("return [1,2]"),
		ScriptPath: "path/to/renamed-file.star",
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff.Diff.Modified() {
		t.Errorf("error: expected not to have been modified")
	}
}
