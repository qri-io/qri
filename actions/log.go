package actions

import (
	"github.com/qri-io/qri/base"
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

	return base.DatasetLog(node.Repo, ref, limit, offset, true)
}
