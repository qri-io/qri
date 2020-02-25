package event

import (
	"sync"
)

// Sync counts events to allow a scope to wait for published events to finish being processed
type Sync struct {
	wg     sync.WaitGroup
	topics map[string]bool
	owner  *bus
	// TODO(dustmop): In future, use an error_collector instead
	err    error
}

// Outstanding increments the number of events that have been published
func (s *Sync) Outstanding(topic Topic, numSubs int) {
	if s.owner == nil {
		return
	}
	s.topics[string(topic)] = true
	s.wg.Add(numSubs)
}

// Finish counts that an event as finished being processed
func (s *Sync) Finish(topic Topic, err error) {
	if s.owner == nil {
		return
	}
	if _, ok := s.topics[string(topic)]; ok {
		// TODO(dustmop): In future, use an error_collector instead
		if err != nil {
			s.err = err
		}
		s.wg.Done()
	}
}

// Wait will block until all published events have been processed
func (s *Sync) Wait() error {
	if s == nil || s.owner == nil {
		return nil
	}
	s.wg.Wait()
	if s.owner != nil {
		s.owner.removeSync(s)
		s.owner = nil
	}
	return s.err
}
