package queries

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	// "github.com/qri-io/castore"
	"github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsgraph"
	sql "github.com/qri-io/dataset_sql"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/core/graphs"
)

// func NewRequests(store *ipfs_datastore.Datastore, ns map[string]datastore.Key) *Requests {
// 	return &Requests{
// 		store: store,
// 		ns:    ns,
// 	}
// }

type Requests struct {
	Store *ipfs_datastore.Datastore
	// namespace graph
	Ns      map[string]datastore.Key
	RGraph  dsgraph.QueryResults
	RqGraph dsgraph.ResourceQueries

	NsGraphPath string
	RqGraphPath string
	RGraphPath  string
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

func (d *Requests) List(p *ListParams, res *dsgraph.QueryResults) error {
	// TODO - is this right?
	*res = d.RGraph
	return nil
}

type GetParams struct {
	Path string
	Name string
	Hash string
}

func (d *Requests) Get(p *GetParams, res *dataset.Dataset) error {
	resource, err := core.GetResource(d.Store, datastore.NewKey(p.Path))
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
		if r.Ns[ref].String() == "" {
			return fmt.Errorf("couldn't find resource for table name: %s", ref)
		}
		q.Resources[mapped] = r.Ns[ref]
	}

	qData, err := q.MarshalJSON()
	if err != nil {
		return err
	}

	qhash, err := r.Store.AddAndPinBytes(qData)
	if err != nil {
		fmt.Println("add bytes error", err.Error())
		return err
	}

	fmt.Printf("query hash: %s\n", qhash)

	qpath := datastore.NewKey("/ipfs/" + qhash)

	cache := r.RGraph[qpath]

	if len(cache) > 0 {
		resource, err = core.GetResource(r.Store, cache[0])
		if err != nil {
			results, err = core.GetStructuredData(r.Store, resource.Path)
		}
	}

	// format, err := dataset.ParseDataFormatString(cmd.Flag("format").Value.String())
	// if err != nil {
	// 	ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
	// }
	resource, results, err = sql.ExecQuery(r.Store, q, func(o *sql.ExecOpt) {
		o.Format = dataset.JsonDataFormat
	})
	if err != nil {
		fmt.Println("exec error", err)
		return err
	}

	resource.Query = qpath

	resultshash, err := r.Store.AddAndPinBytes(results)
	if err != nil {
		return err
	}
	fmt.Printf("results hash: %s\n", resultshash)

	resource.Path = datastore.NewKey("/ipfs/" + resultshash)

	rbytes, err := resource.MarshalJSON()
	if err != nil {
		return err
	}

	rhash, err := r.Store.AddAndPinBytes(rbytes)
	fmt.Printf("result resource hash: %s\n", rhash)

	r.RGraph.AddResult(qpath, datastore.NewKey("/ipfs/"+rhash))
	err = graphs.SaveQueryResultsGraph(r.RGraphPath, r.RGraph)
	if err != nil {
		return err
	}

	for _, key := range q.Resources {
		r.RqGraph.AddQuery(key, qpath)
	}
	err = graphs.SaveResourceQueriesGraph(r.RqGraphPath, r.RqGraph)
	if err != nil {
		return err
	}

	*res = dataset.Dataset{
		Resource: *resource,
	}
	return nil
}
