package cron

import (
	flatbuffers "github.com/google/flatbuffers/go"
	cronfb "github.com/qri-io/qri/update/cron/cron_fbs"
)

// jobs is a list of jobs that implements the sort.Interface, sorting a list
// of jobs in reverse-chronological-then-alphabetical order
type jobs []*Job

func (js jobs) Len() int { return len(js) }
func (js jobs) Less(i, j int) bool {
	if js[i].RunStart.Equal(js[j].RunStart) {
		return js[i].Name < js[j].Name
	}
	return js[i].RunStart.After(js[j].RunStart)
}
func (js jobs) Swap(i, j int) { js[i], js[j] = js[j], js[i] }

func (js jobs) FlatbufferBytes() []byte {
	builder := flatbuffers.NewBuilder(0)
	count := len(js)

	offsets := make([]flatbuffers.UOffsetT, count)
	for i, j := range js {
		offsets[i] = j.MarshalFlatbuffer(builder)
	}

	cronfb.JobsStartListVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	jsvo := builder.EndVector(count)

	cronfb.JobsStart(builder)
	cronfb.JobsAddList(builder, jsvo)
	off := cronfb.JobsEnd(builder)

	builder.Finish(off)
	return builder.FinishedBytes()
}

func unmarshalJobsFlatbuffer(data []byte) (js jobs, err error) {
	jsFb := cronfb.GetRootAsJobs(data, 0)
	dec := &cronfb.Job{}
	js = make(jobs, jsFb.ListLength())
	for i := 0; i < jsFb.ListLength(); i++ {
		jsFb.List(dec, i)
		js[i] = &Job{}
		if err := js[i].UnmarshalFlatbuffer(dec); err != nil {
			return nil, err
		}
	}
	return js, nil
}
