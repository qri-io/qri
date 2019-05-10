package cron

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFbJobStore(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "TestFsJobStore")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	newStore := func() JobStore {
		return NewFlatbufferJobStore(filepath.Join(tmp, "jobs.dat"))
	}
	RunJobStoreTests(t, newStore)
}

func BenchmarkFbJobStore(b *testing.B) {
	ctx := context.Background()
	js := make(jobs, 1000)
	for i := range js {
		js[i] = &Job{
			Name:        fmt.Sprintf("job_%d", i),
			Type:        JTDataset,
			Periodicity: mustRepeatingInterval("R/P1H"),
		}
	}

	tmp, err := ioutil.TempDir(os.TempDir(), "TestFsJobStore")
	if err != nil {
		b.Fatal(err)
	}

	defer os.RemoveAll(tmp)
	store := NewFlatbufferJobStore(filepath.Join(tmp, "jobs.dat"))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := store.PutJobs(ctx, js...); err != nil {
			b.Fatal(err)
		}
		if _, err := store.ListJobs(ctx, 0, -1); err != nil {
			b.Fatal(err)
		}
	}
}
