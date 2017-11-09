package dsutil

// import (
// 	"archive/zip"
// 	// "os"
// 	// "path/filepath"

// 	"github.com/qri-io/dataset"
// 	"github.com/qri-io/fs/local"

// 	"testing"
// )

// func TestPackageDataset(t *testing.T) {
// 	// wd, err := os.Getwd()
// 	// if err != nil {
// 	// 	t.Errorf("error reading working directory: %s", err.Error())
// 	// 	return
// 	// }
// 	store := local.NewLocalStore("test_data")

// 	data, err := store.Read("local/b/dataset.json")
// 	if err != nil {
// 		t.Errorf("error reading b/dataset.json: %s", err.Error())
// 		return
// 	}

// 	ds := &dataset.Dataset{}
// 	if err := ds.UnmarshalJSON(data); err != nil {
// 		t.Errorf("error unmarshalling b/dataset.json: %s", err.Error())
// 		return
// 	}

// 	r, size, err := PackageDataset(store, ds)
// 	// r, size, err := ns.Package(dataset.NewAddress("local.b"))
// 	if err != nil {
// 		t.Errorf("error packaging dataset: %s", err.Error())
// 		return
// 	}

// 	zr, err := zip.NewReader(r, size)
// 	if err != nil {
// 		t.Errorf("error creating zip reader: %s", err.Error())
// 		return
// 	}

// 	for _, f := range zr.File {
// 		// fmt.Println(f.Name)
// 		rc, err := f.Open()
// 		if err != nil {
// 			t.Errorf("error opening file %s in package", f.Name)
// 			break
// 		}

// 		if err := rc.Close(); err != nil {
// 			t.Errorf("error closing file %s in package", f.Name)
// 			break
// 		}
// 	}
// }
