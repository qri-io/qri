package core

import (
	"fmt"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
)

type SearchRequests struct {
	store cafs.Filestore
	repo  repo.Repo
	// node  *p2p.QriNode
}

func NewSearchRequests(store cafs.Filestore, r repo.Repo) *SearchRequests {
	return &SearchRequests{
		store: store,
		repo:  r,
		// node:  node,
	}
}

func (d *SearchRequests) Search(p *repo.SearchParams, res *[]*repo.DatasetRef) error {
	// if d.node != nil {
	// 	r, err := d.node.Search(p.Query, p.Limit, p.Offset)
	// 	if err != nil {
	// 		return err
	// 	}

	if s, ok := d.repo.(repo.Searchable); ok {
		results, err := s.Search(*p)
		if err != nil {
			return fmt.Errorf("error searching: %s", err.Error())
		}
		*res = results
		return nil
	} else {
		return fmt.Errorf("this repo doesn't support search")
	}

	return nil
}
