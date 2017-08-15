package queries

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo"
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
	resource, err := core.GetResource(d.store, datastore.NewKey(p.Path))
	if err != nil {
		return err
	}

	*res = dataset.Dataset{
		Resource: *resource,
	}
	return nil
}

func (r *Requests) Run(p *dataset.Query, res *dataset.Dataset) error {
	var (
		resource *dataset.Resource
		results  []byte
	)

	ns, err := r.repo.Namespace()
	if err != nil {
		return err
	}

	// TODO - make format output the parsed statement as well
	// to avoid triple-parsing
	sqlstr, _, remap, err := sql.Format(p.Statement)
	if err != nil {
		return err
	}

	q := &dataset.Query{
		Syntax:    p.Syntax,
		Resources: map[string]datastore.Key{},
		Statement: sqlstr,
		// TODO - set query schema
	}

	// collect table references
	for mapped, ref := range remap {
		// for i, adr := range stmt.References() {
		if ns[ref].String() == "" {
			return fmt.Errorf("couldn't find resource for table name: %s", ref)
		}
		q.Resources[mapped] = ns[ref]
	}

	qData, err := q.MarshalJSON()
	if err != nil {
		return err
	}

	qhash, err := r.store.AddAndPinBytes(qData)
	if err != nil {
		fmt.Println("add bytes error", err.Error())
		return err
	}

	fmt.Printf("query hash: %s\n", qhash)

	qpath := datastore.NewKey("/ipfs/" + qhash)

	rgraph, err := r.repo.QueryResults()
	cache := rgraph[qpath]

	if len(cache) > 0 {
		resource, err = core.GetResource(r.store, cache[0])
		if err != nil {
			results, err = core.GetStructuredData(r.store, resource.Path)
		}
	}

	// format, err := dataset.ParseDataFormatString(cmd.Flag("format").Value.String())
	// if err != nil {
	// 	ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
	// }
	resource, results, err = sql.ExecQuery(r.store, q, func(o *sql.ExecOpt) {
		o.Format = dataset.JsonDataFormat
	})
	if err != nil {
		fmt.Println("exec error", err)
		return err
	}

	resource.Query = qpath

	resultshash, err := r.store.AddAndPinBytes(results)
	if err != nil {
		return err
	}
	fmt.Printf("results hash: %s\n", resultshash)

	resource.Path = datastore.NewKey("/ipfs/" + resultshash)

	rbytes, err := resource.MarshalJSON()
	if err != nil {
		return err
	}

	rhash, err := r.store.AddAndPinBytes(rbytes)
	fmt.Printf("result resource hash: %s\n", rhash)

	rgraph.AddResult(qpath, datastore.NewKey("/ipfs/"+rhash))
	err = r.repo.SaveQueryResults(rgraph)
	if err != nil {
		return err
	}

	rqgraph, err := r.repo.ResourceQueries()
	if err != nil {
		return err
	}

	for _, key := range q.Resources {
		rqgraph.AddQuery(key, qpath)
	}
	err = r.repo.SaveResourceQueries(rqgraph)
	if err != nil {
		return err
	}

	*res = dataset.Dataset{
		Resource: *resource,
	}
	return nil
}
