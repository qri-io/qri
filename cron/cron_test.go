package cron

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"

	"github.com/qri-io/qfs"
)

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
	cron.ScheduleDataset(ds, nil)

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
	_, err := cron.ScheduleShellScript(qfs.NewMemfileBytes("test.sh", nil), "R/P1W")
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

func TestMemStore(t *testing.T) {
	newStore := func() JobStore {
		return &MemJobStore{}
	}
	RunJobStoreTests(t, newStore)
}

func CompareJobs(a, b *Job) error {
	if a.Name != b.Name {
		return fmt.Errorf("Name mismatch. %s != %s", a.Name, b.Name)
	}
	if a.Periodicity != b.Periodicity {
		return fmt.Errorf("Periodicity mismatch. %s != %s", a.Name, b.Name)
	}
	// use unix comparisons to ignore millisecond & nanosecond precision errors
	if a.LastRun.Unix() != b.LastRun.Unix() {
		return fmt.Errorf("LastRun mismatch. %s != %s", a.LastRun, b.LastRun)
	}
	if a.Type != b.Type {
		return fmt.Errorf("Type mistmatch. %s != %s", a.Type, b.Type)
	}
	return nil
}

func CompareJobSlices(a, b []*Job) error {
	if len(a) != len(b) {
		return fmt.Errorf("length mistmatch: %d != %d", len(a), len(b))
	}

	for i, jobA := range a {
		if err := CompareJobs(jobA, b[i]); err != nil {
			return fmt.Errorf("job index %d mistmatch: %s", i, err)
		}
	}

	return nil
}

func RunJobStoreTests(t *testing.T, newStore func() JobStore) {
	t.Run("JobStoreTest", func(t *testing.T) {
		store := newStore()
		jobs, err := store.Jobs(0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 0 {
			t.Errorf("expected new store to contain no jobs")
		}

		jobOne := &Job{
			Name:        "job_one",
			Periodicity: "R/PT1H",
			Type:        JTDataset,
		}
		if err = store.PutJob(jobOne); err != nil {
			t.Errorf("putting job one: %s", err)
		}

		if jobs, err = store.Jobs(0, 0); err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 1 {
			t.Fatal("expected default get to return inserted job")
		}
		if err := CompareJobs(jobOne, jobs[0]); err != nil {
			t.Errorf("stored job mistmatch: %s", err)
		}

		jobTwo := &Job{
			Name:        "job two",
			Periodicity: "R/P3M",
			Type:        JTShellScript,
			LastRun:     time.Date(2001, 1, 1, 1, 1, 1, 1, time.UTC),
		}
		if err = store.PutJob(jobTwo); err != nil {
			t.Errorf("putting job one: %s", err)
		}

		if jobs, err = store.Jobs(0, 0); err != nil {
			t.Fatal(err)
		}
		expect := []*Job{jobTwo, jobOne}
		if err := CompareJobSlices(expect, jobs); err != nil {
			t.Error(err)
		}

		updatedJobOne := &Job{
			Name:        jobOne.Name,
			Periodicity: jobOne.Periodicity,
			Type:        jobOne.Type,
			LastRun:     time.Date(2002, 1, 1, 1, 1, 1, 1, time.UTC),
		}
		if err = store.PutJob(updatedJobOne); err != nil {
			t.Errorf("putting job one: %s", err)
		}

		if jobs, err = store.Jobs(1, 1); err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 1 {
			t.Fatal("expected limit 1 length to equal 1")
		}
		if err = CompareJobs(jobTwo, jobs[0]); err != nil {
			t.Error(err)
		}

		job, err := store.Job(updatedJobOne.Name)
		if err != nil {
			t.Fatal(err)
		}
		if err = CompareJobs(updatedJobOne, job); err != nil {
			t.Error(err)
		}

		if err = store.DeleteJob(updatedJobOne.Name); err != nil {
			t.Error(err)
		}
		if err = store.DeleteJob(jobTwo.Name); err != nil {
			t.Error(err)
		}

		if jobs, err = store.Jobs(0, 0); err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 0 {
			t.Error("expected deleted jobs to equal zero")
		}

		if dest, ok := store.(qfs.Destroyer); ok {
			if err := dest.Destroy(); err != nil {
				t.Log(err)
			}
		}
	})

	t.Run("TestJobStoreValidPut", func(t *testing.T) {
		bad := []struct {
			description string
			job         *Job
		}{
			{"empty", &Job{}},
			{"no name", &Job{Periodicity: "R/PT1H", Type: JTDataset}},
			{"no periodicity", &Job{Name: "some_name", Type: JTDataset}},
			{"no type", &Job{Name: "some_name", Periodicity: "R/PT1H"}},

			{"invalid periodicity", &Job{Name: "some_name", Periodicity: "wat", Type: JTDataset}},
			{"invalid JobType", &Job{Name: "some_name", Periodicity: "R/PT1H", Type: JobType("huh")}},
		}

		store := newStore()
		for i, c := range bad {
			if err := store.PutJob(c.job); err == nil {
				t.Errorf("bad case %d %s: expected error, got nil", i, c.description)
			}
		}
	})

	t.Run("TestJobStoreConcurrentUse", func(t *testing.T) {
		t.Skip("TODO (b5)")
	})
}
