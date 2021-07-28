package watchfs

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
)

var log = golog.Logger("watchfs")

// EventPath stores information about a path that is capable of generating events
type EventPath struct {
	Path     string
	Username string
	Dsname   string
}

// FilesysWatcher will watch a set of directory paths, and send messages on a channel for events
// concerning those paths. These events are:
// * A new file in one of those paths was created
// * An existing file was modified / written to
// * An existing file was deleted
// * One of the folders being watched was renamed, but that folder is still being watched
// * One of the folders was removed, which makes it no longer watched
// TODO(dlong): Folder rename and remove are not supported yet.
type FilesysWatcher struct {
	watcher *fsnotify.Watcher
	assoc   map[string]EventPath
	bus     event.Bus
}

// NewFilesysWatcher returns a new FilesysWatcher
func NewFilesysWatcher(ctx context.Context, bus event.Bus) (*FilesysWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if bus == nil {
		return nil, fmt.Errorf("bus is required")
	}

	w := &FilesysWatcher{
		assoc:   map[string]EventPath{},
		watcher: watcher,
		bus:     bus,
	}

	bus.SubscribeTypes(w.eventHandler,
		event.ETFSICreateLink,
	)

	// Dispatch filesystem events
	go func() {
		for {
			select {
			case fsEvent, ok := <-w.watcher.Events:
				if !ok {
					log.Debugf("error getting event")
					continue
				}
				if fsEvent.Op == fsnotify.Chmod {
					// Don't care about CHMOD, skip it
					continue
				}
				if fsEvent.Op&fsnotify.Write == fsnotify.Write {
					w.publishEvent(event.ETModifiedFile, fsEvent.Name, "")
				}
				if fsEvent.Op&fsnotify.Create == fsnotify.Create {
					w.publishEvent(event.ETCreatedNewFile, fsEvent.Name, "")
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return w, nil
}

func (w *FilesysWatcher) eventHandler(ctx context.Context, e event.Event) error {
	switch e.Type {
	case event.ETFSICreateLink:
		go func() {
			if fce, ok := e.Payload.(event.FSICreateLink); ok {
				log.Debugf("received link event. adding watcher for path: %s", fce.FSIPath)
				w.Watch(EventPath{
					Path:     fce.FSIPath,
					Username: fce.Username,
					Dsname:   fce.Name,
				})
			}
		}()
	}

	return nil
}

// WatchAllFSIPaths sets up filesystem watchers for all FSI-linked datasets in
// a given qri repo
func (w *FilesysWatcher) WatchAllFSIPaths(ctx context.Context, r repo.Repo) error {
	refs, err := r.References(0, 100)
	if err != nil {
		return err
	}
	// Extract fsi paths for all working directories.
	paths := make([]EventPath, 0, len(refs))
	for _, ref := range refs {
		if ref.FSIPath != "" {
			paths = append(paths, EventPath{
				Path:     ref.FSIPath,
				Username: ref.Peername,
				Dsname:   ref.Name,
			})
		}
	}

	w.watchPaths(paths)

	return nil
}

// Begin will start watching the given directory paths
func (w *FilesysWatcher) watchPaths(paths []EventPath) {
	for _, p := range paths {
		err := w.watcher.Add(p.Path)
		if err != nil {
			log.Errorf("%s", err)
		}
		w.assoc[p.Path] = p
	}
}

// Watch starts watching an additional path
func (w *FilesysWatcher) Watch(path EventPath) {
	w.assoc[path.Path] = path
	w.watcher.Add(path.Path)
}

// publishEvent sends a message on the channel about an event
func (w *FilesysWatcher) publishEvent(typ event.Type, sour, dest string) {
	if w.filterSource(sour) {
		log.Debugf("filesystem event %q %s -> %s\n", typ, sour, dest)

		dir := filepath.Dir(sour)
		ep := w.assoc[dir]
		event := event.WatchfsChange{
			Username:    ep.Username,
			Dsname:      ep.Dsname,
			Source:      sour,
			Destination: dest,
			Time:        time.Now(),
		}

		go func() {
			if err := w.bus.Publish(context.Background(), typ, event); err != nil {
				log.Error(err)
			}
		}()
	}
}

func (w *FilesysWatcher) filterSource(sourceFile string) bool {
	return component.IsKnownFilename(sourceFile, component.GetKnownFilenames())
}
