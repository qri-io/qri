// Package event implements an event bus.
// for a great introduction to the event bus pattern in go, see:
// https://levelup.gitconnected.com/lets-write-a-simple-event-bus-in-go-79b9480d8997
package event

import (
	"context"
	"fmt"
	"sync"
	"time"

	golog "github.com/ipfs/go-log"
)

var (
	log = golog.Logger("event")

	// ErrBusClosed indicates the event bus is no longer coordinating events
	// because it's parent context has closed
	ErrBusClosed = fmt.Errorf("event bus is closed")
	// NowFunc is the function that generates timestamps (tests may override)
	NowFunc = time.Now
)

// Type is the set of all kinds of events emitted by the bus. Use "Type" to
// distinguish between different events. Event emitters should declare Types
// as constants and document the expected payload type. This term, although
// similar to the keyword in go, is used to match what react/redux use in
// their event system.
type Type string

// Event represents an event that subscribers will receive from the bus
type Event struct {
	Type      Type
	Timestamp int64
	SessionID string
	Payload   interface{}
}

// Handler is a function that will be called by the event bus whenever a
// matching event is published. Handler calls are blocking, called in order
// of subscription. Any error returned by a handler is passed back to the
// event publisher.
// The handler context originates from the publisher, and in practice will often
// be scoped to a "request context" like an HTTP request or CLI command
// invocation.
// Generally, even handlers should aim to return quickly, and only delegate to
// goroutines when the publishing event is firing on a long-running process
type Handler func(ctx context.Context, e Event) error

// Publisher is an interface that can only publish an event
type Publisher interface {
	Publish(ctx context.Context, typ Type, payload interface{}) error
	PublishID(ctx context.Context, typ Type, sessionID string, payload interface{}) error
}

// Bus is a central coordination point for event publication and subscription.
// Zero or more subscribers register to be notified of events, optionally by type
// or id, then a publisher writes an event to the bus, which broadcasts to all
// matching subscribers
type Bus interface {
	// Publish an event to the bus
	Publish(ctx context.Context, typ Type, data interface{}) error
	// PublishID publishes an event with an arbitrary session id
	PublishID(ctx context.Context, typ Type, sessionID string, data interface{}) error
	// Subscribe to one or more eventTypes with a handler function that will be called
	// whenever the event type is published
	SubscribeTypes(handler Handler, eventTypes ...Type)
	// SubscribeID subscribes to only events that have a matching session id
	SubscribeID(handler Handler, sessionID string)
	// SubscribeAll subscribes to all events
	SubscribeAll(handler Handler)
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

// PublishID does nothing with the event
func (nilBus) PublishID(_ context.Context, _ Type, _ string, _ interface{}) error {
	return nil
}

// SubscribeTypes does nothing
func (nilBus) SubscribeTypes(handler Handler, eventTypes ...Type) {}

func (nilBus) SubscribeID(handler Handler, id string) {}

func (nilBus) SubscribeAll(handler Handler) {}

func (nilBus) NumSubscribers() int {
	return 0
}

type bus struct {
	lk      sync.RWMutex
	closed  bool
	subs    map[Type][]Handler
	allSubs []Handler
	idSubs  map[string][]Handler
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
		subs:    map[Type][]Handler{},
		idSubs:  map[string][]Handler{},
		allSubs: []Handler{},
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
func (b *bus) Publish(ctx context.Context, typ Type, payload interface{}) error {
	return b.publish(ctx, typ, "", payload)
}

// PublishID sends an event with a given sessionID to the bus
func (b *bus) PublishID(ctx context.Context, typ Type, sessionID string, payload interface{}) error {
	return b.publish(ctx, typ, sessionID, payload)
}

func (b *bus) publish(ctx context.Context, typ Type, sessionID string, payload interface{}) error {
	b.lk.RLock()
	defer b.lk.RUnlock()
	log.Debugw("publish", "type", typ, "payload", payload)

	if b.closed {
		return ErrBusClosed
	}

	e := Event{
		Type:      typ,
		Timestamp: NowFunc().UnixNano(),
		SessionID: sessionID,
		Payload:   payload,
	}

	// TODO(dustmop): Add instrumentation, perhaps to ctx, to make logging / tracing
	// a single event easier to do.

	for _, handler := range b.subs[typ] {
		if err := handler(ctx, e); err != nil {
			return err
		}
	}

	if sessionID != "" {
		for _, handler := range b.idSubs[sessionID] {
			if err := handler(ctx, e); err != nil {
				return err
			}
		}
	}

	for _, handler := range b.allSubs {
		if err := handler(ctx, e); err != nil {
			return err
		}
	}

	return nil
}

// Subscribe requests events from the given type, returning a channel of those events
func (b *bus) SubscribeTypes(handler Handler, eventTypes ...Type) {
	b.lk.Lock()
	defer b.lk.Unlock()
	log.Debugf("Subscribe to types: %v", eventTypes)

	for _, typ := range eventTypes {
		b.subs[typ] = append(b.subs[typ], handler)
	}
}

// SubscribeID requests events that match the given sessionID
func (b *bus) SubscribeID(handler Handler, sessionID string) {
	b.lk.Lock()
	defer b.lk.Unlock()
	log.Debugf("Subscribe to ID: %v", sessionID)
	b.idSubs[sessionID] = append(b.idSubs[sessionID], handler)
}

// SubscribeAll requests all events from the bus
func (b *bus) SubscribeAll(handler Handler) {
	b.lk.Lock()
	defer b.lk.Unlock()
	log.Debugf("Subscribe All")
	b.allSubs = append(b.allSubs, handler)
}

// NumSubscribers returns the number of subscribers to the bus's events
func (b *bus) NumSubscribers() int {
	b.lk.RLock()
	defer b.lk.RUnlock()
	total := 0
	for _, handlers := range b.subs {
		total += len(handlers)
	}
	for _, handlers := range b.idSubs {
		total += len(handlers)
	}
	total += len(b.allSubs)
	return total
}
