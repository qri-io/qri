package cron

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// NewFlatbufferJobStore creates a job store that persists to a file
func NewFlatbufferJobStore(path string) JobStore {
	return &FlatbufferJobStore{
		path: path,
	}
}

// FlatbufferJobStore is a jobstore implementation that saves to a file of
// flatbuffer bytes.
// FlatbufferJobStore is safe for concurrent use
type FlatbufferJobStore struct {
	lock sync.Mutex
	path string
}

// ListJobs lists jobs currently in the store
func (s *FlatbufferJobStore) ListJobs(ctx context.Context, offset, limit int) ([]*Job, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	js, err := s.loadJobs()
	if err != nil {
		return nil, err
	}

	if limit < 0 {
		limit = len(js)
	}

	ss := make([]*Job, 0, limit)
	for i, job := range js {
		if i < offset {
			continue
		} else if len(ss) == limit {
			break
		}

		ss = append(ss, job)
	}
	return ss, nil
}

func (s *FlatbufferJobStore) loadJobs() (js jobs, err error) {
	data, err := ioutil.ReadFile(s.path)
	if os.IsNotExist(err) {
		return jobs{}, nil
	}

	return unmarshalJobsFlatbuffer(data)
}

func (s *FlatbufferJobStore) saveJobs(js jobs) error {
	return ioutil.WriteFile(s.path, js.FlatbufferBytes(), os.ModePerm)
}

// Job gets job details from the store by name
func (s *FlatbufferJobStore) Job(ctx context.Context, name string) (*Job, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	js, err := s.loadJobs()
	if err != nil {
		return nil, err
	}

	for _, job := range js {
		if job.Name == name {
			return job, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

// PutJobs places one or more jobs in the store. Putting a job who's name
// already exists must overwrite the previous job, making all job names unique
func (s *FlatbufferJobStore) PutJobs(ctx context.Context, add ...*Job) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	js, err := s.loadJobs()
	if err != nil {
		return err
	}

ADD:
	for _, job := range add {
		if err := job.Validate(); err != nil {
			return err
		}

		// TODO (b5) - avoid  O(n^2) ish complexity here using a better backing
		// data structure
		for i, j := range js {
			if job.Name == j.Name {
				js[i] = job
				continue ADD
			}
		}

		js = append(js, job)
	}

	sort.Sort(js)
	return s.saveJobs(js)
}

// PutJob places a job in the store. If the job name matches the name of a job
// that already exists, it will be overwritten with the new job
func (s *FlatbufferJobStore) PutJob(ctx context.Context, job *Job) error {
	return s.PutJobs(ctx, job)
}

// DeleteJob removes a job from the store by name. deleting a non-existent job
// won't return an error
func (s *FlatbufferJobStore) DeleteJob(ctx context.Context, name string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	js, err := s.loadJobs()
	if err != nil {
		return err
	}

	for i, j := range js {
		if j.Name == name {
			if i+1 == len(js) {
				js = js[:i]
				break
			}

			js = append(js[:i], js[i+1:]...)
			break
		}
	}
	return s.saveJobs(js)
}

const logsDirName = "logfiles"

// CreateLogFile creates a log file in the specified logs directory
func (s *FlatbufferJobStore) CreateLogFile(j *Job) (f io.WriteCloser, path string, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var logsDir string
	if logsDir, err = s.logsDir(); err != nil {
		return
	}
	path = filepath.Join(logsDir, j.LogName()+".log")

	f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	return
}

func (s *FlatbufferJobStore) logsDir() (string, error) {
	path := filepath.Join(filepath.Dir(s.path), logsDirName)
	err := os.MkdirAll(path, os.ModePerm)
	return path, err
}

// Destroy removes the path entirely
func (s *FlatbufferJobStore) Destroy() error {
	os.RemoveAll(filepath.Join(filepath.Dir(s.path), logsDirName))
	return os.Remove(s.path)
}
