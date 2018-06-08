package core

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/repo"
	"github.com/qri-io/registry/regclient"
)

// SearchRequests encapsulates business logic for the qri search
// command
type SearchRequests struct {
	repo repo.Repo
	cli  *rpc.Client
}

// CoreRequestsName implements the requests
func (d SearchRequests) CoreRequestsName() string { return "search" }

// NewSearchRequests creates a SearchRequests pointer from either a repo
// or an rpc.Client
func NewSearchRequests(r repo.Repo, cli *rpc.Client) *SearchRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewSearchRequests"))
	}

	return &SearchRequests{
		repo: r,
		cli:  cli,
	}
}

// SearchResult struct
type SearchResult struct {
	Type, ID string
	Value    interface{}
}

// Search queries for items on qri related to given parameters
func (d *SearchRequests) Search(p *regclient.SearchParams, res *[]SearchResult) error {
	if d.cli != nil {
		return d.cli.Call("SearchRequests.Search", p, res)
	}

	reg := d.repo.Registry()

	results, err := reg.Search(p)
	if err != nil {
		return err
	}

	searchResults := make([]SearchResult, len(results))
	for i, result := range results {
		searchResults[i].Type = result.Type
		searchResults[i].ID = result.ID
		searchResults[i].Value = result.Value
	}
	*res = searchResults
	return nil
}

// ReindexSearchParams defines parmeters for
// the Reindex method
// type ReindexSearchParams struct {
// 	// no args for reindex
// }

// Reindex instructs a qri node to re-calculate it's search index
// func (d *SearchRequests) Reindex(p *ReindexSearchParams, done *bool) error {
// 	if d.cli != nil {
// 		return d.cli.Call("SearchRequests.Reindex", p, done)
// 	}

// 	if fsr, ok := d.repo.(*fsrepo.Repo); ok {
// 		err := fsr.UpdateSearchIndex(d.repo.Store())
// 		if err != nil {
// 			log.Debug(err.Error())
// 			return fmt.Errorf("error reindexing: %s", err.Error())
// 		}
// 		*done = true
// 		return nil
// 	}

// 	return fmt.Errorf("search reindexing is currently only supported on file-system repos")
// }
