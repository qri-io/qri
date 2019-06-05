package cron

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/ioes"
)

func TestCronHTTP(t *testing.T) {
	s := &MemJobStore{}
	l := &MemJobStore{}

	factory := func(context.Context) RunJobFunc {
		return func(ctx context.Context, streams ioes.IOStreams, job *Job) error {
			return nil
		}
	}

	cliCtx := context.Background()
	cli := HTTPClient{Addr: ":7897"}
	if err := cli.Ping(); err != ErrUnreachable {
		t.Error("expected ping to server that is off to return ErrUnreachable")
	}

	cr := NewCron(s, l, factory)
	// TODO (b5) - how do we keep this from being a leaking goroutine?
	go cr.ServeHTTP(":7897")

	time.Sleep(time.Millisecond * 100)
	if err := cli.Ping(); err != nil {
		t.Errorf("expected ping to active server to not fail. got: %s", err)
	}

	jobs, err := cli.ListJobs(cliCtx, 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 0 {
		t.Error("expected 0 jobs")
	}

	dsJob := &Job{
		Name:        "b5/libp2p_node_count",
		Type:        JTDataset,
		Periodicity: mustRepeatingInterval("R/P1W"),
		RunStart:    time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if err = cli.Schedule(cliCtx, dsJob); err != nil {
		t.Fatal(err.Error())
	}

	jobs, err = cli.ListJobs(cliCtx, 0, -1)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(jobs) != 1 {
		t.Error("expected len of jobs to equal 1")
	}

	_, err = cli.Job(cliCtx, jobs[0].Name)
	if err != nil {
		t.Fatal(err.Error())
	}

	if err := cli.Unschedule(cliCtx, dsJob.Name); err != nil {
		t.Fatal(err)
	}

	jobs, err = cli.ListJobs(cliCtx, 0, -1)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(jobs) != 0 {
		t.Error("expected len of jobs to equal 0")
	}
}
