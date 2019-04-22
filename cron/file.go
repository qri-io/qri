package cron

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"

	"github.com/ugorji/go/codec"
)

// NewFileJobStore creates a job store that persists to a CBOR file
// specified at path
func NewFileJobStore(path string) JobStore {
	return &FileJobStore{
		path: path,
	}
}

// FileJobStore is a jobstore implementation that saves to a CBOR file
// Jobs stored in FileJobStore can be persisted for the duration of a process
// at the longest.
// FileJobStore is safe for concurrent use
type FileJobStore struct {
	lock sync.Mutex
	path string
}

// Jobs lists jobs currently in the store
func (s *FileJobStore) Jobs(offset, limit int) ([]*Job, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	js, err := s.loadJobs()
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = len(js)
	}

	ss := make([]*Job, limit)
	added := 0
	for i, job := range js {
		if i < offset {
			continue
		} else if added == limit {
			break
		}

		ss[added] = job
		added++
	}
	return ss[:added], nil
}

func (s *FileJobStore) handle() codec.Handle {
	return &codec.CborHandle{
		// Need to use RFC3339 timestamps to preserve as much precision as possible
		TimeRFC3339: true,
	}
}

func (s *FileJobStore) loadJobs() (js jobs, err error) {
	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return jobs{}, nil
	}

	js = jobs{}
	err = codec.NewDecoder(f, s.handle()).Decode(&js)
	return
}

func (s *FileJobStore) saveJobs(jobs []*Job) error {
	buf := &bytes.Buffer{}
	if err := codec.NewEncoder(buf, s.handle()).Encode(jobs); err != nil {
		return err
	}
	return ioutil.WriteFile(s.path, buf.Bytes(), os.ModePerm)
}

// Job gets job details from the store by name
func (s *FileJobStore) Job(name string) (*Job, error) {
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

// PutJob places a job in the store. If the job name matches the name of a job
// that already exists, it will be overwritten with the new job
func (s *FileJobStore) PutJob(job *Job) error {
	if err := ValidateJob(job); err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	js, err := s.loadJobs()
	if err != nil {
		return err
	}

	for i, j := range js {
		if job.Name == j.Name {
			js[i] = job

			sort.Sort(js)
			return s.saveJobs(js)
		}
	}

	js = append(js, job)
	sort.Sort(js)
	return s.saveJobs(js)
}

// DeleteJob removes a job from the store by name. deleting a non-existent job
// won't return an error
func (s *FileJobStore) DeleteJob(name string) error {
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

// Destroy removes the path entirely
func (s *FileJobStore) Destroy() error {
	return os.Remove(s.path)
}
