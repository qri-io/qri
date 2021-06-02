package lib

import (
	"fmt"
	"strings"

	"github.com/qri-io/qri/dsref"
)

// APIEndpoint is a simple alias to have a consistent definition
// of our API endpoints
type APIEndpoint string

// String allows for less casting in general code
func (ae APIEndpoint) String() string {
	return string(ae)
}

// NoTrailingSlash returns the path without a traling slash
func (ae APIEndpoint) NoTrailingSlash() string {
	s := string(ae)
	s = strings.TrimSuffix(s, "/")
	return s
}

// WithSuffix returns a new endpoint with added path suffix
func (ae APIEndpoint) WithSuffix(suffix string) APIEndpoint {
	return APIEndpoint(fmt.Sprintf("%s/%s", ae, suffix))
}

const (
	// aggregate endpoints

	// AEList lists all datasets in your collection
	AEList = APIEndpoint("/list")
	// AEDiff is an endpoint for generating dataset diffs
	AEDiff = APIEndpoint("/diff")
	// AEChanges is an endpoint for generating dataset change reports
	AEChanges = APIEndpoint("/changes")
	// AESQL executes SQL commands
	AESQL = APIEndpoint("/sql")

	// auth endpoints

	// AECreateAuthToken creates an auth token for a user
	AECreateAuthToken = APIEndpoint("/access/token")

	// automation endpoints

	// AEApply invokes a transform apply
	AEApply = APIEndpoint("/auto/apply")

	AEDeploy = APIEndpoint("/auto/deploy")

	// dataset endpoints

	// AEComponentStatus returns what changed for a specific commit
	AEComponentStatus = APIEndpoint("/ds/componentstatus")
	// AEGet is an endpoint for fetch individual dataset components
	AEGet = APIEndpoint("/ds/get")
	// AEActivity is an endpoint that returns a dataset activity list
	AEActivity = APIEndpoint("/ds/activity")
	// AERename is an endpoint for renaming datasets
	AERename = APIEndpoint("/ds/rename")
	// AESave is an endpoint for saving a dataset
	AESave = APIEndpoint("/ds/save")
	// AEPull facilittates dataset pull requests from a remote
	AEPull = APIEndpoint("/ds/pull")
	// AEPush facilitates dataset push requests to a remote
	AEPush = APIEndpoint("/ds/push")
	// AERender renders the current dataset ref
	AERender = APIEndpoint("/ds/render")
	// AERemove exposes the dataset remove mechanics
	AERemove = APIEndpoint("/ds/remove")
	// AEValidate is an endpoint for validating datasets
	AEValidate = APIEndpoint("/ds/validate")
	// AEManifest generates a manifest for a dataset path
	AEManifest = APIEndpoint("/ds/manifest")
	// AEManifestMissing generates a manifest of blocks that are not present on this repo for a given manifest
	AEManifestMissing = APIEndpoint("/ds/manifest/missing")
	// AEDAGInfo generates a dag.Info for a dataset path
	AEDAGInfo = APIEndpoint("/ds/daginfo")

	// peer endpoints

	// AEPeer fetches a specific peer
	AEPeer = APIEndpoint("/peer")
	// AEConnect initiates an explicit connection to a peer
	AEConnect = APIEndpoint("/peer/connect")
	// AEDisconnect closes an explicit connection to a peer
	AEDisconnect = APIEndpoint("/peer/disconnect")
	// AEPeers fetches all the peers
	AEPeers = APIEndpoint("/peer/list")

	// profile endpoints

	// AEGetProfile is an alias for the me endpoint
	AEGetProfile = APIEndpoint("/profile")
	// AESetProfile is an endpoint to set the profile
	AESetProfile = APIEndpoint("/profile/set")
	// AESetProfilePhoto is an endpoint to set the profile photo
	AESetProfilePhoto = APIEndpoint("/profile/photo")
	// AESetPosterPhoto is an endpoint to set the profile poster
	AESetPosterPhoto = APIEndpoint("/profile/poster")

	// remote client endpoints

	// AEFeeds fetches and index of named feeds
	AEFeeds = APIEndpoint("/remote/feeds")
	// AEPreview fetches a dataset preview from the registry
	AEPreview = APIEndpoint("/remote/preview")
	// AERemoteRemove removes a dataset from a given remote
	AERemoteRemove = APIEndpoint("/remote/remove")
	// AERegistryNew creates a new user on the registry
	AERegistryNew = APIEndpoint("/remote/registry/profile/new")
	// AERegistryProve links an the current peer with an existing
	// user on the registry
	AERegistryProve = APIEndpoint("/remote/registry/profile/prove")
	// AESearch returns a list of dataset search results
	AESearch = APIEndpoint("/remote/search")

	// fsi endpoints

	// AEStatus returns the filesystem dataset status
	AEStatus = APIEndpoint("/wd/status")
	// AEInit invokes a dataset initialization on the filesystem
	AEInit = APIEndpoint("/wd/init")
	// AECanInitDatasetWorkDir returns whether a dataset can be initialized
	AECanInitDatasetWorkDir = APIEndpoint("/wd/caninitworkdir")
	// AECheckout invokes a dataset checkout to the filesystem
	AECheckout = APIEndpoint("/wd/checkout")
	// AERestore invokes a restore
	AERestore = APIEndpoint("/wd/restore")
	// AEFSIWrite writes input data to the filesystem
	AEFSIWrite = APIEndpoint("/wd/write")
	// AEFSICreateLink creates an fsi link
	AEFSICreateLink = APIEndpoint("/wd/createlink")
	// AEFSIUnlink removes the fsi link
	AEFSIUnlink = APIEndpoint("/wd/unlink")
	// AEEnsureRef ensures that the ref is fsi linked
	AEEnsureRef = APIEndpoint("/wd/ensureref")

	// sync endpoints

	// AERemoteDSync exposes the dsync mechanics
	AERemoteDSync = APIEndpoint("/remote/dsync")
	// AERemoteLogSync exposes the logsync mechanics
	AERemoteLogSync = APIEndpoint("/remote/logsync")
	// AERemoteRefs exposes the remote ref resolution mechanics
	AERemoteRefs = APIEndpoint("/remote/refs")

	// other endpoints

	// AEConnections lists qri & IPFS connections
	AEConnections = APIEndpoint("/connections")
	// AEConnectedQriProfiles lists qri profile connections
	AEConnectedQriProfiles = APIEndpoint("/connections/qri")

	// DenyHTTP will disable HTTP access to a method
	DenyHTTP = APIEndpoint("")
)

// DsRefFromPath parses a path and returns a dsref.Ref
func DsRefFromPath(path string) (dsref.Ref, error) {
	refstr := HTTPPathToQriPath(path)
	return dsref.ParsePeerRef(refstr)
}

// HTTPPathToQriPath converts a http path to a
// qri path
func HTTPPathToQriPath(path string) string {
	paramIndex := strings.Index(path, "?")
	if paramIndex != -1 {
		path = path[:paramIndex]
	}
	// TODO(dustmop): If a user has a dataset named "at", this breaks
	path = strings.Replace(path, "/at/", "@/", 1)
	path = strings.TrimPrefix(path, "/")
	return path
}
