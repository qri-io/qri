package startf

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/tabular"
	"github.com/qri-io/qfs"
)

func TestDatasetBodyGet(t *testing.T) {
	bodyData := `cat,meow,5
dog,bark,6
eel,zap,7
`
	ds := &dataset.Dataset{
		Name: "my_ds",
		Structure: &dataset.Structure{
			Format: "csv",
			Schema: tabular.BaseTabularSchema,
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.csv", []byte(bodyData)))
	runExecScript(t, ds, "testdata/dataset_body_get.star", "testdata/dataset_body_get.expect.txt")
}

func TestDatasetBodySet(t *testing.T) {
	ds := &dataset.Dataset{
		Name: "my_ds",
		Structure: &dataset.Structure{
			Format: "csv",
			Schema: tabular.BaseTabularSchema,
		},
	}
	runExecScript(t, ds, "testdata/dataset_body_set.star", "testdata/dataset_body_set.expect.txt")
}

func runExecScript(t *testing.T, ds *dataset.Dataset, scriptFilename, expectFilename string) {
	ctx := context.Background()

	ds.Transform = &dataset.Transform{}
	ds.Transform.Syntax = "starlark"
	ds.Transform.SetScriptFile(scriptFile(t, scriptFilename))

	// Run the script and capture its print output
	buf := &bytes.Buffer{}
	err := ExecScript(ctx, ds, SetErrWriter(buf))
	if err != nil {
		t.Fatal(err)
	}
	actual := string(buf.Bytes())

	// Compare the actual output to the expected text
	expectBytes, err := ioutil.ReadFile(expectFilename)
	if err != nil {
		t.Fatal(err)
	}
	expect := string(expectBytes)
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("mismatch. (-want +got):\n%s", diff)
	}
}
