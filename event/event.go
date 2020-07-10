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

// Type is the set of all kinds of events emitted by the bus. Use the "Type"
// type to distinguish between different events. Event emitters should
// declare Types as constants and document the expected payload type.
type Type string

// Handler is a function that will be called by the event bus whenever a
// subscribed topic is published. Handler calls are blocking, called in order
// of subscription. Any error returned by a handler is passed back to the
// event publisher.
// The handler context originates from the publisher, and in practice will often
// be scoped to a "request context" like an HTTP request or CLI command
// invocation.
// Generally, even handlers should aim to return quickly, and only delegate to
// goroutines when the publishing event is firing on a long-running process
type Handler func(ctx context.Context, t Type, payload interface{}) error

// Publisher is an interface that can only publish an event
type Publisher interface {
	Publish(ctx context.Context, t Type, payload interface{}) error
}

// Bus is a central coordination point for event publication and subscription
// zero or more subscribers register topics to be notified of, a publisher
// writes a topic event to the bus, which broadcasts to all subscribers of that
// topic
type Bus interface {
	// Publish an event to the bus
	Publish(ctx context.Context, t Type, data interface{}) error
	// Subscribe to one or more topics with a handler function that will be called
	// whenever the event topic is published
	Subscribe(handler Handler, topics ...Type)
	// NumSubscriptions returns the number of subscribers to the bus's events
	NumSubscribers() int
}

// NilBus replaces a nil value. it implements the bus interface, but does
// nothing
var NilBus = nilBus{}

type nilBus struct{}

// assert at compile time that nilBus implements the Bus interface
var _ Bus = (*nilBus)(nil)

// Publish does nothing with the event
func (nilBus) Publish(_ context.Context, _ Type, _ interface{}) error {
	return nil
}

func (nilBus) Subscribe(handler Handler, topics ...Type) {}

func (nilBus) NumSubscribers() int {
	return 0
}

type bus struct {
	lk     sync.RWMutex
	closed bool
	subs   map[Type][]Handler
}

// assert at compile time that bus implements the Bus interface
var _ Bus = (*bus)(nil)

// NewBus creates a new event bus. Event busses should be instantiated as a
// singleton. If the passed in context is cancelled, the bus will stop emitting
// events and close all subscribed channels
//
// TODO (b5) - finish context-closing cleanup
func NewBus(ctx context.Context) Bus {
	b := &bus{
		subs: map[Type][]Handler{},
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
func (b *bus) Publish(ctx context.Context, topic Type, data interface{}) error {
	b.lk.RLock()
	defer b.lk.RUnlock()

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
func (b *bus) Subscribe(handler Handler, topics ...Type) {
	b.lk.Lock()
	defer b.lk.Unlock()
	log.Debugf("Subscribe: %v", topics)

	for _, topic := range topics {
		b.subs[topic] = append(b.subs[topic], handler)
	}
}

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
