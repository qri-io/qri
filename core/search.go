package core

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/actions"
	"github.com/qri-io/registry/regclient"
)

// SearchRequests encapsulates business logic for the qri search
// command
type SearchRequests struct {
	cli  *rpc.Client
	repo repo.Repo
}

// NewSearchRequests creates a SearchRequests pointer from either a repo
// or an rpc.Client
func NewSearchRequests(r repo.Repo, cli *rpc.Client) *SearchRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewSearchRequests"))
	}
	return &SearchRequests{
		cli:  cli,
		repo: &actions.Registry{r},
	}
}

// CoreRequestsName implements the requests
func (sr SearchRequests) CoreRequestsName() string { return "search" }

// SearchParams defines paremeters for the search Method
type SearchParams struct {
	QueryString string
	Limit       int
	Offset      int
}

// SearchResult struct
type SearchResult struct {
	Type, ID string
	Value    interface{}
}

// Search queries for items on qri related to given parameters
func (sr *SearchRequests) Search(p *SearchParams, results *[]SearchResult) error {
	if sr.cli != nil {
		return sr.cli.Call("SearchRequests.Search", p, results)
	}
	reg := sr.repo.Registry()
	if p == nil {
		return fmt.Errorf("error: search params cannot be nil")
	}
	params := &regclient.SearchParams{p.QueryString, nil, p.Limit, p.Offset}

	regResults, err := reg.Search(params)
	if err != nil {
		return err
	}

	searchResults := make([]SearchResult, len(regResults))
	for i, result := range regResults {
		searchResults[i].Type = result.Type
		searchResults[i].ID = result.ID
		searchResults[i].Value = result.Value
	}
	*results = searchResults
	return nil
}
