package actions

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// DatasetLog fetches the history of changes to a dataset
func DatasetLog(node *p2p.QriNode, ref repo.DatasetRef, limit, offset int) (rlog []repo.DatasetRef, err error) {
	local, err := ResolveDatasetRef(node, &ref)
	if err != nil {
		return
	}

	if !local {
		return node.RequestDatasetLog(ref, limit, offset)
	}

	for {
		ds, e := dsfs.LoadDataset(node.Repo.Store(), datastore.NewKey(ref.Path))
		if e != nil {
			return nil, e
		}
		ref.Dataset = ds.Encode()

		offset--
		if offset > 0 {
			continue
		}

		rlog = append(rlog, ref)

		limit--
		if limit == 0 || ref.Dataset.PreviousPath == "" {
			break
		}
		ref.Path = ref.Dataset.PreviousPath
	}

	return rlog, nil
}
