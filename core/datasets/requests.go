package datasets

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/castore/ipfs"
	"github.com/qri-io/qri/core/graphs"
	// "github.com/qri-io/castore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
)

func NewRequests(store *ipfs_datastore.Datastore, ns map[string]datastore.Key, nspath string) *Requests {
	return &Requests{
		store:       store,
		ns:          ns,
		nsGraphPath: nspath,
	}
}

type Requests struct {
	store *ipfs_datastore.Datastore
	// namespace graph
	ns          map[string]datastore.Key
	nsGraphPath string
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

type SaveParams struct {
	Name    string
	Dataset *dataset.Dataset
}

func (r *Requests) Save(p *SaveParams, res *dataset.Dataset) error {
	resource := p.Dataset.Resource

	rdata, err := resource.MarshalJSON()
	if err != nil {
		return err
	}
	qhash, err := r.store.AddAndPinBytes(rdata)
	if err != nil {
		return err
	}

	r.ns[p.Name] = datastore.NewKey("/ipfs/" + qhash)
	if err := graphs.SaveNamespaceGraph(r.nsGraphPath, r.ns); err != nil {
		return err
	}

	*res = dataset.Dataset{
		Resource: resource,
	}
	return nil
}

type DeleteParams struct {
	Name string
}

func (r *Requests) Delete(p *DeleteParams, ok *bool) error {
	// TODO - unpin resource and data
	// resource := p.Dataset.Resource

	delete(r.ns, p.Name)
	if err := graphs.SaveNamespaceGraph(r.nsGraphPath, r.ns); err != nil {
		return err
	}
	*ok = true
	return nil
}
