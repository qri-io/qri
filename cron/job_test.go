package cron

import (
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	cron "github.com/qri-io/qri/cron/cron_fbs"
)

func TestJobFb(t *testing.T) {
	jorbs := jobs{
		&Job{
			Name:        "job_one",
			Periodicity: mustRepeatingInterval("R/PT1H"),
			Type:        JTDataset,
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
		offsets[i] = j.MarshalFb(builder)
	}

	cron.JobsStartListVector(builder, len(jorbs))
	for i := len(jorbs) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	jsvo := builder.EndVector(len(jorbs))

	cron.JobsStart(builder)
	cron.JobsAddList(builder, jsvo)
	off := cron.JobsEnd(builder)

	builder.Finish(off)
	data := builder.FinishedBytes()

	js := cron.GetRootAsJobs(data, 0)
	dec := &cron.Job{}
	t.Log(js.ListLength())
	for i := 0; i < js.ListLength(); i++ {
		js.List(dec, i)
		decJob := &Job{}
		if err := decJob.UnmarshalFb(dec); err != nil {
			t.Error(err)
		}
		if err := CompareJobs(jorbs[i], decJob); err != nil {
			t.Error(err)
		}
	}
}
