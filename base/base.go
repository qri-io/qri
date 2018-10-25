// Package base defines business that operates on local data
// it's main job is to composing APIs from the lower half of our tech stack,
// providing uniform functions for higher up packages, mainly p2p and actions.
// p2p and actions should use base as the only way of operate on the local repo
// Here's some ascii art to clarify the stack:
//
//    ┌───────────────┐ ┌───────────────┐
//    │      cmd      │ │     api       │
//    └───────────────┘ └───────────────┘
//    ┌─────────────────────────────────┐
//    │               lib               │
//    └─────────────────────────────────┘
//    ┌─────────────────────────────────┐
//    │             actions             │
//    └─────────────────────────────────┘
//    ┌───────────────────────┐
//    │          p2p          │
//    └───────────────────────┘
//    ┌─────────────────────────────────┐
//    │              base               │  <-- you are here
//    └─────────────────────────────────┘
//    ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
//    │ repo │ │ dsfs │ │ cafs │ │ ...  │
//    └──────┘ └──────┘ └──────┘ └──────┘
//
// There are packages omitted from this diagram, but these are the vitals.
// base functions mainly work with repo.Repo instances, using repo interface methods
// and related packages to do their work. This is part of a larger pattern of
// having actions rely on lower level interfaces wherever possible to enhance
// configurability
package base

import (
	golog "github.com/ipfs/go-log"
)

var log = golog.Logger("base")
