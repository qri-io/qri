package server

import (
	"fmt"
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/core/search"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

func NewSearchRequests(store cafs.Filestore, r repo.Repo, node *p2p.QriNode) *SearchRequests {
	return &SearchRequests{
		store: store,
		repo:  r,
		node:  node,
	}
}

type SearchRequests struct {
	store cafs.Filestore
	repo  repo.Repo
	node  *p2p.QriNode
}

type SearchParams struct {
	Query  string
	Limit  int
	Offset int
}

func (d *SearchRequests) Search(p *SearchParams, res *[]*repo.DatasetRef) error {
	if d.node != nil {
		msg, err := d.node.Search(p.Query, p.Limit, p.Offset)
		if err != nil {
			return err
		}

		fmt.Println(msg)

		*res = []*repo.DatasetRef{}
		return nil
	}

	r, err := search.Search(d.repo, d.store, search.NewDatasetQuery(p.Query, p.Limit, p.Offset))
	if err != nil {
		return err
	}

	*res = r
	return nil
}
