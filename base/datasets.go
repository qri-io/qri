package base

import (
	"fmt"

	datastore "github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

// ListDatasets lists datasets from a repo
func ListDatasets(r repo.Repo, limit, offset int, RPC, publishedOnly bool) (res []repo.DatasetRef, err error) {
	store := r.Store()
	res, err = r.References(limit, offset)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error getting dataset list: %s", err.Error())
	}

	if publishedOnly {
		pub := make([]repo.DatasetRef, len(res))
		i := 0
		for _, ref := range res {
			if ref.Published {
				pub[i] = ref
				i++
			}
		}
		res = pub[:i]
	}

	renames := repo.NewNeedPeernameRenames()
	for i, ref := range res {
		// May need to change peername.
		if err := repo.CanonicalizeProfile(r, &res[i], &renames); err != nil {
			return nil, fmt.Errorf("error canonicalizing dataset peername: %s", err.Error())
		}

		ds, err := dsfs.LoadDataset(store, datastore.NewKey(ref.Path))
		if err != nil {
			return nil, fmt.Errorf("error loading path: %s, err: %s", ref.Path, err.Error())
		}
		res[i].Dataset = ds.Encode()
		if RPC {
			res[i].Dataset.Structure.Schema = nil
		}
	}

	// TODO: If renames.Renames is non-empty, apply it to r
	return
}
