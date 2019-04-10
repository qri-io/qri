package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/registry/regclient"
)

// SearchMethods search qri for datasets
type SearchMethods interface {
	Methods
	Search(p *SearchParams, results *[]SearchResult) error
}

// NewSearchMethods creates a searchMethods from an instance
func NewSearchMethods(inst Instance) SearchMethods {
	return searchMethods{
		cli:  inst.RPC(),
		node: inst.Node(),
	}
}

// searchMethods encapsulates business logic for the qri search
// command
type searchMethods struct {
	cli  *rpc.Client
	node *p2p.QriNode
}

// MethodsKind implements the requests
func (sr searchMethods) MethodsKind() string { return "SearchMethods" }

// SearchParams defines paremeters for the search Method
type SearchParams struct {
	QueryString string `json:"q"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

// SearchResult struct
type SearchResult struct {
	Type, ID string
	Value    interface{}
}

// Search queries for items on qri related to given parameters
func (sr searchMethods) Search(p *SearchParams, results *[]SearchResult) error {
	if sr.cli != nil {
		return sr.cli.Call("SearchMethods.Search", p, results)
	}
	if p == nil {
		return fmt.Errorf("error: search params cannot be nil")
	}

	reg := sr.node.Repo.Registry()
	if reg == nil {
		return repo.ErrNoRegistry
	}
	params := &regclient.SearchParams{
		QueryString: p.QueryString,
		Limit:       p.Limit,
		Offset:      p.Offset,
	}

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
