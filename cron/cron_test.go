package cron

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/ioes"
	"github.com/qri-io/iso8601"
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
	job := &Job{
		Name:        "b5/libp2p_node_count",
		Type:        JTDataset,
		Periodicity: mustRepeatingInterval("R/P1W"),
	}

	factory := func(outer context.Context) RunJobFunc {
		return func(ctx context.Context, streams ioes.IOStreams, job *Job) error {
			switch job.Type {
			case JTDataset:
				updateCount++
				// ds.Commit.Timestamp = time.Now()
				return nil
			}
			t.Fatalf("runner called with invalid job: %v", job)
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	cron := NewCronInterval(&MemJobStore{}, &MemJobStore{}, factory, time.Millisecond*50)
	if err := cron.Schedule(ctx, job); err != nil {
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

func TestCronShellScript(t *testing.T) {
	pdci := DefaultCheckInterval
	defer func() { DefaultCheckInterval = pdci }()
	DefaultCheckInterval = time.Millisecond * 50

	updateCount := 0

	job := &Job{
		Name:        "foo.sh",
		Type:        JTShellScript,
		Periodicity: mustRepeatingInterval("R/P1W"),
	}

	// scriptRunner := LocalShellScriptRunner("testdata")
	factory := func(outer context.Context) RunJobFunc {
		return func(ctx context.Context, streams ioes.IOStreams, job *Job) error {
			switch job.Type {
			case JTShellScript:
				updateCount++
				// return scriptRunner(ctx, streams, job)
				return nil
			}
			t.Fatalf("runner called with invalid job: %v", job)
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	cron := NewCron(&MemJobStore{}, &MemJobStore{}, factory)
	if err := cron.Schedule(ctx, job); err != nil {
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
