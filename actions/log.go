package actions

import (
	"context"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// DatasetLog fetches the history of changes to a dataset
// TODO (b5) - implement remote log fetching
func DatasetLog(ctx context.Context, node *p2p.QriNode, ref repo.DatasetRef, limit, offset int) (rlog []repo.DatasetRef, err error) {
	local, err := ResolveDatasetRef(ctx, node, nil, "", &ref)
	if err != nil {
		return
	}

	if !local {
		return node.RequestDatasetLog(ctx, ref, limit, offset)
	}

	return base.DatasetLog(ctx, node.Repo, ref, limit, offset, true)
}
