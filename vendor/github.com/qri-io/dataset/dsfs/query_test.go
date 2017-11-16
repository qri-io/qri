package dsfs

import (
	"testing"

	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
)

func TestLoadAbstractQuery(t *testing.T) {
	store := memfs.NewMapstore()
	q := &dataset.AbstractQuery{Statement: "select * from whatever booooooo go home"}
	a, err := SaveAbstractQuery(store, q, true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	if _, err = LoadAbstractQuery(store, a); err != nil {
		t.Errorf(err.Error())
	}
	// TODO - other tests & stuff
}

func TestQueryLoadAbstractStructures(t *testing.T) {
	// store := datastore.NewMapDatastore()
	// TODO - finish dis test
}
