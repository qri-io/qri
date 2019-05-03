// Package cron schedules dataset and shell script updates
package cron

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/ioes"
	cronfb "github.com/qri-io/qri/cron/cron_fbs"
)

var (
	log = golog.Logger("cron")
	// DefaultCheckInterval is the frequency cron will check all stored jobs
	// for scheduled updates without any additional configuration. Qri recommends
	// not running updates more than once an hour for performance and storage
	// consumption reasons, making a check every minute a reasonable default
	DefaultCheckInterval = time.Minute
)

// Scheduler is the "generic" interface for the Cron Scheduler, it's implemented
// by both Cron and HTTPClient for easier RPC communication
type Scheduler interface {
	Jobs(ctx context.Context, offset, limit int) ([]*Job, error)
	Job(ctx context.Context, name string) (*Job, error)
	Schedule(ctx context.Context, job *Job) error
	Unschedule(ctx context.Context, name string) error
}

// RunJobFunc is a function for executing a job. Cron takes care of scheduling
// job execution, and delegates the work of executing a job to a RunJobFunc
// implementation.
type RunJobFunc func(ctx context.Context, streams ioes.IOStreams, job *Job) error

// RunJobFactory is a function that returns a runner
type RunJobFactory func(ctx context.Context) (runner RunJobFunc)

// NewCron creates a Cron with the default check interval
func NewCron(js JobStore, factory RunJobFactory) *Cron {
	return NewCronInterval(js, factory, DefaultCheckInterval)
}

// NewCronInterval creates a Cron with a custom check interval
func NewCronInterval(js JobStore, factory RunJobFactory, checkInterval time.Duration) *Cron {
	return &Cron{
		store:    js,
		interval: checkInterval,
		factory:  factory,
	}
}

// Cron coordinates the scheduling of running "jobs" at specified periodicities
// (intervals) with a provided job runner function
type Cron struct {
	store    JobStore
	interval time.Duration
	factory  RunJobFactory
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
	check := func(ctx context.Context) {
		now := time.Now()
		ctx, cleanup := context.WithCancel(ctx)
		defer cleanup()

		log.Debugf("running check")
		jobs, err := c.store.Jobs(ctx, 0, 0)
		if err != nil {
			log.Errorf("getting jobs from store: %s", err)
			return
		}

		run := []*Job{}
		for _, job := range jobs {
			if now.After(job.NextExec()) {
				run = append(run, job)
			}
		}

		if len(run) > 0 {
			log.Errorf("found %d job(s) to run", len(run))
			runner := c.factory(ctx)
			for _, job := range run {
				// TODO (b5) - if we want things like per-job timeout, we should create
				// a new job-scoped context here
				c.runJob(ctx, job, runner)
			}
		} else {
			log.Errorf("no jobs to run")
		}
	}

	// initial call to check
	go check(ctx)

	t := time.NewTicker(c.interval)
	for {
		select {
		case <-t.C:
			go check(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Cron) runJob(ctx context.Context, job *Job, runner RunJobFunc) {
	log.Debugf("run job: %s", job.Name)
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	err := &bytes.Buffer{}
	streams := ioes.NewIOStreams(in, out, err)

	if err := runner(ctx, streams, job); err != nil {
		log.Errorf("run job: %s error: %s", job.Name, err.Error())
		job.LastError = err.Error()
	} else {
		log.Errorf("run job: %s success", job.Name)
		job.LastError = ""
	}
	job.LastRun = time.Now()
	c.store.PutJob(ctx, job)
}

// Schedule adds a job to the cron scheduler
func (c *Cron) Schedule(ctx context.Context, job *Job) error {
	if err := job.Validate(); err != nil {
		return err
	}

	return c.store.PutJob(ctx, job)
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
	// TODO (b5): implement an HTTP run handler
	w.WriteHeader(http.StatusTooEarly)
	w.Write([]byte("not finished"))
	// c.runJob(r.Context(), nil)
}
