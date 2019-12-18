package watchfs

import (
	"time"

	"github.com/fsnotify/fsnotify"
	golog "github.com/ipfs/go-log"
)

var log = golog.Logger("watchfs")

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
}

// NewFilesysWatcher returns a new FilesysWatcher
func NewFilesysWatcher() *FilesysWatcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	return &FilesysWatcher{Watcher: watcher}
}

// Begin will start watching the given directory paths
func (w *FilesysWatcher) Begin(directoryPaths []string) chan FilesysEvent {
	for _, p := range directoryPaths {
		err := w.Watcher.Add(p)
		if err != nil {
			log.Errorf("%s", err)
		}
	}

	messages := make(chan FilesysEvent)
	w.Sender = messages

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

// sendEvent sends a message on the channel about an event
func (w *FilesysWatcher) sendEvent(etype EventType, sour, dest string) {
	e := FilesysEvent{
		Type:        etype,
		Source:      sour,
		Destination: dest,
		Time:        time.Now(),
	}
	w.Sender <- e
}
