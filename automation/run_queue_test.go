package automation

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/event"
)

func TestRunQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rq := NewRunQueue(ctx, event.NilBus, 50*time.Millisecond, 1)
	msgs := []string{}
	expectMsgs := []string{
		"first message",
		"second message",
		"third message",
	}
	ownerID := "owner"
	runID := "run"
	initID := "init"
	mode := "apply"
	f := func(msg string) runQueueFunc {
		return func(ctx context.Context) error {
			msgs = append(msgs, msg)
			return nil
		}
	}

	if err := rq.Push(ctx, ownerID, initID, runID, mode, f(expectMsgs[0])); err != nil {
		t.Fatal(err)
	}
	<-time.After(100 * time.Millisecond)
	if diff := cmp.Diff(expectMsgs[:1], msgs); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
		return
	}

	if err := rq.Push(ctx, ownerID, initID, runID, mode, f(expectMsgs[1])); err != nil {
		t.Fatal(err)
	}
	if err := rq.Push(ctx, ownerID, initID, runID, mode, f(expectMsgs[2])); err != nil {
		t.Fatal(err)
	}
	<-time.After(200 * time.Millisecond)
	cancel()
	if err := rq.Push(ctx, ownerID, initID, runID, mode, f("bad message")); err != nil {
		t.Fatal(err)
	}
	<-time.After(100 * time.Millisecond)
	if diff := cmp.Diff(expectMsgs, msgs); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
		return
	}
}

func TestRunQueueCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rq := NewRunQueue(ctx, event.NilBus, 50*time.Millisecond, 1)
	ownerID := "owner"
	initID := "init"
	runID := "run"
	mode := "apply"
	msg := make(chan string)
	runStarted := make(chan struct{})
	f := func(ctx context.Context) error {
		runStarted <- struct{}{}
		select {
		case <-ctx.Done():
			msg <- "canceled!"
			return nil
		case <-time.After(200 * time.Millisecond):
			msg <- "did not cancel with 100 milliseconds"
			return nil
		}
	}

	if err := rq.Push(ctx, ownerID, initID, runID, mode, f); err != nil {
		t.Fatal(err)
	}
	<-runStarted
	rq.Cancel(runID)
	gotMsg := <-msg
	if gotMsg != "canceled!" {
		t.Errorf(gotMsg)
	}
}
