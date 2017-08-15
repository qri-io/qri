package peer

import (
	"github.com/ipfs/go-datastore"
	// "github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo/profile"
)

type Repo struct {
	Profile   *profile.Profile
	Namespace map[string]datastore.Key
}
