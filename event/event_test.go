package event

import (
	"context"
	"fmt"
)

func Example() {
	const (
		ETMainSaidHello   = Topic("main:SaidHello")
		ETMainOpSucceeded = Topic("main:OperationSucceeded")
		ETMainOpFailed    = Topic("main:OperationFailed")
	)

	ctx, done := context.WithCancel(context.Background())
	bus := NewBus(ctx)
	ch1 := make(chan Event)
	ch2 := make(chan Event)
	ch3 := make(chan Event)

	bus.Subscribe(ch1, ETMainSaidHello)
	bus.Subscribe(ch2, ETMainSaidHello)
	bus.Subscribe(ch3, ETMainSaidHello)

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
