package cmd

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/qri-io/dag"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
)

func TestDoesCommandExist(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if doesCommandExist("ls") == false {
			t.Error("ls command does not exist!")
		}
		if doesCommandExist("ls111") == true {
			t.Error("ls111 command should not exist!")
		}
	}
}

func TestProgressBars(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := event.NewBus(ctx)

	buf := &bytes.Buffer{}
	PrintProgressBarsOnEvents(buf, bus)

	ref := dsref.MustParse("c/d")

	events := []struct {
		t event.Topic
		p interface{}
	}{
		{event.ETDatasetSaveStarted, event.DsSaveEvent{Username: "a", Name: "b", Completion: 0.1}},
		{event.ETDatasetSaveProgress, event.DsSaveEvent{Username: "a", Name: "b", Completion: 0.2}},
		{event.ETDatasetSaveProgress, event.DsSaveEvent{Username: "a", Name: "b", Completion: 0.3}},
		{event.ETDatasetSaveCompleted, event.DsSaveEvent{Username: "a", Name: "b", Completion: 0.3, Error: fmt.Errorf("oh noes")}},

		{event.ETRemoteClientPullVersionProgress, event.RemoteEvent{Ref: ref, Progress: dag.Completion{0, 1, 1}}},
		{event.ETRemoteClientPushVersionProgress, event.RemoteEvent{Ref: ref, Progress: dag.Completion{0, 1, 1}}},
		{event.ETRemoteClientPullVersionCompleted, event.RemoteEvent{Ref: ref, Progress: dag.Completion{0, 1, 1}, Error: fmt.Errorf("ooooh noes")}},
		{event.ETRemoteClientPushVersionCompleted, event.RemoteEvent{Ref: ref, Progress: dag.Completion{0, 1, 1}, Error: fmt.Errorf("ooooh noes")}},
	}

	for _, e := range events {
		bus.Publish(ctx, e.t, e.p)
	}
}
