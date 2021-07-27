package api

import (
	qhttp "github.com/qri-io/qri/lib/http"
)

const (
	// base endpoints

	// AEHome is the / endpoint
	AEHome qhttp.APIEndpoint = "/"
	// AEHealth is the service health check endpoint
	AEHealth qhttp.APIEndpoint = "/health"
	// AEIPFS is the IPFS endpoint
	AEIPFS qhttp.APIEndpoint = "/qfs/ipfs/{path:.*}"
	// AEWebUI serves the remote WebUI
	AEWebUI qhttp.APIEndpoint = "/webui"

	// dataset endpoints

	// AEGetCSVFullRef is the route used to get a body as a csv, that can also handle a specific hash
	AEGetCSVFullRef qhttp.APIEndpoint = "/ds/get/{username}/{name}/at/{fs}/{hash}/body.csv"
	// AEGetCSVShortRef is the route used to get a body as a csv
	AEGetCSVShortRef qhttp.APIEndpoint = "/ds/get/{username}/{name}/body.csv"
	// AEUnpack unpacks a zip file and sends it back
	AEUnpack qhttp.APIEndpoint = "/ds/unpack"
)
