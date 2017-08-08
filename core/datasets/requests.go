package datasets

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/castore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
)

func NewRequests(store castore.Datastore, ns map[string]datastore.Key) *Requests {
	return &Requests{
		store: store,
		ns:    ns,
	}
}

type Requests struct {
	store castore.Datastore
	// namespace graph
	ns map[string]datastore.Key
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

func (d *Requests) List(p *ListParams, res *map[string]datastore.Key) error {
	*res = d.ns
	return nil
}

type GetParams struct {
	Path string
	Name string
	Hash string
}

func (d *Requests) Get(p *GetParams, res *dataset.Dataset) error {

	resource, err := core.GetResource(d.store, datastore.NewKey(p.Path))
	if err != nil {
		return err
	}

	*res = dataset.Dataset{
		Resource: *resource,
	}
	return nil
}
