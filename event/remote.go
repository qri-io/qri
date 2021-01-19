package event

import (
	"github.com/qri-io/dag"
	"github.com/qri-io/qri/dsref"
)

const (
	// ETRemoteClientPushVersionProgress indicates a change in progress of a
	// dataset version push. Progress can fire as much as once-per-block.
	// subscriptions do not block the publisher
	// payload will be a RemoteEvent
	ETRemoteClientPushVersionProgress = Topic("remoteClient:PushVersionProgress")
	// ETRemoteClientPushVersionCompleted indicates a version successfully pushed
	// to a remote.
	// payload will be a RemoteEvent
	ETRemoteClientPushVersionCompleted = Topic("remoteClient:PushVersionCompleted")
	// ETRemoteClientPushDatasetCompleted indicates pushing a dataset
	// (logbook + versions) completed
	// payload will be a RemoteEvent
	ETRemoteClientPushDatasetCompleted = Topic("remoteClient:PushDatasetCompleted")
	// ETRemoteClientPullVersionProgress indicates a change in progress of a
	// dataset version pull. Progress can fire as much as once-per-block.
	// subscriptions do not block the publisher
	// payload will be a RemoteEvent
	ETRemoteClientPullVersionProgress = Topic("remoteClient:PullVersionProgress")
	// ETRemoteClientPullVersionCompleted indicates a version successfully pulled
	// from a remote.
	// payload will be a RemoteEvent
	ETRemoteClientPullVersionCompleted = Topic("remoteClient:PullVersionCompleted")
	// ETRemoteClientPullDatasetCompleted indicates pulling a dataset
	// (logbook + versions) completed
	// payload will be a RemoteEvent
	ETRemoteClientPullDatasetCompleted = Topic("remoteClient:PullDatasetCompleted")
	// ETRemoteClientRemoveDatasetCompleted indicates removing a dataset
	// (logbook + versions) remove completed
	// payload will be a RemoteEvent
	ETRemoteClientRemoveDatasetCompleted = Topic("remoteClient:RemoveDatasetCompleted")
)

// RemoteEvent encapsulates the push / pull progress of a dataset version
type RemoteEvent struct {
	Ref        dsref.Ref      `json:"ref"`
	RemoteAddr string         `json:"remoteAddr"`
	Progress   dag.Completion `json:"progress"`
	Error      error          `json:"error,omitempty"`
}
