package dscache

import (
	"strings"
	"testing"

	"github.com/qri-io/qri/logbook"
)

// TODO(dlong): Test NewDscache, IsEmpty, Assign, ListRefs, Update

func TestNilCallable(t *testing.T) {
	var (
		cache *Dscache
		err   error
	)

	if !cache.IsEmpty() {
		t.Errorf("expected IsEmpty: got !IsEmpty")
	}
	if err = cache.Assign(&Dscache{}); err != ErrNoDscache {
		t.Errorf("expected '%s': got '%s'", ErrNoDscache, err)
	}
	if str := cache.VerboseString(true); !strings.Contains(str, "empty dscache") {
		t.Errorf("expected str to Contain 'empty dscache': got '%s'", str)
	}
	if _, err = cache.ListRefs(); err != ErrNoDscache {
		t.Errorf("expected '%s': got '%s'", ErrNoDscache, err)
	}
	if err = cache.Update(&logbook.Action{}); err != ErrNoDscache {
		t.Errorf("expected '%s': got '%s'", ErrNoDscache, err)
	}
}
