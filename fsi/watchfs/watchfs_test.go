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
	reporef "github.com/qri-io/qri/repo/ref"
	repotest "github.com/qri-io/qri/repo/test"
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
	bus.SubscribeTypes(func(_ context.Context, e event.Event) error {
		t.Logf("got event!")
		if e.Type != event.ETCreatedNewFile {
			t.Errorf("wrong event type. wanted: %q, got: %q", event.ETCreatedNewFile, e.Type)
		}
		got = e.Payload.(event.WatchfsChange)
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
		Username: "test_peer_filesys_watcher",
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
		Username:    "test_peer_filesys_watcher",
		Dsname:      "ds_name",
		Source:      target,
		Destination: "",
		Time:        got.Time,
	}
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("filesys event (-want +got):\n%s", diff)
	}
}

func TestWatchAllFSIPaths(t *testing.T) {
	ctx := context.Background()

	// set up a repo w/ a dataset that has an FSIPath
	r, err := repotest.NewTestRepo()
	if err != nil {
		t.Fatal(err)
	}

	pro := r.Profiles().Owner()

	ref, err := r.GetRef(reporef.DatasetRef{
		Peername: pro.Peername,
		// name taken from repo/test_repo.go
		Name: "cities",
	})
	if err != nil {
		t.Fatal(err)
	}

	tmpdir, err := ioutil.TempDir("", "watchfs_watch_all_fsi_path")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	ref.FSIPath = tmpdir
	if err := r.PutRef(ref); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher, err := NewFilesysWatcher(ctx, event.NilBus)
	if err != nil {
		t.Fatal(err)
	}

	if err := watcher.WatchAllFSIPaths(ctx, r); err != nil {
		t.Fatal(err)
	}

	eventPath, ok := watcher.assoc[ref.FSIPath]
	if !ok {
		t.Errorf("expected watcher to have EventPath for path %q", ref.FSIPath)
		return
	}
	if eventPath.Dsname != ref.Name {
		t.Errorf("expected eventPath to have name %q, instead had name %q", ref.Name, eventPath.Dsname)
	}
	if eventPath.Username != ref.Peername {
		t.Errorf("exected eventPath to have username %q, instead had username %q", ref.Peername, eventPath.Username)
	}
	if eventPath.Path != ref.FSIPath {
		t.Errorf("expected eventPath to have path %q, instead had path %q", ref.FSIPath, eventPath.Path)
	}
}
