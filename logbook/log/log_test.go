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
	op{},
	UserInit{},
	UserChange{},
	UserRename{},
	UserDelete{},
	NameInit{},
	NameChange{},
	NameDelete{},
	VersionSave{},
	VersionDelete{},
	VersionPublish{},
	VersionUnpublish{},
	ACLInit{},
	ACLUpdate{},
	ACLDelete{},
)

func TestLogsetFlatbuffer(t *testing.T) {
	everyOpLog := &Log{
		signature: nil,
		ops: []Operation{
			UserInit{
				op: op{
					ref: "QmHashOfSteveSPublicKey",
				},
				Username: "steve",
			},
		},
	}

	set := &Set{
		logs: map[string]*Log{
			"steve": everyOpLog,
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
