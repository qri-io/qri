package lib

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
)

func TestApplyTransform(t *testing.T) {
	tr := newTestRunner(t)
	defer tr.Delete()

	// Save a dataset with a body
	_, err := tr.SaveWithParams(&SaveParams{
		Ref:      "me/cities_ds",
		BodyPath: "testdata/cities_2/body.csv",
	})
	if err != nil {
		t.Error(err)
	}

	// Apply a transformation
	res, err := tr.ApplyWithParams(&ApplyParams{
		Refstr: "me/cities_ds",
		Transform: &dataset.Transform{
			ScriptPath: "testdata/cities_2/add_city.star",
		},
		Wait: true,
	})
	if err != nil {
		t.Error(err)
	}

	output, err := json.Marshal(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	actual := string(output)
	expect := `[["toronto",50000000,55.5,false],["new york",8500000,44.4,true],["chicago",300000,44.4,true],["chatham",35000,65.25,true],["raleigh",250000,50.65,true],["tokyo",9200000,48.5,false]]`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("qri list (-want +got):\n%s", diff)
	}
}
