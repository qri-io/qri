package datasets

import (
	"fmt"
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

func (d *Requests) List(p *ListParams, res *[]*dataset.Dataset) error {
	replies := make([]*dataset.Dataset, p.Limit)
	i := 0
	// TODO - generate a sorted copy of keys, iterate through that respecting
	// limit & offset
	for name, key := range d.ns {
		if i >= p.Limit {
			break
		}

		v, err := d.store.Get(key)
		if err != nil {
			return err
		}
		resource, err := dataset.UnmarshalResource(v)
		if err != nil {
			return err
		}
		replies[i] = &dataset.Dataset{
			Metadata: dataset.Metadata{
				Title:   name,
				Subject: key,
			},
			Resource: *resource,
		}
		i++
	}
	*res = replies[:i]
	return nil
}

type GetParams struct {
	Path datastore.Key
	Name string
	Hash string
}

func (d *Requests) Get(p *GetParams, res *dataset.Dataset) error {
	resource, err := core.GetResource(d.store, p.Path)
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
	Path datastore.Key
}

func (r *Requests) Delete(p *DeleteParams, ok *bool) error {
	// TODO - unpin resource and data
	// resource := p.Dataset.Resource
	if p.Name == "" && p.Path.String() != "" {
		for name, val := range r.ns {
			if val.Equal(p.Path) {
				p.Name = name
			}
		}
	}

	if p.Name == "" {
		return fmt.Errorf("couldn't find dataset: %s", p.Path.String())
	} else if r.ns[p.Name] == datastore.NewKey("") {
		return fmt.Errorf("couldn't find dataset: %s", p.Name)
	}

	delete(r.ns, p.Name)
	if err := graphs.SaveNamespaceGraph(r.nsGraphPath, r.ns); err != nil {
		return err
	}
	*ok = true
	return nil
}

type StructuredDataParams struct {
	Path datastore.Key
}

type StructuredData struct {
	Path datastore.Key `json:"path"`
	Data interface{}   `json:"data"`
}

func (r *Requests) StructuredData(p *StructuredDataParams, data *StructuredData) error {
	v, err := r.store.Get(p.Path)
	if err != nil {
		return err
	}

	switch t := v.(type) {
	case []byte:
		v = string(t)
	}

	*data = StructuredData{
		Path: p.Path,
		Data: v,
	}
	return nil
}
