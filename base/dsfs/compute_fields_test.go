package dsfs

import (
	"context"
	"sync"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/event"
)

func TestComputeFieldsFile(t *testing.T) {
	var lk sync.Mutex
	ds := &dataset.Dataset{
		Commit: &dataset.Commit{},
		Structure: &dataset.Structure{
			Format:      dataset.NDJSONDataFormat.String(),
			Compression: "zst",
			Schema:      dataset.BaseSchemaArray,
		},
	}

	ds.SetBodyFile(qfs.NewMemfileBytes(ds.Structure.BodyFilename(), []byte("[0,1,2]\n[3,4,5]")))
	cff, err := newComputeFieldsFile(context.Background(), &lk, nil, event.NilBus, nil, ds, nil, SaveSwitches{})
	if err != nil {
		t.Fatal(err)
	}

	expect := "/body.ndjson.zst"
	if expect != cff.FileName() {
		t.Errorf("unexpected filename. want: %q got %q", expect, cff.FileName())
	}
}
