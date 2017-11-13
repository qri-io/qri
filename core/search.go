package core

import (
	"fmt"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
)

type SearchRequests struct {
	store cafs.Filestore
	repo  repo.Repo
	// node  *p2p.QriNode
}

func NewSearchRequests(r repo.Repo) *SearchRequests {
	return &SearchRequests{
		repo: r,
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

type ReindexSearchParams struct {
	// no args for reindex
}

func (d *SearchRequests) Reindex(p *ReindexSearchParams, done *bool) error {
	if fsr, ok := d.repo.(*fs_repo.Repo); ok {
		err := fsr.UpdateSearchIndex(d.repo.Store())
		if err != nil {
			return fmt.Errorf("error reindexing: %s", err.Error())
		}
		*done = true
		return nil
	}

	return fmt.Errorf("search reindexing is currently only supported on file-system repos")
}
