package core

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

type HistoryRequests struct {
	repo repo.Repo
}

func NewHistoryRequests(r repo.Repo) *HistoryRequests {
	return &HistoryRequests{
		repo: r,
	}
}

type LogParams struct {
	Path datastore.Key
	ListParams
}

func (d *HistoryRequests) Log(params *LogParams, res *[]*repo.DatasetRef) (err error) {
	log := []*repo.DatasetRef{}
	limit := params.Limit
	ref := &repo.DatasetRef{Path: params.Path}

	if params.Path.String() == "" {
		return fmt.Errorf("path is required")
	}

	for {
		ref.Dataset, err = dsfs.LoadDataset(d.repo.Store(), ref.Path)
		if err != nil {
			return err
		}
		log = append(log, ref)

		limit--
		if limit == 0 || ref.Dataset.Previous.String() == "" {
			break
		}
		// TODO - clean this up
		_, cleaned := dsfs.RefType(ref.Dataset.Previous.String())
		ref = &repo.DatasetRef{Path: datastore.NewKey(cleaned)}
	}

	*res = log
	return nil
}
