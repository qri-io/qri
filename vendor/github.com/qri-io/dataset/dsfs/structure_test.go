package dsfs

import (
	"testing"

	"github.com/qri-io/cafs/memfs"
)

func TestLoadStructure(t *testing.T) {
	store := memfs.NewMapstore()
	a, err := SaveStructure(store, AirportCodesStructure, true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	if _, err := LoadStructure(store, a); err != nil {
		t.Errorf(err.Error())
	}
	// TODO - other tests & stuff
}
