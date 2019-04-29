// Package cron schedules dataset and shell script updates
package cron

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"time"

	// flatbuffers "github.com/google/flatbuffers/go"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/iso8601"
	"github.com/qri-io/qfs"
	cronfb "github.com/qri-io/qri/cron/cron_fbs"
)

var (
	log = golog.Logger("cron")
	// DefaultCheckInterval is the frequency cron will check all stored jobs
	// for scheduled updates without any additional configuration. Qri recommends
	// not running updates more than once an hour for performance and storage
	// consumption reasons, making a check every 15 minutes a reasonable default
	DefaultCheckInterval = time.Minute * 15
)

// Scheduler is the "generic" interface for the Cron Scheduler, it's implemented
// by both Cron and HTTPClient for easier RPC communication
type Scheduler interface {
	Jobs(ctx context.Context, offset, limit int) ([]*Job, error)
	ScheduleDataset(ctx context.Context, ds *dataset.Dataset, periodicity string, opts *DatasetOptions) (*Job, error)
	ScheduleShellScript(ctx context.Context, f qfs.File, periodicity string, opts *ShellScriptOptions) (*Job, error)
	Unschedule(ctx context.Context, name string) error
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

// assert Cron implements ReadJobs at compile time
var _ ReadJobs = (*Cron)(nil)

// assert Cron is a Scheduler at compile time
var _ Scheduler = (*Cron)(nil)

// Jobs proxies to the underlying store for reading jobs
func (c *Cron) Jobs(ctx context.Context, offset, limit int) ([]*Job, error) {
	return c.store.Jobs(ctx, offset, limit)
}

// Job proxies to the underlying store for reading a job by name
func (c *Cron) Job(ctx context.Context, name string) (*Job, error) {
	return c.store.Job(ctx, name)
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
				jobs, err := c.store.Jobs(ctx, 0, 0)
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
		c.runJob(ctx, job)
	}
}

func (c *Cron) runJob(ctx context.Context, job *Job) {
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
	c.store.PutJob(ctx, job)
}

// ScheduleDataset adds a dataset to the cron scheduler
func (c *Cron) ScheduleDataset(ctx context.Context, ds *dataset.Dataset, periodicity string, opts *DatasetOptions) (*Job, error) {
	job, err := datasetToJob(ds, periodicity, opts)
	if err != nil {
		return nil, err
	}

	err = c.store.PutJob(ctx, job)
	return job, err
}

func datasetToJob(ds *dataset.Dataset, periodicity string, opts *DatasetOptions) (job *Job, err error) {
	if periodicity == "" && ds.Meta != nil && ds.Meta.AccrualPeriodicity != "" {
		periodicity = ds.Meta.AccrualPeriodicity
	}

	if periodicity == "" {
		return nil, fmt.Errorf("scheduling dataset updates requires a meta component with accrualPeriodicity set")
	}

	p, err := iso8601.ParseRepeatingInterval(periodicity)
	if err != nil {
		return nil, err
	}

	job = &Job{
		// TODO (b5) - dataset.Dataset needs an Alias() method:
		Name:        fmt.Sprintf("%s/%s", ds.Peername, ds.Name),
		Periodicity: p,
		Type:        JTDataset,
	}
	if opts != nil {
		job.Options = opts
	}
	err = job.Validate()

	return
}

// ScheduleShellScript adds a shell script job type to the dataset
func (c *Cron) ScheduleShellScript(ctx context.Context, f qfs.File, periodicity string, opts *ShellScriptOptions) (*Job, error) {
	job, err := shellScriptToJob(f, periodicity, opts)
	if err != nil {
		return nil, err
	}

	err = c.store.PutJob(ctx, job)
	return job, err
}

func shellScriptToJob(f qfs.File, periodicity string, opts *ShellScriptOptions) (job *Job, err error) {
	p, err := iso8601.ParseRepeatingInterval(periodicity)
	if err != nil {
		return nil, err
	}

	job = &Job{
		Name:        f.FullPath(),
		Periodicity: p,
		Type:        JTShellScript,
	}
	if opts != nil {
		job.Options = opts
	}
	err = job.Validate()

	return
}

// Unschedule removes a job from the cron scheduler, cancelling any future
// job executions
func (c *Cron) Unschedule(ctx context.Context, name string) error {
	return c.store.DeleteJob(ctx, name)
}

// ServeHTTP spins up an HTTP server at the specified address
func (c *Cron) ServeHTTP(addr string) error {
	s := &http.Server{
		Addr:    addr,
		Handler: newCronRoutes(c),
	}
	return s.ListenAndServe()
}

func newCronRoutes(c *Cron) http.Handler {

	m := http.NewServeMux()
	m.HandleFunc("/", c.statusHandler)
	m.HandleFunc("/jobs", c.jobsHandler)
	m.HandleFunc("/run", c.runHandler)

	return m
}

func (c *Cron) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (c *Cron) jobsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		js, err := c.Jobs(r.Context(), 0, 0)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(jobs(js).FlatbufferBytes())
		return

	case "POST":
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		j := cronfb.GetRootAsJob(data, 0)
		job := &Job{}
		if err := job.UnmarshalFlatbuffer(j); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		}

		if err := c.store.PutJob(r.Context(), job); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

	case "DELETE":
		name := r.FormValue("name")
		if err := c.Unschedule(r.Context(), name); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}

}

func (c *Cron) runHandler(w http.ResponseWriter, r *http.Request) {
	c.runJob(r.Context(), nil)
}
