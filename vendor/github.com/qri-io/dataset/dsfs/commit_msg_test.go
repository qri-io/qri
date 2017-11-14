package dsfs

import (
	"testing"

	"github.com/qri-io/cafs/memfs"
)

func TestLoadCommitMsg(t *testing.T) {
	store := memfs.NewMapstore()
	a, err := SaveCommitMsg(store, AirportCodesCommitMsg, true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	if _, err := LoadCommitMsg(store, a); err != nil {
		t.Errorf(err.Error())
	}
	// TODO - other tests & stuff
}
