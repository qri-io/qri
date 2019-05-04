// Package cron schedules dataset and shell script updates
package cron

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	apiutil "github.com/datatogether/api/apiutil"
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

// Scheduler is the generic interface for the Cron Scheduler, it's implemented
// by both Cron and HTTPClient for easier RPC communication
type Scheduler interface {
	// Jobs lists currently scheduled jobs
	Jobs(ctx context.Context, offset, limit int) ([]*Job, error)
	// Job gets a single scheduled job by name
	Job(ctx context.Context, name string) (*Job, error)

	// Schedule adds a job to the scheduler for execution once every period
	Schedule(ctx context.Context, job *Job) error
	// Unschedule removes a job from the scheduler
	Unschedule(ctx context.Context, name string) error

	// Logs gives a log of executed jobs
	Logs(ctx context.Context, offset, limit int) ([]*Job, error)
	// LoggedJob returns a single executed job by job.LogName
	LoggedJob(ctx context.Context, logName string) (*Job, error)
	// JobLogFile returns a reader for a file at the given name
	LoggedJobFile(ctx context.Context, logName string) (io.ReadCloser, error)
}

// RunJobFunc is a function for executing a job. Cron takes care of scheduling
// job execution, and delegates the work of executing a job to a RunJobFunc
// implementation.
type RunJobFunc func(ctx context.Context, streams ioes.IOStreams, job *Job) error

// RunJobFactory is a function that returns a runner
type RunJobFactory func(ctx context.Context) (runner RunJobFunc)

// NewCron creates a Cron with the default check interval
func NewCron(schedule, log JobStore, factory RunJobFactory) *Cron {
	return NewCronInterval(schedule, log, factory, DefaultCheckInterval)
}

// NewCronInterval creates a Cron with a custom check interval
func NewCronInterval(schedule, log JobStore, factory RunJobFactory, checkInterval time.Duration) *Cron {
	return &Cron{
		schedule: schedule,
		log:      log,

		interval: checkInterval,
		factory:  factory,
	}
}

// Cron coordinates the scheduling of running jobs at specified periodicities
// (intervals) with a provided job runner function
type Cron struct {
	schedule JobStore
	log      JobStore
	interval time.Duration
	factory  RunJobFactory
}

// assert Cron is a Scheduler at compile time
var _ Scheduler = (*Cron)(nil)

// Jobs proxies to the schedule store for reading jobs
func (c *Cron) Jobs(ctx context.Context, offset, limit int) ([]*Job, error) {
	return c.schedule.Jobs(ctx, offset, limit)
}

// Job proxies to the schedule store for reading a job by name
func (c *Cron) Job(ctx context.Context, name string) (*Job, error) {
	return c.schedule.Job(ctx, name)
}

// Logs returns a list of jobs that have been executed
func (c *Cron) Logs(ctx context.Context, offset, limit int) ([]*Job, error) {
	return c.log.Jobs(ctx, offset, limit)
}

// LoggedJob gives a specific Job by logged job name
func (c *Cron) LoggedJob(ctx context.Context, logName string) (*Job, error) {
	return c.log.Job(ctx, logName)
}

// LoggedJobFile returns a reader for a file at the given name
func (c *Cron) LoggedJobFile(ctx context.Context, logName string) (io.ReadCloser, error) {
	// reader := c.log.
	// TODO (b5):
	return nil, fmt.Errorf("not finished")
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
		jobs, err := c.schedule.Jobs(ctx, 0, 0)
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
			log.Debugf("found %d job(s) to run", len(run))
			runner := c.factory(ctx)
			for _, job := range run {
				// TODO (b5) - if we want things like per-job timeout, we should create
				// a new job-scoped context here
				c.runJob(ctx, job, runner)
			}
		} else {
			log.Debugf("no jobs to run")
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
	job.LastRunStart = time.Now()

	streams := ioes.NewDiscardIOStreams()
	if lfc, ok := c.log.(LogFileCreator); ok {
		if file, logPath, err := lfc.CreateLogFile(job); err == nil {
			log.Debugf("using log file: %s", logPath)
			defer file.Close()
			streams = ioes.NewIOStreams(nil, file, file)
			job.LogFilePath = logPath
		}
	}

	streams.ErrOut.Write([]byte(fmt.Sprintf("%s %s\n", job.LastRunStart, job.Name)))

	if err := runner(ctx, streams, job); err != nil {
		log.Errorf("run job: %s error: %s", job.Name, err.Error())
		job.LastError = err.Error()
	} else {
		log.Debugf("run job: %s success", job.Name)
		job.LastError = ""
	}
	job.LastRunStop = time.Now()

	// the updated job that goes to the schedule store shouldn't have a log path
	scheduleJob := job.Copy()
	scheduleJob.LogFilePath = ""
	if err := c.schedule.PutJob(ctx, scheduleJob); err != nil {
		log.Error(err)
	}

	job.Name = job.LogName()
	if err := c.log.PutJob(ctx, job); err != nil {
		log.Error(err)
	}
}

// Schedule adds a job to the cron scheduler
func (c *Cron) Schedule(ctx context.Context, job *Job) error {
	if err := job.Validate(); err != nil {
		return err
	}

	return c.schedule.PutJob(ctx, job)
}

// Unschedule removes a job from the cron scheduler, cancelling any future
// job executions
func (c *Cron) Unschedule(ctx context.Context, name string) error {
	return c.schedule.DeleteJob(ctx, name)
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
	m.HandleFunc("/job", c.jobHandler)
	m.HandleFunc("/logs", c.logsHandler)
	m.HandleFunc("/log", c.loggedJobHandler)
	m.HandleFunc("/log/output", c.loggedJobFileHandler)
	m.HandleFunc("/run", c.runHandler)

	return m
}

func (c *Cron) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (c *Cron) jobsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// TODO (b5): handle these errors, but they'll default to 0 so it's mainly
		// for reporting when we're given odd values
		offset, _ := apiutil.ReqParamInt("offset", r)
		limit, _ := apiutil.ReqParamInt("limit", r)

		js, err := c.Jobs(r.Context(), offset, limit)
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
			return
		}

		if err := c.schedule.PutJob(r.Context(), job); err != nil {
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

func (c *Cron) jobHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	job, err := c.Job(r.Context(), name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(job.FlatbufferBytes())
}

func (c *Cron) logsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case "GET":
		// TODO (b5): handle these errors, but they'll default to 0 so it's mainly
		// for reporting when we're given odd values
		offset, _ := apiutil.ReqParamInt("offset", r)
		limit, _ := apiutil.ReqParamInt("limit", r)

		log, err := c.Logs(r.Context(), offset, limit)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(jobs(log).FlatbufferBytes())
		return

	}
}

func (c *Cron) loggedJobHandler(w http.ResponseWriter, r *http.Request) {
	logName := r.FormValue("log_name")
	job, err := c.LoggedJob(r.Context(), logName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(job.FlatbufferBytes())
}

func (c *Cron) loggedJobFileHandler(w http.ResponseWriter, r *http.Request) {
	logName := r.FormValue("log_name")
	f, err := c.LoggedJobFile(r.Context(), logName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	io.Copy(w, f)
	return
}

func (c *Cron) runHandler(w http.ResponseWriter, r *http.Request) {
	// TODO (b5): implement an HTTP run handler
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("not finished"))
	// c.runJob(r.Context(), nil)
}
