package log

import (
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/logbook/logfb"
)

var allowUnexported = cmp.AllowUnexported(
	Set{},
	Log{},
	Op{},
)

func TestAllOpsLogsetFlatbuffer(t *testing.T) {

	set := &Set{
		logs: map[string]*Log{
			"steve": &Log{
				signature: nil,
				ops: []Op{
					Op{
						Type:      OpTypeInit,
						Model:     0x0001,
						Ref:       "QmRefHash",
						Prev:      "QmPrevHash",
						Relations: []string{"a", "b", "c"},
						Name:      "steve",
						AuthorID:  "QmSteveHash",
						Timestamp: 1,
						Size:      2,
						Note:      "note!",
					},
				},
			},
		},
	}

	builder := flatbuffers.NewBuilder(0)
	off := set.MarshalFlatbuffer(builder)
	builder.Finish(off)

	data := builder.FinishedBytes()
	logsetfb := logfb.GetRootAsLogset(data, 0)

	got := &Set{}
	if err := got.UnmarshalFlatbuffer(logsetfb); err != nil {
		t.Fatalf("unmarshalling flatbuffer bytes: %s", err.Error())
	}

	if diff := cmp.Diff(set, got, allowUnexported); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
