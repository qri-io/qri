package registry

import (
	"fmt"

	"github.com/qri-io/dataset"
)

// Follower is an opt-in interface for registries that wish to
// support dataset following
type Follower interface {
	Get(p *FollowGetParams) ([]*dataset.Dataset, error)
	Follow(p *FollowParams) (*dataset.Dataset, error)
}

// FollowGetParams encapsulates parameters provided to Follower.Get
type FollowGetParams struct {
	Username string `json:"username"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	// TODO(arqu): order by fields are missing
}

// FollowParams encapsulates parameters provided to Follower.Follow
type FollowParams struct {
	Ref    string `json:"ref"`
	Status int    `json:"status"`
}

// ErrFollowingNotSupported is the canonical error to indicate following
// isn't implemented
var ErrFollowingNotSupported = fmt.Errorf("following not supported")

// NilFollower is a basic implementation of Follower which returns
// an error to indicate that following is not supported
type NilFollower bool

// Get returns an error indicating that listing followed datasets is not supported
func (nf NilFollower) Get(p *FollowGetParams) ([]*dataset.Dataset, error) {
	return nil, ErrFollowingNotSupported
}

// Follow returns an error indicating that following a dataset is not supported
func (nf NilFollower) Follow(p *FollowParams) (*dataset.Dataset, error) {
	return nil, ErrFollowingNotSupported
}
