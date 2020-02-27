// Package event implements an event bus.
// for a great introduction to the event bus pattern in go, see:
// https://levelup.gitconnected.com/lets-write-a-simple-event-bus-in-go-79b9480d8997
package event

import (
	"context"
	"sync"

	golog "github.com/ipfs/go-log"
)

var log = golog.Logger("event")

// Topic is the set of all topics emitted by the bus. Use the topic type to
// distinguish event names. Event emitters should declare Topics as constants
// and document the expected data payload type
type Topic string

// Event is a topic & data payload
type Event struct {
	Topic
	Payload interface{}
}

// Publisher is an interface that can only publish an event
type Publisher interface {
	Publish(t Topic, data interface{})
}

// NilPublisher replaces a nil value, does nothing
type NilPublisher struct {
}

// Publish does nothing with the event
func (n *NilPublisher) Publish(t Topic, data interface{}) {
}

// Bus is a central coordination point for event publication and subscription
// zero or more subscribers register topics to be notified of, a publisher
// writes a topic event to the bus, which broadcasts to all subscribers of that
// topic
type Bus interface {
	// Publish an event to the bus
	Publish(t Topic, data interface{})
	// Subscribe to one or more topics
	Subscribe(topics ...Topic) <-chan Event
	// Unsubscribe cleans up a channel that no longer need to receive events
	Unsubscribe(<-chan Event)
	// SubscribeOnce to one or more topics. the returned channel will only fire
	// once, when the first event that matches any of the given topics
	// the common use case for multiple subscriptions is subscribing to both
	// success and error events
	SubscribeOnce(types ...Topic) <-chan Event
	// NumSubscriptions returns the number of subscribers to the bus's events
	NumSubscribers() int
}

type dataChannels []chan Event

type bus struct {
	ctx context.Context

	lk   sync.RWMutex
	subs map[Topic]dataChannels

	onceLk sync.RWMutex
	onces  []onceSub
}

type onceSub struct {
	ch     chan Event
	topics map[Topic]bool
}

// NewBus creates a new event bus. Event busses should be instantiated as a
// singleton. If the passed in context is cancelled, the bus will stop emitting
// events and close all subscribed channels
//
// TODO (b5) - finish context-closing cleanup
func NewBus(ctx context.Context) Bus {
	b := &bus{
		ctx:  ctx,
		subs: map[Topic]dataChannels{},
	}

	go func(b *bus) {
		<-b.ctx.Done()
		log.Debugf("close bus")
		// TODO (b5) - cleanup bus resources, potentially closing channels
		// properly closing channels will require we keep a set of channels in
		// addition to the map to avoid double-closing
	}(b)

	return b
}

// Publish sends an event to the bus
func (b *bus) Publish(topic Topic, data interface{}) {
	b.lk.RLock()
	defer b.lk.RUnlock()
	log.Debugf("Publish: %s", topic)

	event := Event{Payload: data, Topic: topic}

	if chans, ok := b.subs[topic]; ok {
		// slices in this map refer to same array even though they are passed by value
		// creating a new slice preserves locking correctly
		channels := append(dataChannels{}, chans...)
		go func(e Event, dataChannelSlices dataChannels) {
			for _, ch := range dataChannelSlices {
				ch <- e
			}
		}(event, channels)
	}

	go func(e Event) {
		b.onceLk.Lock()
		defer b.onceLk.Unlock()

		for i, sub := range b.onces {
			if sub.topics[topic] {
				sub.ch <- event
				close(sub.ch)
				log.Debug("closing once ch with topic: %s", topic)
				// Remove the subscription
				b.onces[i] = b.onces[len(b.onces)-1] // copy last element to index i
				b.onces[len(b.onces)-1] = onceSub{}  // erase last element (write zero value)
				b.onces = b.onces[:len(b.onces)-1]   // truncate slice
			}
		}
	}(event)
}

// Subscribe requests events from the given topic, returning a channel of those events
func (b *bus) Subscribe(topics ...Topic) <-chan Event {
	b.lk.Lock()
	defer b.lk.Unlock()
	log.Debugf("Subscribe: %v", topics)

	ch := make(chan Event)

	for _, topic := range topics {
		if prev, ok := b.subs[topic]; ok {
			b.subs[topic] = append(prev, ch)
		} else {
			b.subs[topic] = dataChannels{ch}
		}
	}

	return ch
}

// Unsubscribe cleans up a channel that no longer need to receive events
func (b *bus) Unsubscribe(unsub <-chan Event) {
	for topic, channels := range b.subs {
		var replace dataChannels
		for i, ch := range channels {
			if ch == unsub {
				replace = append(channels[:i], channels[i+1:]...)
			}
		}
		if replace != nil {
			b.subs[topic] = replace
		}
	}
}

// NumSubscribers returns the number of subscribers to the bus's events
func (b *bus) NumSubscribers() int {
	total := 0
	for _, channels := range b.subs {
		total += len(channels)
	}
	return total
}

// SubscribeOnce will only get one event of the topic, then close itself
func (b *bus) SubscribeOnce(topics ...Topic) <-chan Event {
	b.onceLk.Lock()
	defer b.onceLk.Unlock()
	log.Debugf("SubscribeOnce: %v", topics)

	topicMap := map[Topic]bool{}
	for _, t := range topics {
		topicMap[t] = true
	}

	ch := make(chan Event)
	b.onces = append(b.onces, onceSub{
		ch:     ch,
		topics: topicMap,
	})

	return ch
}
