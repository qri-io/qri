package event

import (
	"context"
	"fmt"
	"sync"

	"testing"
)

const ETMainTestEvent = Topic("main:TestEvent")

func TestSync(t *testing.T) {
	ctx := context.Background()
	counter := syncCounter{}

	bus := NewBus(ctx)
	ch1 := bus.Subscribe(ETMainTestEvent)
	ch2 := bus.Subscribe(ETMainTestEvent)
	ch3 := bus.Subscribe(ETMainTestEvent)

	handleEvents(bus, ch1, nil, &counter)
	handleEvents(bus, ch2, nil, &counter)
	handleEvents(bus, ch3, nil, &counter)

	// Function which will publish an event, then wait for all subscribers to acknowledge it
	syncFunc(bus)

	if counter.Count != 3 {
		t.Errorf("expected Count to be 3, got %d", counter.Count)
	}
}

type syncCounter struct {
	Lock  sync.RWMutex
	Count int
}

func (c *syncCounter) Inc() {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.Count++
}

func syncFunc(bus Bus) error {
	s := bus.Synchronizer()
	bus.Publish(ETMainTestEvent, 1)
	return s.Wait()
}

func handleEvents(bus Bus, ch <-chan Event, err error, counter *syncCounter) {
	go func() {
		e := <-ch
		counter.Inc()
		bus.Acknowledge(e, err)
	}()
}

func TestAcknowledgeError(t *testing.T) {
	ctx := context.Background()
	bus := NewBus(ctx)

	s := bus.Synchronizer()
	e := Event{Topic: ETMainTestEvent}
	s.Outstanding(e.Topic, 1)
	bus.Acknowledge(e, fmt.Errorf("a test error"))
	err := s.Wait()
	expect := "a test error"
	if err == nil || err.Error() != expect {
		t.Errorf("expected to get error with message %q, got %q", expect, err)
	}
}

func TestHandleEventError(t *testing.T) {
	ctx := context.Background()
	counter := syncCounter{}

	bus := NewBus(ctx)
	ch1 := bus.Subscribe(ETMainTestEvent)

	handleEvents(bus, ch1, fmt.Errorf("a test error"), &counter)
	err := syncFunc(bus)

	expect := "a test error"
	if err == nil || err.Error() != expect {
		t.Errorf("expected to get error with message %q, got %q", expect, err)
	}
}
