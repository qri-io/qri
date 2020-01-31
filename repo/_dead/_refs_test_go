package repo

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRefsFlatbuffer(t *testing.T) {
	ls := Refs{
		DatasetRef{
			Name:    "ref_one",
			Path:    "/path/one",
			FSIPath: "/fsi/path/one",
		},
		DatasetRef{
			Name:    "ref_two",
			Path:    "/path/two",
			FSIPath: "/fsi/path/two",
		},
	}

	// builder := flatbuffers.NewBuilder(0)
	data := ls.FlatbufferBytes()

	got, err := UnmarshalRefsFlatbuffer(data)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(ls, got); diff != "" {
		t.Errorf("linklist mismatch(-want +got):\n%s", diff)
	}
}
