package event

import (
	"sync"
)

// Sync counts events to allow a scope to wait for published events to finish being processed
type Sync struct {
	wg     sync.WaitGroup
	topics map[string]bool
}

// Outstanding increments the number of events that have been published
func (s *Sync) Outstanding(topic Topic) {
	s.topics[string(topic)] = true
	s.wg.Add(1)
}

// Finish counts that an event as finished being processed
func (s *Sync) Finish(topic Topic) bool {
	if _, ok := s.topics[string(topic)]; ok {
		s.wg.Done()
		return true
	}
	return false
}

// Wait will block until all published events have been processed
func (s *Sync) Wait() {
	s.wg.Wait()
}
