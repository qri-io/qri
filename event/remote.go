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
	ETRemoteClientPushVersionProgress = Type("remoteClient:PushVersionProgress")
	// ETRemoteClientPushVersionCompleted indicates a version successfully pushed
	// to a remote.
	// payload will be a RemoteEvent
	ETRemoteClientPushVersionCompleted = Type("remoteClient:PushVersionCompleted")
	// ETRemoteClientPushDatasetCompleted indicates pushing a dataset
	// (logbook + versions) completed
	// payload will be a RemoteEvent
	ETRemoteClientPushDatasetCompleted = Type("remoteClient:PushDatasetCompleted")
	// ETDatasetPushed fires at the same logical time as
	// ETRemoteClientPushDatasetCompleted, but contains a versionInfo payload
	// for subscribers that need additional fields from the pushed dataset
	// payload will be a dsref.VersionInfo
	ETDatasetPushed = Type("remoteClient:DatasetPushed")
	// ETRemoteClientPullVersionProgress indicates a change in progress of a
	// dataset version pull. Progress can fire as much as once-per-block.
	// subscriptions do not block the publisher
	// payload will be a RemoteEvent
	ETRemoteClientPullVersionProgress = Type("remoteClient:PullVersionProgress")
	// ETRemoteClientPullVersionCompleted indicates a version successfully pulled
	// from a remote.
	// payload will be a RemoteEvent
	ETRemoteClientPullVersionCompleted = Type("remoteClient:PullVersionCompleted")
	// ETRemoteClientPullDatasetCompleted indicates pulling a dataset
	// (logbook + versions) completed
	// payload will be a RemoteEvent
	ETRemoteClientPullDatasetCompleted = Type("remoteClient:PullDatasetCompleted")
	// ETDatasetPulled fires at the same logical time as
	// ETRemoteClientPullDatasetCompleted, but contains a versionInfo payload
	// for subscribers that need additional fields from the pulled dataset
	// payload will be a dsref.VersionInfo
	ETDatasetPulled = Type("remoteClient:DatasetPulled")
	// ETRemoteClientRemoveDatasetCompleted indicates removing a dataset
	// (logbook + versions) remove completed
	// payload will be a RemoteEvent
	ETRemoteClientRemoveDatasetCompleted = Type("remoteClient:RemoveDatasetCompleted")
)

// RemoteEvent encapsulates the push / pull progress of a dataset version
type RemoteEvent struct {
	Ref        dsref.Ref      `json:"ref"`
	RemoteAddr string         `json:"remoteAddr"`
	Progress   dag.Completion `json:"progress"`
	Error      error          `json:"error,omitempty"`
}

const (
	// ETRegistryProfileCreated indicates a successful profile creation on the
	// configured registry
	// payload will be a RegistryProfileCreated
	ETRegistryProfileCreated = Type("registry:ProfileCreated")
)

// RegistryProfileCreated encapsulates fields in a profile creation response
// from the configured registry
type RegistryProfileCreated struct {
	RegistryLocation string `json:"registryLocation"`
	ProfileID        string `json:"profileID"`
	Username         string `json:"username"`
}
