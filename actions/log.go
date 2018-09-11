package actions

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// Log defines dataset log actions
type Log struct {
	Node *p2p.QriNode
}

// DatasetLog fetches the history of changes to a dataset
func (act Log) DatasetLog(ref repo.DatasetRef, limit, offset int) (rlog []repo.DatasetRef, err error) {
	local, err := ResolveDatasetRef(act.Node, &ref)
	if err != nil {
		return
	}

	if !local {
		return act.Node.RequestDatasetLog(ref, limit, offset)
	}

	for {
		ds, e := dsfs.LoadDataset(act.Node.Repo.Store(), datastore.NewKey(ref.Path))
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
