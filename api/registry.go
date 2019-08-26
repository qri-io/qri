package api

import (
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// RegistryHandlers wraps a requests struct to interface with http.HandlerFunc
type RegistryHandlers struct {
	lib.RegistryRequests
	repo repo.Repo
}

// NewRegistryHandlers allocates a RegistryHandlers pointer
func NewRegistryHandlers(node *p2p.QriNode) *RegistryHandlers {
	req := lib.NewRegistryRequests(node, nil)
	h := RegistryHandlers{*req, node.Repo}
	return &h
}
