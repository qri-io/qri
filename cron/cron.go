// Package cron schedules dataset and shell script updates
package cron

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
)

var (
	log = golog.Logger("cron")
	// max time represents is a date far in the future
	maxTime = time.Date(9999, 12, 31, 23, 59, 59, 9999, time.UTC)
	// DefaultCheckInterval is the frequency cron will check all stored jobs
	// for scheduled updates without any additional configuration. Qri recommends
	// not running updates more than once an hour for performance and storage
	// consumption reasons, making a check every 15 minutes a reasonable default
	DefaultCheckInterval = time.Minute * 15
)

// Job represents a "cron job" that can be scheduled for repeated execution at
// a specified Periodicity (time interval)
type Job struct {
	// Name of the Job
	Name      string
	Type      JobType
	LastRun   time.Time
	LastError string
	// an ISO 8601 periodicity string
	Periodicity string
	Secrets     map[string]string
}

// NextExec returns the next time execution horizion. If job periodicity is
// improperly configuged, the returned time will be maxTime, causing the job
// to never execute (or at least, not execute this century)
func (job *Job) NextExec() time.Time {
	// check if job needs running
	d := dataset.AccuralDuration(job.Periodicity)
	if d == 0 {
		return maxTime
	}
	return job.LastRun.Add(d)
}

// ReadJobs are functions for fetching a set of jobs. ReadJobs defines canoncial
// behavior for listing & fetching jobs
type ReadJobs interface {
	// Jobs should return the set of jobs sorted in reverse-chronological order
	// (newest first order) of the last time they were run. When two LastRun times
	// are equal, Jobs should alpha sort the names
	// passing a limit and offset of 0 must return the entire list of stored jobs
	Jobs(offset, limit int) ([]*Job, error)
	// Job gets a job by it's name. All job names in a set must be unique. It's
	// the job of the set backing ReadJobs functions to enforce uniqueness
	Job(name string) (*Job, error)
}

// RunJobFunc is a function for executing a job. Cron takes care of scheduling
// job execution, and delegates the work of executing a job to a RunJobFunc
// implementation.
type RunJobFunc func(ctx context.Context, streams ioes.IOStreams, job *Job) error

// LocalShellScriptRunner creates a script runner anchored at a local path
// The runner it wires operating sytsem command in/out/errour to the iostreams
// provided by RunJobFunc. All paths are in relation to the provided base path
// Commands are executed with access to the same enviornment variables as the
// process the runner is executing in
// The executing command blocks until completion
func LocalShellScriptRunner(basepath string) RunJobFunc {
	return func(ctx context.Context, streams ioes.IOStreams, job *Job) error {
		path := job.Name
		if qfs.PathKind(job.Name) == "local" {
			path = filepath.Join(basepath, path)
		}

		cmd := exec.Command(path)
		cmd.Dir = basepath
		cmd.Stderr = streams.ErrOut
		cmd.Stdout = streams.Out
		cmd.Stdin = streams.In
		return cmd.Run()
	}
}

// JobType is a type for distinguishing between two different kinds of jobs
// JobType should be used as a shorthand for defining how to execute a job
type JobType string

const (
	// JTDataset indicates a job that runs "qri update" on a dataset specified
	// by Job Name. The job periodicity is determined by the specified dataset's
	// Meta.AccrualPeriodicity field. LastRun should closely match the datasets's
	// latest Commit.Timestamp value
	JTDataset JobType = "dataset"
	// JTShellScript represents a shell script to be run locally, which might
	// update one or more datasets. A non-zero exit code from shell script
	// indicates the job failed to execute properly
	JTShellScript JobType = "shell"
)

// JobStore handles the persistence of Job details. JobStore implementations
// must be safe for concurrent use
type JobStore interface {
	// JobStores must implement the ReadJobs interface for fetching stored jobs
	ReadJobs
	// PutJob places a job in the store. Putting a job who's name already exists
	// must overwrite the previous job, making all job names unique
	PutJob(*Job) error
	// DeleteJob removes a job from the store
	DeleteJob(name string) error
}

// ValidateJob confirms a Job contains valid details for scheduling
func ValidateJob(job *Job) error {
	if job.Name == "" {
		return fmt.Errorf("name is required")
	}
	if job.Periodicity == "" {
		return fmt.Errorf("period is required")
	}
	if dataset.AccuralDuration(job.Periodicity) == 0 {
		return fmt.Errorf("periodicity '%s' is invalid", job.Periodicity)
	}
	if job.Type != JTDataset && job.Type != JTShellScript {
		return fmt.Errorf("invalid job type: %s", job.Type)
	}
	return nil
}

// NewCron creates a Cron with the default check interval
func NewCron(js JobStore, runner RunJobFunc) *Cron {
	return NewCronInterval(js, runner, DefaultCheckInterval)
}

// NewCronInterval creates a Cron with a custom check interval
func NewCronInterval(js JobStore, runner RunJobFunc, checkInterval time.Duration) *Cron {
	return &Cron{
		store:    js,
		interval: checkInterval,
		runner:   runner,
	}
}

// Cron coordinates the scheduling of running "jobs" at specified periodicities
// (intervals) with a provided job runner function
type Cron struct {
	store    JobStore
	interval time.Duration
	runner   RunJobFunc
}

// assert cron implements ReadJobs at compile time
var _ ReadJobs = (*Cron)(nil)

// Jobs proxies to the underlying store for reading jobs
func (c *Cron) Jobs(offset, limit int) ([]*Job, error) {
	return c.store.Jobs(offset, limit)
}

// Job proxies to the underlying store for reading a job by name
func (c *Cron) Job(name string) (*Job, error) {
	return c.store.Job(name)
}

// Start initiates the check loop, looking for updates to execute once at every
// iteration of the configured check interval.
// Start blocks until the passed context completes
func (c *Cron) Start(ctx context.Context) error {
	t := time.NewTicker(c.interval)
	for {
		select {
		case <-t.C:
			go func() {
				jobs, err := c.store.Jobs(0, 0)
				if err != nil {
					log.Errorf("getting jobs from store: %s", err)
					return
				}

				for _, job := range jobs {
					go c.maybeRunJob(ctx, job)
				}
			}()
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Cron) maybeRunJob(ctx context.Context, job *Job) {
	if time.Now().After(job.NextExec()) {
		in := &bytes.Buffer{}
		out := &bytes.Buffer{}
		err := &bytes.Buffer{}
		streams := ioes.NewIOStreams(in, out, err)

		if err := c.runner(ctx, streams, job); err != nil {
			job.LastError = err.Error()
		} else {
			job.LastError = ""
		}
		job.LastRun = time.Now()
		c.store.PutJob(job)
	}
}

// ScheduleDataset adds a dataset to the cron scheduler
func (c *Cron) ScheduleDataset(ds *dataset.Dataset, secrets map[string]string) (*Job, error) {
	if ds.Meta == nil {
		return nil, fmt.Errorf("scheduling dataset updates requires a meta component with accrualPeriodicity set")
	}

	job := &Job{
		// TODO (b5) - dataset.Dataset needs an Alias() method:
		Name:        fmt.Sprintf("%s/%s", ds.Peername, ds.Name),
		Periodicity: ds.Meta.AccrualPeriodicity,
		Type:        JTDataset,
		Secrets:     secrets,
	}

	err := c.store.PutJob(job)
	return job, err
}

// ScheduleShellScript adds a shell script job type to the dataset
func (c *Cron) ScheduleShellScript(f qfs.File, periodicity string) (*Job, error) {
	job := &Job{
		Name:        f.FullPath(),
		Periodicity: periodicity,
		Type:        JTShellScript,
	}

	err := c.store.PutJob(job)
	return job, err
}

// Unschedule removes a job from the cron scheduler, cancelling any future
// job executions
func (c *Cron) Unschedule(name string) error {
	return c.store.DeleteJob(name)
}

// MemJobStore is an in-memory implementation of the JobStore interface
// Jobs stored in MemJobStore can be persisted for the duration of a process
// at the longest.
// MemJobStore is save for concurrent use
type MemJobStore struct {
	lock sync.Mutex
	jobs jobs
}

// Jobs lists jobs currently in the store
func (s *MemJobStore) Jobs(offset, limit int) ([]*Job, error) {
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
func (s *MemJobStore) Job(name string) (*Job, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, job := range s.jobs {
		if job.Name == name {
			return job, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

// PutJob places a job in the store. If the job name matches the name of a job
// that already exists, it will be overwritten with the new job
func (s *MemJobStore) PutJob(job *Job) error {
	if err := ValidateJob(job); err != nil {
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
func (s *MemJobStore) DeleteJob(name string) error {
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
	if js[i].LastRun.Equal(js[j].LastRun) {
		return js[i].Name < js[j].Name
	}
	return js[i].LastRun.After(js[j].LastRun)
}
func (js jobs) Swap(i, j int) { js[i], js[j] = js[j], js[i] }
