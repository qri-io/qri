package event

import (
	"context"
	"fmt"
	"testing"
)

func Example() {
	const (
		ETMainSaidHello   = Topic("main:SaidHello")
		ETMainOpSucceeded = Topic("main:OperationSucceeded")
		ETMainOpFailed    = Topic("main:OperationFailed")
	)

	ctx, done := context.WithCancel(context.Background())
	bus := NewBus(ctx)
	ch1 := bus.Subscribe(ETMainSaidHello)
	ch2 := bus.Subscribe(ETMainSaidHello)
	ch3 := bus.Subscribe(ETMainSaidHello)

	go bus.Publish(ETMainSaidHello, "hello")

	tasks := 3

	for {
		select {
		case d := <-ch1:
			fmt.Println(d.Payload)
		case d := <-ch2:
			fmt.Println(d.Payload)
		case d := <-ch3:
			fmt.Println(d.Payload)
		}

		tasks--
		if tasks == 0 {
			break
		}
	}

	opCh := bus.SubscribeOnce(ETMainOpSucceeded, ETMainOpFailed)

	go bus.Publish(ETMainOpFailed, fmt.Errorf("it didn't work?"))

	event := <-opCh
	fmt.Println(event.Payload)
	done()

	// Output: hello
	// hello
	// hello
	// it didn't work?
}

func TestSubscribeUnsubscribe(t *testing.T) {
	ctx := context.Background()
	const testTopic = Topic("test_event")

	b := NewBus(ctx)
	ch1 := b.Subscribe(testTopic)
	ch2 := b.Subscribe(testTopic)

	if b.NumSubscribers() != 2 {
		t.Errorf("expected 2 subscribers, got %d", b.NumSubscribers())
	}

	b.Unsubscribe(ch1)

	if b.NumSubscribers() != 1 {
		t.Errorf("expected 1 subscribers, got %d", b.NumSubscribers())
	}

	b.Unsubscribe(ch2)

	if b.NumSubscribers() != 0 {
		t.Errorf("expected 1 subscribers, got %d", b.NumSubscribers())
	}
}
