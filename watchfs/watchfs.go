package watchfs

import (
	"context"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/event"
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
	Watcher *fsnotify.Watcher
	Sender  chan FilesysEvent
	Assoc   map[string]EventPath
}

// NewFilesysWatcher returns a new FilesysWatcher
func NewFilesysWatcher(ctx context.Context, bus event.Bus) *FilesysWatcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	w := FilesysWatcher{Watcher: watcher}
	if bus != nil {
		bus.Subscribe(w.eventHandler, event.ETFSICreateLinkEvent)
	}
	return &w
}

func (w *FilesysWatcher) eventHandler(ctx context.Context, t event.Type, payload interface{}) error {
	switch t {
	case event.ETFSICreateLinkEvent:
		go func() {
			if fce, ok := payload.(event.FSICreateLinkEvent); ok {
				log.Debugf("received link event. adding watcher for path: %s", fce.FSIPath)
				w.Add(EventPath{
					Path:     fce.FSIPath,
					Username: fce.Username,
					Dsname:   fce.Dsname,
				})
			}
		}()
	}

	return nil
}

// Begin will start watching the given directory paths
func (w *FilesysWatcher) Begin(paths []EventPath) chan FilesysEvent {
	// Associate paths with additional information
	assoc := make(map[string]EventPath)

	for _, p := range paths {
		err := w.Watcher.Add(p.Path)
		if err != nil {
			log.Errorf("%s", err)
		}
		assoc[p.Path] = p
	}

	messages := make(chan FilesysEvent)
	w.Sender = messages
	w.Assoc = assoc

	// Dispatch filesystem events
	go func() {
		for {
			select {
			case event, ok := <-w.Watcher.Events:
				if !ok {
					log.Debugf("error getting event")
					continue
				}

				if event.Op == fsnotify.Chmod {
					// Don't care about CHMOD, skip it
					continue
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					w.sendEvent(ModifyFileEvent, event.Name, "")
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					w.sendEvent(CreateNewFileEvent, event.Name, "")
				}
			}
		}
	}()

	return messages
}

// Add starts watching an additional path
func (w *FilesysWatcher) Add(path EventPath) {
	w.Assoc[path.Path] = path
	w.Watcher.Add(path.Path)
}

// sendEvent sends a message on the channel about an event
func (w *FilesysWatcher) sendEvent(etype EventType, sour, dest string) {
	log.Debugf("filesystem event %q %s -> %s\n", etype, sour, dest)
	dir := filepath.Dir(sour)
	ep := w.Assoc[dir]
	event := FilesysEvent{
		Type:        etype,
		Username:    ep.Username,
		Dsname:      ep.Dsname,
		Source:      sour,
		Destination: dest,
		Time:        time.Now(),
	}
	w.Sender <- event
}
