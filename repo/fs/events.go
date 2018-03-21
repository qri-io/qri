package fsrepo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
)

// EventLog is a file-based implementation of the repo.EventLog interface
type EventLog struct {
	basepath
	file  File
	store cafs.Filestore
}

// NewEventLog allocates a new file-based EventLog instance
func NewEventLog(base string, file File, store cafs.Filestore) EventLog {
	return EventLog{basepath: basepath(base), file: file, store: store}
}

// LogEvent adds a Event to the store
func (ql EventLog) LogEvent(t repo.EventType, ref repo.DatasetRef) error {
	log, err := ql.logs()
	if err != nil {
		return err
	}

	e := &repo.Event{
		Time: time.Now(),
		Type: t,
		Ref:  ref,
	}
	log = append([]*repo.Event{e}, log...)
	sort.Slice(log, func(i, j int) bool { return log[i].Time.After(log[j].Time) })
	return ql.saveFile(log, ql.file)
}

// Events fetches a set of Events from the store
func (ql EventLog) Events(limit, offset int) ([]*repo.Event, error) {
	logs, err := ql.logs()
	if err != nil {
		return nil, err
	}

	if offset > len(logs) {
		offset = len(logs)
	}
	stop := limit + offset
	if stop > len(logs) {
		stop = len(logs)
	}

	return logs[offset:stop], nil
}

// EventsSince fetches a set of Events from the store that occur after a given timestamp
func (ql EventLog) EventsSince(t time.Time) ([]*repo.Event, error) {
	logs, err := ql.logs()
	if err != nil {
		return nil, err
	}

	events := make([]*repo.Event, 0, len(logs))
	for _, e := range logs {
		if e.Time.After(t) {
			events = append([]*repo.Event{e}, events...)
		}
	}

	return events, nil
}

func (ql *EventLog) logs() ([]*repo.Event, error) {
	ds := []*repo.Event{}
	data, err := ioutil.ReadFile(ql.filepath(ql.file))
	if err != nil {
		if os.IsNotExist(err) {
			return ds, nil
		}
		log.Debug(err.Error())
		return ds, fmt.Errorf("error loading logs: %s", err.Error())
	}

	if err := json.Unmarshal(data, &ds); err != nil {
		log.Debug(err.Error())
		return ds, fmt.Errorf("error unmarshaling logs: %s", err.Error())
	}
	return ds, nil
}
