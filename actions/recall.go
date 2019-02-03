package actions

import (
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/rev"
)

// Recall loads revisions of a dataset from history
func Recall(node *p2p.QriNode, str string, ref repo.DatasetRef) (*dataset.Dataset, error) {
	if str == "" {
		return &dataset.Dataset{}, nil
	}

	revs, err := rev.ParseRevs(str)
	if err != nil {
		return nil, err
	}

	if err := repo.CanonicalizeDatasetRef(node.Repo, &ref); err != nil {
		return nil, err
	}

	res, err := base.LoadRevs(node.Repo, ref, revs)
	if err != nil {
		return nil, err
	}

	return res, nil
}
