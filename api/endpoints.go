package api

import (
	"github.com/qri-io/qri/lib"
)

const (
	// base endpoints

	// AEHome is the / endpoint
	AEHome = lib.APIEndpoint("/")
	// AEHealth is the service health check endpoint
	AEHealth = lib.APIEndpoint("/health")
	// AEIPFS is the IPFS endpoint
	AEIPFS = lib.APIEndpoint("/qfs/ipfs/{path:.*}")
	// AEWebUI serves the remote WebUI
	AEWebUI = lib.APIEndpoint("/webui")

	// dataset endpoints

	// AEGetCSVFullRef is the route used to get a body as a csv, that can also handle a specific hash
	AEGetCSVFullRef = lib.APIEndpoint("/ds/get/{username}/{name}/at/{fs}/{hash}/body.csv")
	// AEGetCSVShortRef is the route used to get a body as a csv
	AEGetCSVShortRef = lib.APIEndpoint("/ds/get/{username}/{name}/body.csv")
	// AEUnpack unpacks a zip file and sends it back
	AEUnpack = lib.APIEndpoint("/ds/unpack")
)
