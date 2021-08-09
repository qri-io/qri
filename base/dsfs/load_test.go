package dsfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/event"
)

func TestLoadDataset(t *testing.T) {
	ctx := context.Background()
	fs := qfs.NewMemFS()
	dsData, err := ioutil.ReadFile("testdata/all_fields/input.dataset.json")
	if err != nil {
		t.Errorf("error loading test dataset: %s", err.Error())
		return
	}
	ds := &dataset.Dataset{}
	if err := ds.UnmarshalJSON(dsData); err != nil {
		t.Fatal(err)
	}
	body, err := ioutil.ReadFile("testdata/all_fields/body.csv")
	if err != nil {
		t.Fatal(err)
	}

	ds.SetBodyFile(qfs.NewMemfileBytes("/body.csv", body))

	// These tests are using hard-coded ids that require this exact peer's private key.
	pk := testkeys.GetKeyData(10).PrivKey

	apath, err := WriteDataset(ctx, fs, fs, nil, ds, event.NilBus, pk, SaveSwitches{})
	if err != nil {
		t.Fatal(err)
	}

	loadedDataset, err := LoadDataset(ctx, fs, apath)
	if err != nil {
		t.Fatal(err)
	}
	// prove we aren't returning a path to a dataset that ends with `/dataset.json`
	if strings.Contains(loadedDataset.Path, PackageFileDataset.Filename()) {
		t.Errorf("path should not contain the basename of the dataset file: %s", loadedDataset.Path)
	}

	cases := []struct {
		ds  *dataset.Dataset
		err string
	}{
		{dataset.NewDatasetRef("/bad/path"),
			"loading dataset: reading dataset.json file: path not found"},
		{&dataset.Dataset{
			Meta: dataset.NewMetaRef("QmFoo"),
		}, "loading dataset: reading dataset.json file: file is not a directory"},

		// TODO (b5) - fix
		// {&dataset.Dataset{
		// 	Structure: dataset.NewStructureRef(fmt.Sprintf("%s/bad/path", apath)),
		// }, "error loading dataset structure: error loading structure file: cafs: path not found"},
		// {&dataset.Dataset{
		// 	Structure: dataset.NewStructureRef("/bad/path"),
		// }, "error loading dataset structure: error loading structure file: cafs: path not found"},
		// {&dataset.Dataset{
		// 	Transform: dataset.NewTransformRef("/bad/path"),
		// }, "error loading dataset transform: error loading transform raw data: cafs: path not found"},
		// {&dataset.Dataset{
		// 	Commit: dataset.NewCommitRef("/bad/path"),
		// }, "error loading dataset commit: error loading commit file: cafs: path not found"},
		// {&dataset.Dataset{
		// 	Viz: dataset.NewVizRef("/bad/path"),
		// }, "error loading dataset viz: error loading viz file: cafs: path not found"},
	}

	for i, c := range cases {
		path := c.ds.Path
		if !c.ds.IsEmpty() {
			dsf, err := JSONFile(PackageFileDataset.String(), c.ds)
			if err != nil {
				t.Errorf("case %d error generating json file: %s", i, err.Error())
				continue
			}

			path, err = fs.Put(ctx, qfs.NewMemfileReader(PackageFileDataset.String(), dsf))
			if err != nil {
				t.Errorf("case %d error putting file in store: %s", i, err.Error())
				continue
			}
		}

		_, err = LoadDataset(ctx, fs, path)
		if !(err != nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}
	}
}

func TestLoadBody(t *testing.T) {
	ctx := context.Background()
	datasets, fs, err := makeFilestore()
	if err != nil {
		t.Fatalf("error creating test filestore: %s", err.Error())
	}

	ds, err := LoadDataset(ctx, fs, datasets["cities"])
	if err != nil {
		t.Fatalf("error loading dataset: %s", err.Error())
	}

	f, err := LoadBody(ctx, fs, ds)
	if err != nil {
		t.Fatalf("error loading data: %s", err.Error())
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("error reading data file: %s", err.Error())
	}

	eq, err := ioutil.ReadFile("testdata/cities/body.csv")
	if err != nil {
		t.Fatalf("error reading test file: %s", err.Error())
	}

	if !bytes.Equal(data, eq) {
		t.Errorf("byte mismatch. expected: %s, got: %s", string(eq), string(data))
	}
}

func TestLoadTransform(t *testing.T) {
	// TODO - restore
	// store := cafs.NewMapstore()
	// q := &dataset.AbstractTransform{Statement: "select * from whatever booooooo go home"}
	// a, err := SaveAbstractTransform(store, q, true)
	// if err != nil {
	// 	t.Errorf(err.Error())
	// 	return
	// }

	// if _, err = LoadTransform(store, a); err != nil {
	// 	t.Errorf(err.Error())
	// }
	// TODO - other tests & stuff
}

var ErrStreamsNotEqual = fmt.Errorf("streams are not equal")

// EqualReader confirms two readers are exactly the same, throwing an error
// if they return
type EqualReader struct {
	a, b io.Reader
}

func (r *EqualReader) Read(p []byte) (int, error) {
	pb := make([]byte, len(p))
	readA, err := r.a.Read(p)
	if err != nil {
		return readA, err
	}

	readB, err := r.b.Read(pb)
	if err != nil {
		return readA, err
	}

	if readA != readB {
		return readA, ErrStreamsNotEqual
	}

	for i, b := range p {
		if pb[i] != b {
			return readA, ErrStreamsNotEqual
		}
	}

	return readA, nil
}
