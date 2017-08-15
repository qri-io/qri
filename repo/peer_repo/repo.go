package peer_repo

import (
	"github.com/ipfs/go-datastore"
	"time"
	// "github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo/profile"
)

type Repo struct {
	Profile   *profile.Profile         `json:"profile"`
	LastFetch time.Time                `json:"lastFetch"`
	Namespace map[string]datastore.Key `json:"namespace"`
}
