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
	defer done()

	bus := NewBus(ctx)

	makeDoneHandler := func(label string) Handler {
		return func(ctx context.Context, topic Topic, payload interface{}) error {
			fmt.Printf("%s handler called\n", label)
			return nil
		}
	}

	bus.Subscribe(makeDoneHandler("first"), ETMainSaidHello, ETMainOpSucceeded)
	bus.Subscribe(makeDoneHandler("second"), ETMainSaidHello)
	bus.Subscribe(makeDoneHandler("third"), ETMainSaidHello)

	bus.Publish(ctx, ETMainSaidHello, "hello")
	bus.Publish(ctx, ETMainOpSucceeded, "operation worked!")

	// opCh := bus.SubscribeOnce(ETMainOpSucceeded, ETMainOpFailed)

	// go bus.Publish(ETMainOpFailed, fmt.Errorf("it didn't work?"))

	// event := <-opCh
	// fmt.Println(event.Payload)
	// done()

	// Output: first handler called
	// second handler called
	// third handler called
	// first handler called
}
