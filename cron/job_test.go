package cron

import (
	"fmt"
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	cron "github.com/qri-io/qri/cron/cron_fbs"
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

func CompareJobs(a, b *Job) error {
	if a.Name != b.Name {
		return fmt.Errorf("Name mismatch. %s != %s", a.Name, b.Name)
	}
	if a.Periodicity != b.Periodicity {
		return fmt.Errorf("Periodicity mismatch. %s != %s", a.Name, b.Name)
	}
	// use unix comparisons to ignore millisecond & nanosecond precision errors
	if a.LastRun.Unix() != b.LastRun.Unix() {
		return fmt.Errorf("LastRun mismatch. %s != %s", a.LastRun, b.LastRun)
	}
	if a.Type != b.Type {
		return fmt.Errorf("Type mistmatch. %s != %s", a.Type, b.Type)
	}
	return nil
}

func TestJobFlatbuffer(t *testing.T) {
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
		offsets[i] = j.MarshalFlatbuffer(builder)
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
		if err := decJob.UnmarshalFlatbuffer(dec); err != nil {
			t.Error(err)
		}
		if err := CompareJobs(jorbs[i], decJob); err != nil {
			t.Error(err)
		}
	}
}
