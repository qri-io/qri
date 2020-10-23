package archive

import (
	"archive/zip"
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
)

var blankInitID = ""

func TestWriteZip(t *testing.T) {
	ctx := context.Background()
	fs, names, err := testStore()
	if err != nil {
		t.Errorf("error creating store: %s", err.Error())
		return
	}

	ds, err := dsfs.LoadDataset(ctx, store, names["movies"])
	if err != nil {
		t.Errorf("error fetching movies dataset from store: %s", err.Error())
		return
	}

	buf := &bytes.Buffer{}
	err = WriteZip(ctx, store, ds, "yaml", blankInitID, dsref.MustParse("peer/ref@/ipfs/Qmb"), buf)
	if err != nil {
		t.Errorf("error writing zip archive: %s", err.Error())
		return
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Errorf("error creating zip reader: %s", err.Error())
		return
	}

	// TODO (dlong): Actually test the contents of the zip.
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Errorf("error opening file %s in package", f.Name)
			break
		}

		if err := rc.Close(); err != nil {
			t.Errorf("error closing file %s in package", f.Name)
			break
		}
	}
}

func TestWriteZipFullDataset(t *testing.T) {
	ctx := context.Background()
	fs, names, err := testFSWithVizAndTransform()
	if err != nil {
		t.Errorf("error creating filesystem: %s", err.Error())
		return
	}

	ds, err := dsfs.LoadDataset(ctx, fs, names["movies"])
	if err != nil {
		t.Errorf("error fetching movies dataset from store: %s", err.Error())
		return
	}

	err = base.OpenDataset(ctx, fs, ds)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.Get(ctx, names["transform_script"])
	if err != nil {
		t.Errorf("error fetching movies dataset from store: %s", err.Error())
		return
	}

	buf := &bytes.Buffer{}
	err = WriteZip(ctx, fs, ds, "json", blankInitID, dsref.MustParse("peer/ref@/ipfs/Qmb"), buf)
	if err != nil {
		t.Errorf("error writing zip archive: %s", err.Error())
		return
	}

	tmppath := filepath.Join(os.TempDir(), "exported.zip")
	// defer os.RemoveAll(tmppath)
	t.Log(tmppath)
	err = ioutil.WriteFile(tmppath, buf.Bytes(), os.ModePerm)
	if err != nil {
		t.Errorf("error writing temp zip file: %s", err.Error())
		return
	}

	expectFile := zipTestdataFile("exported.zip")
	expectBytes, err := ioutil.ReadFile(expectFile)
	if err != nil {
		t.Errorf("error reading expected bytes: %s", err.Error())
		return
	}
	if diff := cmp.Diff(expectBytes, buf.Bytes()); diff != "" {
		t.Errorf("byte mismatch (-want +got):\n%s", diff)
	}
}

// TODO(dustmop): Rewrite zip importing
//func TestUnzipDatasetBytes(t *testing.T) {
//	path := zipTestdataFile("exported.zip")
//	zipBytes, err := ioutil.ReadFile(path)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	dsp := &dataset.Dataset{}
//	if err := UnzipDatasetBytes(zipBytes, dsp); err != nil {
//		t.Error(err)
//	}
//}
//func TestUnzipDataset(t *testing.T) {
//	if err := UnzipDataset(bytes.NewReader([]byte{}), 0, &dataset.Dataset{}); err == nil {
//		t.Error("expected passing bad reader to error")
//	}
//
//	path := zipTestdataFile("exported.zip")
//	zipBytes, err := ioutil.ReadFile(path)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	dsp := &dataset.Dataset{}
//	if err := UnzipDataset(bytes.NewReader(zipBytes), int64(len(zipBytes)), dsp); err != nil {
//		t.Error(err)
//	}
//}

func TestUnzipGetContents(t *testing.T) {
	if _, err := UnzipGetContents([]byte{}); err == nil {
		t.Error("expected passing bad reader to error")
	}

	path := zipTestdataFile("exported.zip")
	zipBytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	res, err := UnzipGetContents(zipBytes)
	if err != nil {
		t.Error(err)
	}

	keys := getKeys(res)
	expectKeys := []string{"body.csv", "qri-ref.txt", "structure.json", "transform.json"}
	if diff := cmp.Diff(expectKeys, keys); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
