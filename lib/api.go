package lib

import (
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

const (
	// base endpoints

	// AEHome is the / endpoint
	AEHome = APIEndpoint("/")
	// AEHealth is the service health check endpoint
	AEHealth = APIEndpoint("/health")
	// AEIPFS is the IPFS endpoint
	AEIPFS = APIEndpoint("/ipfs/{path:.*}")

	// profile enpoints

	// AEMe is the "own" profile endpoint
	AEMe = APIEndpoint("/me")
	// AEProfile is an alias for the me endpoint
	AEProfile = APIEndpoint("/profile")
	// AEProfilePhoto is an endpoint to serve the profile photo
	AEProfilePhoto = APIEndpoint("/profile/photo")
	// AEProfilePoster is an endpoint to serve the profile poster
	AEProfilePoster = APIEndpoint("/profile/poster")

	// peer endpoints

	// AEPeers fetches all the peers
	AEPeers = APIEndpoint("/peers")
	// AEPeer fetches a specific peer
	AEPeer = APIEndpoint("/peers/{path:.*}")
	// AEConnect initiates an explicit connection to a peer
	AEConnect = APIEndpoint("/connect")
	// AEConnections lists qri & IPFS connections
	AEConnections = APIEndpoint("/connections")

	// remote endpoints

	// AERemoteDSync exposes the dsync mechanics
	AERemoteDSync = APIEndpoint("/remote/dsync")
	// AERemoteLogSync exposes the logsync mechanics
	AERemoteLogSync = APIEndpoint("/remote/logsync")
	// AERemoteRefs exposes the remote ref resolution mechanics
	AERemoteRefs = APIEndpoint("/remote/refs")

	// dataset endpoints

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
	// AEDiff is an endpoint for generating dataset diffs
	AEDiff = APIEndpoint("/diff")
	// AEChanges is an endpoint for generating dataset change reports
	AEChanges = APIEndpoint("/changes")
	// AEUnpack unpacks a zip file and sends it back
	AEUnpack = APIEndpoint("/unpack/{path:.*}")

	// remote client endpoints

	// AEPush facilitates dataset push requests to a remote
	AEPush = APIEndpoint("/push/{path:.*}")
	// AEPull facilittates dataset pull requests from a remote
	AEPull = APIEndpoint("/pull/{path:.*}")
	// AEFeeds fetches and index of named feeds
	AEFeeds = APIEndpoint("/feeds")
	// AEPreview fetches a dataset preview from the registry
	AEPreview = APIEndpoint("/preview/{path:.*}")

	// fsi endpoints

	// AEStatus returns the filesystem dataset status
	AEStatus = APIEndpoint("/status/{path:.*}")
	// AEWhatChanged returns what changed for a specific commit
	AEWhatChanged = APIEndpoint("/whatchanged/{path:.*}")
	// AEInit invokes a dataset initialization on the filesystem
	AEInit = APIEndpoint("/init/{path:.*}")
	// AECheckout invokes a dataset checkout to the filesystem
	AECheckout = APIEndpoint("/checkout/{path:.*}")
	// AERestore invokes a restore
	AERestore = APIEndpoint("/restore/{path:.*}")
	// AEFSIWrite writes input data to the filesystem
	AEFSIWrite = APIEndpoint("/fsi/write/{path:.*}")

	// other endpoints

	// AEHistory returns dataset logs
	AEHistory = APIEndpoint("/history/{path:.*}")
	// AERender renders the current dataset ref
	AERender = APIEndpoint("/render")
	// AERenderAlt renders a given dataset ref
	AERenderAlt = APIEndpoint("/render/{path:.*}")
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
