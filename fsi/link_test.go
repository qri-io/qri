package fsi

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	fsifb "github.com/qri-io/qri/fsi/fsi_fbs"
)

func TestLinkFlatbuffer(t *testing.T) {
	src := &Link{
		Ref:  "le_reference",
		Path: "/une/path",
	}

	data := src.FlatbufferBytes()

	link := fsifb.GetRootAsLink(data, 0)

	got := &Link{}
	got.UnmarshalFlatbuffer(link)

	if diff := cmp.Diff(src, got); diff != "" {
		t.Errorf("compare link mismatch(-want +got):\n%s", diff)
	}
}

func TestLinksFlatbuffer(t *testing.T) {
	ls := links{
		&Link{
			Ref:  "link_one",
			Path: "/path/one",
		},
		&Link{
			Ref:  "job_two",
			Path: "/path/two",
		},
	}

	// builder := flatbuffers.NewBuilder(0)
	data := ls.FlatbufferBytes()

	got, err := unmarshalLinksFlatbuffer(data)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(ls, got); diff != "" {
		t.Errorf("linklist mismatch(-want +got):\n%s", diff)
	}
}
