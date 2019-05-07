package cron

import (
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	cronfb "github.com/qri-io/qri/cron/cron_fbs"
)

func TestJobsFlatbuffer(t *testing.T) {
	jorbs := jobs{
		&Job{
			Name:        "job_one",
			Periodicity: mustRepeatingInterval("R/PT1H"),
			Type:        JTDataset,
			Options:     &DatasetOptions{Title: "Yus"},
		},
		&Job{
			Name:        "job_two",
			Periodicity: mustRepeatingInterval("R/PT1D"),
			Type:        JTShellScript,
		},
	}

	builder := flatbuffers.NewBuilder(0)
	offsets := make([]flatbuffers.UOffsetT, len(jorbs))
	for i, j := range jorbs {
		offsets[i] = j.MarshalFlatbuffer(builder)
	}

	cronfb.JobsStartListVector(builder, len(jorbs))
	for i := len(jorbs) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	jsvo := builder.EndVector(len(jorbs))

	cronfb.JobsStart(builder)
	cronfb.JobsAddList(builder, jsvo)
	off := cronfb.JobsEnd(builder)

	builder.Finish(off)
	data := builder.FinishedBytes()

	js := cronfb.GetRootAsJobs(data, 0)
	dec := &cronfb.Job{}
	t.Log(js.ListLength())
	for i := 0; i < js.ListLength(); i++ {
		js.List(dec, i)
		decJob := &Job{}
		if err := decJob.UnmarshalFlatbuffer(dec); err != nil {
			t.Error(err)
		}
		if err := CompareJobs(jorbs[i], decJob); err != nil {
			t.Error(err)
		}
	}
}
