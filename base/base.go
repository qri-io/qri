// Package base defines business that operates on local data.
// it's main job is to composing APIs from the lower half of our tech stack,
// providing uniform functions for higher up packages, mainly p2p and lib.
// p2p and lib use base as the only way of operate on the local repo
// Here's some ascii art to clarify the stack:
//
//    ┌───────────────┐ ┌───────────────┐
//    │      cmd      │ │     api       │
//    └───────────────┘ └───────────────┘
//    ┌─────────────────────────────────┐
//    │               lib               │
//    └─────────────────────────────────┘
//    ┌───────────────────────┐
//    │          p2p          │
//    └───────────────────────┘
//    ┌─────────────────────────────────┐
//    │              base               │  <-- you are here
//    └─────────────────────────────────┘
//    ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
//    │ repo │ │ dsfs │ │ qfs  │ │ ...  │
//    └──────┘ └──────┘ └──────┘ └──────┘
//
// There are packages omitted from this diagram, but these are the vitals.
// base functions mainly work with repo.Repo instances, using repo interface methods
// and related packages to do their work. This is part of a larger pattern of
// having lib rely on lower level interfaces wherever possible to enhance
// configurability
package base

import (
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
