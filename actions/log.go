package actions

import (
	"context"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// DatasetLog fetches the history of changes to a dataset
func DatasetLog(ctx context.Context, node *p2p.QriNode, ref repo.DatasetRef, limit, offset int) (rlog []base.DatasetLogItem, err error) {
	local, err := ResolveDatasetRef(ctx, node, nil, "", &ref)
	if err != nil {
		return
	}

	if !local {
		res, err := node.RequestDatasetLog(ctx, ref, limit, offset)
		if err != nil {
			return nil, err
		}
		rlog = make([]base.DatasetLogItem, len(res))
		for i, v := range res {
			rlog[i] = base.DatasetLogItem{
				Ref: repo.ConvertToDsref(v),
			}
			if v.Dataset != nil {
				rlog[i].Timestamp = v.Dataset.Commit.Timestamp
				rlog[i].CommitTitle = v.Dataset.Commit.Title
				rlog[i].CommitMessage = v.Dataset.Commit.Message
			}
		}
		return rlog, err
	}

	return base.DatasetLog(ctx, node.Repo, ref, limit, offset, true)
}
