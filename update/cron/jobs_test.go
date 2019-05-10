package cron

import (
	"fmt"
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	cronfb "github.com/qri-io/qri/update/cron/cron_fbs"
)

func CompareJobSlices(a, b []*Job) error {
	if len(a) != len(b) {
		return fmt.Errorf("length mistmatch: %d != %d", len(a), len(b))
	}

	for i, jobA := range a {
		if err := CompareJobs(jobA, b[i]); err != nil {
			return fmt.Errorf("job index %d mistmatch: %s", i, err)
		}
	}

	return nil
}

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
