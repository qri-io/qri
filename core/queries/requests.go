package queries

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo"
	"time"
	// "github.com/qri-io/castore"
	"github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsgraph"
	sql "github.com/qri-io/dataset_sql"
	"github.com/qri-io/qri/core"
)

func NewRequests(store *ipfs_datastore.Datastore, r repo.Repo) *Requests {
	return &Requests{
		store: store,
		repo:  r,
	}
}

type Requests struct {
	store *ipfs_datastore.Datastore
	repo  repo.Repo
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

func (d *Requests) List(p *ListParams, res *dsgraph.QueryResults) error {
	// TODO - is this right?
	qr, err := d.repo.QueryResults()
	if err != nil {
		return err
	}
	*res = qr
	return nil
}

type GetParams struct {
	Path string
	Name string
	Hash string
}

func (d *Requests) Get(p *GetParams, res *dataset.Dataset) error {
	// structure, err := core.GetStructure(d.store, datastore.NewKey(p.Path))
	_, err := core.GetStructure(d.store, datastore.NewKey(p.Path))
	if err != nil {
		return err
	}

	*res = dataset.Dataset{
	// TODO - finish
	// Structure: *structure,
	}
	return nil
}

func (r *Requests) Run(ds *dataset.Dataset, res *dataset.Dataset) error {
	var (
		structure *dataset.Structure
		results   []byte
	)

	ds.Timestamp = time.Now()

	ns, err := r.repo.Namespace()
	if err != nil {
		return err
	}

	// TODO - make format output the parsed statement as well
	// to avoid triple-parsing
	sqlstr, _, remap, err := sql.Format(ds.Query)
	if err != nil {
		return err
	}

	ds.Query = sqlstr

	// ds := &dataset.Dataset{
	// 	QuerySyntax: p.Syntax,
	// 	Query:       sqlstr,
	// 	Resources:   map[string]datastore.Key{},
	// }

	// q := &dataset.Query{
	// 	Syntax:    p.Syntax,
	// 	Resources: map[string]datastore.Key{},
	// 	Statement: sqlstr,
	// 	// TODO - set query schema
	// }

	if ds.Resources == nil {
		ds.Resources = map[string]datastore.Key{}
		// collect table references
		for mapped, ref := range remap {
			// for i, adr := range stmt.References() {
			if ns[ref].String() == "" {
				return fmt.Errorf("couldn't find resource for table name: %s", ref)
			}
			ds.Resources[mapped] = ns[ref]
		}
	}

	// dsData, err := ds.MarshalJSON()
	// if err != nil {
	// 	return err
	// }
	// dshash, err := r.store.AddAndPinBytes(dsData)
	// if err != nil {
	// 	fmt.Println("add bytes error", err.Error())
	// 	return err
	// }

	// TODO - restore query hash discovery
	// fmt.Printf("query hash: %s\n", dshash)

	// dspath := datastore.NewKey("/ipfs/" + dshash)

	rgraph, err := r.repo.QueryResults()
	if err != nil {
		return err
	}
	// cache := rgraph[qpath]

	// if len(cache) > 0 {
	// 	resource, err = core.GetStructure(r.store, cache[0])
	// 	if err != nil {
	// 		results, err = core.GetStructuredData(r.store, resource.Path)
	// 	}
	// }

	// TODO - detect data format from passed-in results structure
	structure, results, err = sql.Exec(r.store, ds, func(o *sql.ExecOpt) {
		o.Format = dataset.CsvDataFormat
	})
	if err != nil {
		fmt.Println("exec error", err)
		return err
	}

	resultshash, err := r.store.AddAndPinBytes(results)
	if err != nil {
		return err
	}
	fmt.Printf("results hash: %s\n", resultshash)

	ds.Data = datastore.NewKey("/ipfs/" + resultshash)

	stbytes, err := structure.MarshalJSON()
	if err != nil {
		return err
	}

	sthash, err := r.store.AddAndPinBytes(stbytes)
	fmt.Printf("result structure hash: %s\n", sthash)

	ds.Structure = datastore.NewKey("/ipfs/" + sthash)

	dsdata, err := ds.MarshalJSON()
	if err != nil {
		return err
	}
	dshash, err := r.store.AddAndPinBytes(dsdata)
	if err != nil {
		return err
	}

	dspath := datastore.NewKey("/ipfs/" + dshash)

	rgraph.AddResult(dspath, dspath)
	err = r.repo.SaveQueryResults(rgraph)
	if err != nil {
		return err
	}

	rqgraph, err := r.repo.ResourceQueries()
	if err != nil {
		return err
	}

	for _, key := range ds.Resources {
		rqgraph.AddQuery(key, dspath)
	}
	err = r.repo.SaveResourceQueries(rqgraph)
	if err != nil {
		return err
	}

	*res = *ds
	return nil
}
