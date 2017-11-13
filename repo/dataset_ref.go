package repo

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
)

// DatasetRef encapsulates a reference to a dataset. This needs to exist to bind
// ways of referring to a dataset to a dataset itself, as datasets can't easily
// contain their own hash information, and names are unique on a per-repository
// basis.
// It's tempting to think this needs to be "bigger", supporting more fields,
// keep in mind that if the information is important at all, it should
// be stored as metadata within the dataset itself.
type DatasetRef struct {
	// The dataset being referenced
	Dataset *dataset.Dataset `json:"dataset"`
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path datastore.Key `json:"path"`
}

// CompareDatasetRef compares two Dataset References, returning an error
// describing any difference between the two references
func CompareDatasetRef(a, b *DatasetRef) error {
	if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("nil mismatch: %v != %v", a, b)
	}
	if a == nil && b == nil {
		return nil
	}

	if a.Name != b.Name {
		return fmt.Errorf("name mismatch. %s != %s", a.Name, b.Name)
	}
	if !a.Path.Equal(b.Path) {
		return fmt.Errorf("path mismatch. %s != %s", a.Path.String(), b.Path.String())
	}
	return nil
}
