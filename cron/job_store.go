package cron

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"

	flatbuffers "github.com/google/flatbuffers/go"
	cronfb "github.com/qri-io/qri/cron/cron_fbs"
)

// ReadJobs are functions for fetching a set of jobs. ReadJobs defines canoncial
// behavior for listing & fetching jobs
type ReadJobs interface {
	// Jobs should return the set of jobs sorted in reverse-chronological order
	// (newest first order) of the last time they were run. When two LastRun times
	// are equal, Jobs should alpha sort the names
	// passing a limit and offset of 0 must return the entire list of stored jobs
	Jobs(ctx context.Context, offset, limit int) ([]*Job, error)
	// Job gets a job by it's name. All job names in a set must be unique. It's
	// the job of the set backing ReadJobs functions to enforce uniqueness
	Job(ctx context.Context, name string) (*Job, error)
}

// JobStore handles the persistence of Job details. JobStore implementations
// must be safe for concurrent use
type JobStore interface {
	// JobStores must implement the ReadJobs interface for fetching stored jobs
	ReadJobs
	// PutJob places one or more jobs in the store. Putting a job who's name
	// already exists must overwrite the previous job, making all job names unique
	PutJobs(context.Context, ...*Job) error
	// PutJob places a job in the store. Putting a job who's name already exists
	// must overwrite the previous job, making all job names unique
	PutJob(context.Context, *Job) error
	// DeleteJob removes a job from the store
	DeleteJob(ctx context.Context, name string) error
}

// LogFileCreator is an interface for generating log files to write to,
// JobStores should implement this interface
type LogFileCreator interface {
	// CreateLogFile returns a file to write output to
	CreateLogFile(job *Job) (f io.WriteCloser, path string, err error)
}

// MemJobStore is an in-memory implementation of the JobStore interface
// Jobs stored in MemJobStore can be persisted for the duration of a process
// at the longest.
// MemJobStore is safe for concurrent use
type MemJobStore struct {
	lock sync.Mutex
	jobs jobs
}

// Jobs lists jobs currently in the store
func (s *MemJobStore) Jobs(ctx context.Context, offset, limit int) ([]*Job, error) {
	if limit <= 0 {
		limit = len(s.jobs)
	}

	jobs := make([]*Job, limit)
	added := 0
	for i, job := range s.jobs {
		if i < offset {
			continue
		} else if added == limit {
			break
		}

		jobs[added] = job
		added++
	}
	return jobs[:added], nil
}

// Job gets job details from the store by name
func (s *MemJobStore) Job(ctx context.Context, name string) (*Job, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, job := range s.jobs {
		if job.Name == name {
			return job, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

// PutJobs places one or more jobs in the store. Putting a job who's name
// already exists must overwrite the previous job, making all job names unique
func (s *MemJobStore) PutJobs(ctx context.Context, js ...*Job) error {
	s.lock.Lock()
	defer func() {
		sort.Sort(s.jobs)
		s.lock.Unlock()
	}()

	for _, job := range js {
		if err := job.Validate(); err != nil {
			return err
		}

		for i, j := range s.jobs {
			if job.Name == j.Name {
				s.jobs[i] = job
				return nil
			}
		}

		s.jobs = append(s.jobs, job)
	}
	return nil
}

// PutJob places a job in the store. If the job name matches the name of a job
// that already exists, it will be overwritten with the new job
func (s *MemJobStore) PutJob(ctx context.Context, job *Job) error {
	if err := job.Validate(); err != nil {
		return err
	}

	s.lock.Lock()
	defer func() {
		sort.Sort(s.jobs)
		s.lock.Unlock()
	}()

	for i, j := range s.jobs {
		if job.Name == j.Name {
			s.jobs[i] = job
			return nil
		}
	}

	s.jobs = append(s.jobs, job)
	return nil
}

// DeleteJob removes a job from the store by name. deleting a non-existent job
// won't return an error
func (s *MemJobStore) DeleteJob(ctx context.Context, name string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i, j := range s.jobs {
		if j.Name == name {
			if i+1 == len(s.jobs) {
				s.jobs = s.jobs[:i]
				break
			}

			s.jobs = append(s.jobs[:i], s.jobs[i+1:]...)
			break
		}
	}
	return nil
}

// jobs is a list of jobs that implements the sort.Interface, sorting a list
// of jobs in reverse-chronological-then-alphabetical order
type jobs []*Job

func (js jobs) Len() int { return len(js) }
func (js jobs) Less(i, j int) bool {
	if js[i].LastRunStart.Equal(js[j].LastRunStart) {
		return js[i].Name < js[j].Name
	}
	return js[i].LastRunStart.After(js[j].LastRunStart)
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
