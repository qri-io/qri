package cron

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"
)

// NewFbFileJobStore creates a job store that persists to a CBOR file
// specified at path
func NewFbFileJobStore(path string) JobStore {
	return &FbFileJobStore{
		path: path,
	}
}

// FbFileJobStore is a jobstore implementation that saves to a CBOR file
// Jobs stored in FbFileJobStore can be persisted for the duration of a process
// at the longest.
// FbFileJobStore is safe for concurrent use
type FbFileJobStore struct {
	lock sync.Mutex
	path string
}

// Jobs lists jobs currently in the store
func (s *FbFileJobStore) Jobs(offset, limit int) ([]*Job, error) {
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

func (s *FbFileJobStore) loadJobs() (js jobs, err error) {
	data, err := ioutil.ReadFile(s.path)
	if os.IsNotExist(err) {
		return jobs{}, nil
	}

	return unmarshalJobsFb(data)
}

func (s *FbFileJobStore) saveJobs(js jobs) error {
	return ioutil.WriteFile(s.path, js.MarshalFb(), os.ModePerm)
}

// Job gets job details from the store by name
func (s *FbFileJobStore) Job(name string) (*Job, error) {
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
func (s *FbFileJobStore) PutJobs(add ...*Job) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	js, err := s.loadJobs()
	if err != nil {
		return err
	}

	for _, job := range add {
		if err := ValidateJob(job); err != nil {
			return err
		}

		for i, j := range js {
			if job.Name == j.Name {
				js[i] = job
				return nil
			}
		}

		js = append(js, job)
	}

	sort.Sort(js)
	return s.saveJobs(js)
}

// PutJob places a job in the store. If the job name matches the name of a job
// that already exists, it will be overwritten with the new job
func (s *FbFileJobStore) PutJob(job *Job) error {
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
func (s *FbFileJobStore) DeleteJob(name string) error {
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
func (s *FbFileJobStore) Destroy() error {
	return os.Remove(s.path)
}
