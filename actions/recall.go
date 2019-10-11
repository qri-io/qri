package actions

import (
	"context"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/dsref"
)

// Recall loads revisions of a dataset from history
func Recall(ctx context.Context, node *p2p.QriNode, str string, ref repo.DatasetRef) (*dataset.Dataset, error) {
	if str == "" {
		return &dataset.Dataset{}, nil
	}

	revs, err := dsref.ParseRevs(str)
	if err != nil {
		return nil, err
	}

	if err := repo.CanonicalizeDatasetRef(node.Repo, &ref); err != nil {
		return nil, err
	}

	res, err := base.LoadRevs(ctx, node.Repo, ref, revs)
	if err != nil {
		return nil, err
	}

	return res, nil
}
