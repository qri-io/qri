package watchfs

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/event"
)

func TestFilesysWatcher(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "watchfs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := event.NewBus(ctx)

	var (
		wg  sync.WaitGroup
		got event.WatchfsChange
	)

	wg.Add(1)
	bus.Subscribe(func(_ context.Context, typ event.Type, payload interface{}) error {
		t.Logf("got event!")
		if typ != event.ETCreatedNewFile {
			t.Errorf("wrong event type. wanted: %q, got: %q", event.ETCreatedNewFile, typ)
		}
		got = payload.(event.WatchfsChange)
		wg.Done()
		return nil
	}, event.ETCreatedNewFile)

	// Create a directory, and watch it
	watchdir := filepath.Join(tmpdir, "watch_me")
	_ = os.Mkdir(watchdir, 0755)
	w, err := NewFilesysWatcher(ctx, bus)
	if err != nil {
		t.Error(err)
	}
	w.Watch(EventPath{
		Username: "test_peer",
		Dsname:   "ds_name",
		Path:     watchdir,
	})
	target := filepath.Join(watchdir, "body.csv")

	// Write a file to the watched directory, get event
	if err := ioutil.WriteFile(target, []byte("test"), os.FileMode(0644)); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	expect := event.WatchfsChange{
		Username:    "test_peer",
		Dsname:      "ds_name",
		Source:      target,
		Destination: "",
		Time:        got.Time,
	}
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("filesys event (-want +got):\n%s", diff)
	}
}
