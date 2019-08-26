// Package cron schedules dataset and shell script updates
package cron

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/ioes"
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
	// ListJobs lists currently scheduled jobs
	ListJobs(ctx context.Context, offset, limit int) ([]*Job, error)
	// Job gets a single scheduled job by name
	Job(ctx context.Context, name string) (*Job, error)

	// Schedule adds a job to the scheduler for execution once every period
	Schedule(ctx context.Context, job *Job) error
	// Unschedule removes a job from the scheduler
	Unschedule(ctx context.Context, name string) error

	// ListLogs gives a log of executed jobs
	ListLogs(ctx context.Context, offset, limit int) ([]*Job, error)
	// Log returns a single executed job by job.LogName
	Log(ctx context.Context, logName string) (*Job, error)
	// JobLogFile returns a reader for a file at the given name
	LogFile(ctx context.Context, logName string) (io.ReadCloser, error)
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

// ListJobs proxies to the schedule store for reading jobs
func (c *Cron) ListJobs(ctx context.Context, offset, limit int) ([]*Job, error) {
	return c.schedule.ListJobs(ctx, offset, limit)
}

// Job proxies to the schedule store for reading a job by name
func (c *Cron) Job(ctx context.Context, name string) (*Job, error) {
	return c.schedule.Job(ctx, name)
}

// ListLogs returns a list of jobs that have been executed
func (c *Cron) ListLogs(ctx context.Context, offset, limit int) ([]*Job, error) {
	return c.log.ListJobs(ctx, offset, limit)
}

// Log gives a specific Job by logged job name
func (c *Cron) Log(ctx context.Context, logName string) (*Job, error) {
	return c.log.Job(ctx, logName)
}

// LogFile returns a reader for a file at the given name
func (c *Cron) LogFile(ctx context.Context, logName string) (io.ReadCloser, error) {
	job, err := c.log.Job(ctx, logName)
	if err != nil {
		return nil, err
	}

	if job.LogFilePath == "" {
		return ioutil.NopCloser(&bytes.Buffer{}), nil
	}

	// TODO (b5): if logs are being stored somewhere other than local this'll break
	// we should add an OpenLogFile method to LogFileCreator & rename the interface
	return os.Open(job.LogFilePath)
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
		jobs, err := c.schedule.ListJobs(ctx, 0, -1)
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
			log.Infof("running %d job(s)", len(run))
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
	job.RunStart = time.Now().In(time.UTC)

	streams := ioes.NewDiscardIOStreams()
	if lfc, ok := c.log.(LogFileCreator); ok {
		if file, logPath, err := lfc.CreateLogFile(job); err == nil {
			log.Debugf("using log file: %s", logPath)
			defer file.Close()
			streams = ioes.NewIOStreams(nil, file, file)
			job.LogFilePath = logPath
		}
	}

	if err := runner(ctx, streams, job); err != nil {
		log.Errorf("run job: %s error: %s", job.Name, err.Error())
		job.RunError = err.Error()
	} else {
		log.Debugf("run job: %s success", job.Name)
		job.RunError = ""
	}
	job.RunStop = time.Now().In(time.UTC)
	job.RunNumber++

	// the updated job that goes to the schedule store shouldn't have a log path
	scheduleJob := job.Copy()
	scheduleJob.LogFilePath = ""
	scheduleJob.RunStart = time.Time{}
	scheduleJob.RunStop = time.Time{}
	scheduleJob.PrevRunStart = job.RunStart
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

	// TODO (b5) - check for prior job & inherit the previous run number

	return c.schedule.PutJob(ctx, job)
}

// Unschedule removes a job from the cron scheduler, cancelling any future
// job executions
func (c *Cron) Unschedule(ctx context.Context, name string) error {
	return c.schedule.DeleteJob(ctx, name)
}
