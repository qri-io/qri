package cron

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/iso8601"
	"github.com/qri-io/qfs"
)

func mustRepeatingInterval(s string) iso8601.RepeatingInterval {
	ri, err := iso8601.ParseRepeatingInterval(s)
	if err != nil {
		panic(err)
	}
	return ri
}

func TestCronDataset(t *testing.T) {
	updateCount := 0
	ds := &dataset.Dataset{
		Peername: "b5",
		Name:     "libp2p_node_count",
		Commit: &dataset.Commit{
			// last update was Jan 1 2019
			Timestamp: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Meta: &dataset.Meta{
			// update once a week
			AccrualPeriodicity: "R/P1W",
		},
	}

	runner := func(ctx context.Context, streams ioes.IOStreams, job *Job) error {
		switch job.Type {
		case JTDataset:
			updateCount++
			ds.Commit.Timestamp = time.Now()
			return nil
		}
		t.Fatalf("runner called with invalid job: %v", job)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	cron := NewCronInterval(&MemJobStore{}, runner, time.Millisecond*50)
	cron.ScheduleDataset(ctx, ds, "", nil)

	if err := cron.Start(ctx); err != nil {
		t.Fatal(err)
	}

	<-ctx.Done()

	expectedUpdateCount := 1
	if expectedUpdateCount != updateCount {
		t.Errorf("update ran wrong number of times. expected: %d, got: %d", expectedUpdateCount, updateCount)
	}
}

func TestCronShellScript(t *testing.T) {
	pdci := DefaultCheckInterval
	defer func() { DefaultCheckInterval = pdci }()
	DefaultCheckInterval = time.Millisecond * 50

	updateCount := 0

	scriptRunner := LocalShellScriptRunner("testdata")
	runner := func(ctx context.Context, streams ioes.IOStreams, job *Job) error {
		switch job.Type {
		case JTShellScript:
			updateCount++
			return scriptRunner(ctx, streams, job)
		}
		t.Fatalf("runner called with invalid job: %v", job)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	cron := NewCron(&MemJobStore{}, runner)
	_, err := cron.ScheduleShellScript(ctx, qfs.NewMemfileBytes("test.sh", nil), "R/P1W", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := cron.Start(ctx); err != nil {
		t.Fatal(err)
	}

	<-ctx.Done()

	expectedUpdateCount := 1
	if expectedUpdateCount != updateCount {
		t.Errorf("update ran wrong number of times. expected: %d, got: %d", expectedUpdateCount, updateCount)
	}
}

func TestCronHTTP(t *testing.T) {
	s := &MemJobStore{}

	runner := func(ctx context.Context, streams ioes.IOStreams, job *Job) error {
		return nil
	}

	cr := NewCron(s, runner)
	// TODO (b5) - how do we keep this from being a leaking goroutine?
	go cr.ServeHTTP(":7897")

	cliCtx := context.Background()
	cli := HTTPClient{Addr: ":7897"}
	jobs, err := cli.Jobs(cliCtx, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 0 {
		t.Error("expected 0 jobs")
	}

	ds := &dataset.Dataset{
		Peername: "b5",
		Name:     "libp2p_node_count",
		Commit: &dataset.Commit{
			// last update was Jan 1 2019
			Timestamp: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Meta: &dataset.Meta{
			// update once a week
			AccrualPeriodicity: "R/P1W",
		},
	}

	dsJob, err := cli.ScheduleDataset(cliCtx, ds, "", nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	jobs, err = cli.Jobs(cliCtx, 0, 0)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(jobs) != 1 {
		t.Error("expected len of jobs to equal 1")
	}

	if err := cli.Unschedule(cliCtx, dsJob.Name); err != nil {
		t.Fatal(err)
	}

	jobs, err = cli.Jobs(cliCtx, 0, 0)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(jobs) != 0 {
		t.Error("expected len of jobs to equal 0")
	}
}
