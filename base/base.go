// Package base defines functions that operate on local data
package base

import (
	"fmt"
	"time"

	golog "github.com/ipfs/go-log"
)

var (
	log = golog.Logger("base")
	// OpenFileTimeoutDuration determines the maximium amount of time to wait for
	// a Filestore to open a file. Some filestores (like IPFS) fallback to a
	// network request when it can't find a file locally. Setting a short timeout
	// prevents waiting for a slow network response, at the expense of leaving
	// files unresolved.
	// TODO (b5) - allow -1 duration as a sentinel value for no timeout
	OpenFileTimeoutDuration = time.Millisecond * 250
)

var (
	// ErrUnlistableReferences is an error for when listing references encounters
	// some problem, but these problems are non-fatal
	ErrUnlistableReferences = fmt.Errorf("Warning: Some datasets could not be listed, because of invalid state. These datasets still exist in your repository, but have references that cannot be resolved. This will be fixed in a future version. You can see all datasets by using `qri list --raw`")
	// ErrNameTaken is an error for when a name for a new dataset is already being used
	ErrNameTaken = fmt.Errorf("name already in use")
	// ErrDatasetLogTimeout is an error for when getting the datasetLog times out
	ErrDatasetLogTimeout = fmt.Errorf("datasetLog: timeout")
)
