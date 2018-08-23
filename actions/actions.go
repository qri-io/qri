// Package actions provides canonical business logic that operates on Repos
// to get higher-order functionality. Actions use only Repo methods
// to do their work, allowing them to be used across any repo.Repo implementation
package actions

import (
	golog "github.com/ipfs/go-log"
)

var log = golog.Logger("actions")
