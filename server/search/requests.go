package search

import (
	"fmt"
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
)

func NewSearchRequests(store cafs.Filestore, r repo.Repo) *SearchRequests {
	return &SearchRequests{
		store: store,
		repo:  r,
		// node:  node,
	}
}

type SearchRequests struct {
	store cafs.Filestore
	repo  repo.Repo
	// node  *p2p.QriNode
}

type SearchParams struct {
	Query  string
	Limit  int
	Offset int
}

func (d *SearchRequests) Search(p *SearchParams, res *[]*repo.DatasetRef) error {
	// if d.node != nil {
	// 	r, err := d.node.Search(p.Query, p.Limit, p.Offset)
	// 	if err != nil {
	// 		return err
	// 	}

	if s, ok := d.repo.(repo.Searchable); ok {
		results, err := s.Search(p.Query)
		if err != nil {
			return err
		}
		*res = results
		return nil
	} else {
		return fmt.Errorf("this repo doesn't support search")
	}

	// 	*res = r
	// 	return nil
	// }

	// r, err := search.Search(d.repo, d.store, search.NewDatasetQuery(p.Query, p.Limit, p.Offset))
	// if err != nil {
	// 	return err
	// }
	// r, err := search.Search(p.Query)

	// *res = r
	return nil
}
