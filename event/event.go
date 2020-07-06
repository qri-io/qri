// Package event implements an event bus.
// for a great introduction to the event bus pattern in go, see:
// https://levelup.gitconnected.com/lets-write-a-simple-event-bus-in-go-79b9480d8997
package event

import (
	"context"
	"fmt"
	"sync"

	golog "github.com/ipfs/go-log"
)

var (
	log = golog.Logger("event")

	// ErrBusClosed indicates the event bus is no longer coordinating events
	// because it's parent context has closed
	ErrBusClosed = fmt.Errorf("event bus is closed")
)

// Topic is the set of all topics emitted by the bus. Use the topic type to
// distinguish event names. Event emitters should declare Topics as constants
// and document the expected data payload type
type Topic string

// Handler is a function that will be called by the event bus whenever a
// subscribed topic is published. Handler calls are blocking, called in order
// of subscription. Any error returned by a handler is passed back to the
// event publisher.
// The handler context originates from the publisher, and in practice will often
// be scoped to a "request context" like an HTTP request or CLI command
// invocation.
// Generally, even handlers should aim to return quickly, and only delegate to
// goroutines when the publishing event is firing on a long-running process
type Handler func(ctx context.Context, t Topic, payload interface{}) error

// Publisher is an interface that can only publish an event
type Publisher interface {
	Publish(ctx context.Context, topic Topic, payload interface{}) error
}

// NilPublisher replaces a nil value, does nothing
type NilPublisher struct{}

// Publish does nothing with the event
func (n *NilPublisher) Publish(_ context.Context, _ Topic, _ interface{}) error {
	return nil
}

// Bus is a central coordination point for event publication and subscription
// zero or more subscribers register topics to be notified of, a publisher
// writes a topic event to the bus, which broadcasts to all subscribers of that
// topic
type Bus interface {
	// Publish an event to the bus
	Publish(ctx context.Context, t Topic, data interface{}) error
	// Subscribe to one or more topics with a handler function that will be called
	// whenever the event topic is published
	Subscribe(handler Handler, topics ...Topic)
	// NumSubscriptions returns the number of subscribers to the bus's events
	NumSubscribers() int
}

type bus struct {
	lk     sync.RWMutex
	closed bool
	subs   map[Topic][]Handler
}

// NewBus creates a new event bus. Event busses should be instantiated as a
// singleton. If the passed in context is cancelled, the bus will stop emitting
// events and close all subscribed channels
//
// TODO (b5) - finish context-closing cleanup
func NewBus(ctx context.Context) Bus {
	b := &bus{
		subs: map[Topic][]Handler{},
	}

	go func(b *bus) {
		<-ctx.Done()
		log.Debugf("close bus")
		b.lk.Lock()
		b.closed = true
		b.lk.Unlock()
	}(b)

	return b
}

// Publish sends an event to the bus
func (b *bus) Publish(ctx context.Context, topic Topic, data interface{}) error {
	b.lk.RLock()
	defer b.lk.RUnlock()
	log.Debugf("Publish: %s", topic)

	if b.closed {
		return ErrBusClosed
	}

	for _, handler := range b.subs[topic] {
		if err := handler(ctx, topic, data); err != nil {
			return err
		}
	}

	return nil
}

// Subscribe requests events from the given topic, returning a channel of those events
func (b *bus) Subscribe(handler Handler, topics ...Topic) {
	b.lk.Lock()
	defer b.lk.Unlock()
	log.Debugf("Subscribe: %v", topics)

	for _, topic := range topics {
		b.subs[topic] = append(b.subs[topic], handler)
	}
}

// // Unsubscribe cleans up a channel that no longer need to receive events
// func (b *bus) Unsubscribe(unsub Handler, rmTopics ...Topic) {
// 	b.lk.Lock()
// 	defer b.lk.Unlock()
// 	for _, rmTopic := range rmTopics {
// 		var replace []Handler
// 		for i, handler := range b.subs[rmTopic] {
// 			if handler == unsub {
// 				replace = append(b.subs[rmTopic][:i], b.subs[rmTopic][i+1:]...)
// 			}
// 		}
// 		if replace != nil {
// 			b.subs[rmTopic] = replace
// 		}
// 	}
// }

// NumSubscribers returns the number of subscribers to the bus's events
func (b *bus) NumSubscribers() int {
	b.lk.Lock()
	defer b.lk.Unlock()
	total := 0
	for _, handlers := range b.subs {
		total += len(handlers)
	}
	return total
}
