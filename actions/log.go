package actions

import (
	"context"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// DatasetLog fetches the history of changes to a dataset
func DatasetLog(ctx context.Context, node *p2p.QriNode, ref repo.DatasetRef, limit, offset int) (rlog []base.DatasetLogItem, err error) {
	err = repo.CanonicalizeDatasetRef(node.Repo, &ref)
	if err != nil && err != repo.ErrNotFound && err != profile.ErrNotFound {
		// return early on any non "not found" error
		return nil, err
	} else if ref.Path == "" {
		return nil, repo.ErrNotFound
	}

	return base.DatasetLog(ctx, node.Repo, ref, limit, offset, true)
}
