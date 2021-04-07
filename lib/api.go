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
	// base endpoints

	// AEHome is the / endpoint
	AEHome = APIEndpoint("/")
	// AEHealth is the service health check endpoint
	AEHealth = APIEndpoint("/health")
	// AEIPFS is the IPFS endpoint
	AEIPFS = APIEndpoint("/ipfs/{path:.*}")

	// profile endpoints

	// AEGetProfile is an alias for the me endpoint
	AEGetProfile = APIEndpoint("/profile")
	// AESetProfile is an endpoint to set the profile
	AESetProfile = APIEndpoint("/profile/set")
	// AESetProfilePhoto is an endpoint to set the profile photo
	AESetProfilePhoto = APIEndpoint("/profile/photo")
	// AESetPosterPhoto is an endpoint to set the profile poster
	AESetPosterPhoto = APIEndpoint("/profile/poster")

	// peer endpoints

	// AEPeers fetches all the peers
	AEPeers = APIEndpoint("/peers")
	// AEPeer fetches a specific peer
	AEPeer = APIEndpoint("/peer")
	// AEConnect initiates an explicit connection to a peer
	AEConnect = APIEndpoint("/connect")
	// AEDisconnect closes an explicit connection to a peer
	AEDisconnect = APIEndpoint("/disconnect")
	// AEConnections lists qri & IPFS connections
	AEConnections = APIEndpoint("/connections")
	// AEConnectedQriProfiles lists qri profile connections
	AEConnectedQriProfiles = APIEndpoint("/connections/qri")

	// remote endpoints

	// AERemoteDSync exposes the dsync mechanics
	AERemoteDSync = APIEndpoint("/remote/dsync")
	// AERemoteLogSync exposes the logsync mechanics
	AERemoteLogSync = APIEndpoint("/remote/logsync")
	// AERemoteRefs exposes the remote ref resolution mechanics
	AERemoteRefs = APIEndpoint("/remote/refs")

	// dataset endpoints

	// AEListRaw lists all datasets in your collection as a well formatted string
	AEListRaw = APIEndpoint("/listraw")
	// AEList lists all datasets in your collection
	AEList = APIEndpoint("/list")
	// AEPeerList lists all datasets in your
	// collection for a particular peer
	AEPeerList = APIEndpoint("/list/{peer}")
	// AESave is an endpoint for saving a dataset
	AESave = APIEndpoint("/save")
	// AERemove exposes the dataset remove mechanics
	AERemove = APIEndpoint("/remove")
	// AEGet is an endpoint for fetch individual dataset components
	AEGet = APIEndpoint("/get")
	// AERename is an endpoint for renaming datasets
	AERename = APIEndpoint("/rename")
	// AEValidate is an endpoint for validating datasets
	AEValidate = APIEndpoint("/validate")
	// AEDiff is an endpoint for generating dataset diffs
	AEDiff = APIEndpoint("/diff")
	// AEChanges is an endpoint for generating dataset change reports
	AEChanges = APIEndpoint("/changes")
	// AEUnpack unpacks a zip file and sends it back
	AEUnpack = APIEndpoint("/unpack/{path:.*}")
	// AEManifest generates a manifest for a dataset path
	AEManifest = APIEndpoint("/manifest")
	// AEManifestMissing generates a manifest of blocks that are not present on this repo for a given manifest
	AEManifestMissing = APIEndpoint("/manifest/missing")
	// AEDAGInfo generates a dag.Info for a dataset path
	AEDAGInfo = APIEndpoint("/dag/info")

	// remote client endpoints

	// AEPush facilitates dataset push requests to a remote
	AEPush = APIEndpoint("/push")
	// AEPull facilitates dataset pull requests from a remote
	AEPull = APIEndpoint("/pull")
	// AEFeeds fetches and index of named feeds
	AEFeeds = APIEndpoint("/feeds")
	// AEPreview fetches a dataset preview from the registry
	AEPreview = APIEndpoint("/preview")
	// AERemoteRemove removes a dataset from a given remote
	AERemoteRemove = APIEndpoint("/remote/remove")

	// fsi endpoints

	// AEStatus returns the filesystem dataset status
	AEStatus = APIEndpoint("/status")
	// AEWhatChanged returns what changed for a specific commit
	AEWhatChanged = APIEndpoint("/whatchanged")
	// AEInit invokes a dataset initialization on the filesystem
	AEInit = APIEndpoint("/init")
	// AECanInitDatasetWorkDir returns whether a dataset can be initialized
	AECanInitDatasetWorkDir = APIEndpoint("/caninitdatasetworkdir")
	// AECheckout invokes a dataset checkout to the filesystem
	AECheckout = APIEndpoint("/checkout")
	// AERestore invokes a restore
	AERestore = APIEndpoint("/restore")
	// AEFSIWrite writes input data to the filesystem
	AEFSIWrite = APIEndpoint("/fsi/write")
	// AEFSICreateLink creates an fsi link
	AEFSICreateLink = APIEndpoint("/fsi/createlink")
	// AEFSIUnlink removes the fsi link
	AEFSIUnlink = APIEndpoint("/fsi/unlink")
	// AEEnsureRef ensures that the ref is fsi linked
	AEEnsureRef = APIEndpoint("/fsi/ensureref")

	// auth endpoints

	// AECreateAuthToken creates an auth token for a user
	AECreateAuthToken = APIEndpoint("/auth/createauthtoken")

	// other endpoints

	// AEHistory returns dataset logs
	AEHistory = APIEndpoint("/history")
	// AEEntries lists log entries for actions taken on a given dataset
	AEEntries = APIEndpoint("/log")
	// AERawLogbook returns the full logbook encoded as human-oriented json
	AERawLogbook = APIEndpoint("/logbook")
	// AELogbookSummary returns a string overview of the logbook
	AELogbookSummary = APIEndpoint("/logbook/summary")
	// AERender renders the current dataset ref
	AERender = APIEndpoint("/render")
	// AERegistryNew creates a new user on the registry
	AERegistryNew = APIEndpoint("/registry/profile/new")
	// AERegistryProve links an the current peer with an existing
	// user on the registry
	AERegistryProve = APIEndpoint("/registry/profile/prove")
	// AESearch returns a list of dataset search results
	AESearch = APIEndpoint("/search")
	// AESQL executes SQL commands
	AESQL = APIEndpoint("/sql")
	// AEApply invokes a transform apply
	AEApply = APIEndpoint("/apply")
	// AEWebUI serves the remote WebUI
	AEWebUI = APIEndpoint("/webui")

	// denyRPC if used will disable RPC calls for a method
	denyRPC = APIEndpoint("")
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
