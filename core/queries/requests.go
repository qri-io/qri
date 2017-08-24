package queries

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/castore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsgraph"
	sql "github.com/qri-io/dataset_sql"
	"github.com/qri-io/qri/repo"
	"time"
)

func NewRequests(store castore.Datastore, r repo.Repo) *Requests {
	return &Requests{
		store: store,
		repo:  r,
	}
}

type Requests struct {
	store castore.Datastore
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
	// TODO - huh? do we even need to load queries
	q, err := dataset.LoadDataset(d.store, datastore.NewKey(p.Path))
	if err != nil {
		return err
	}

	*res = *q
	return nil
}

func (r *Requests) Run(ds *dataset.Dataset, res *dataset.Dataset) error {
	var (
		structure *dataset.Structure
		results   []byte
		err       error
	)

	ds.Timestamp = time.Now()

	ns, err := r.repo.Namespace()
	if err != nil {
		return err
	}

	// TODO - make format output the parsed statement as well
	// to avoid triple-parsing
	sqlstr, _, remap, err := sql.Format(ds.QueryString)
	if err != nil {
		return err
	}

	ds.QueryString = sqlstr

	// fmt.Println(sqlstr)

	if ds.Resources == nil {
		ds.Resources = map[string]*dataset.Dataset{}
		// collect table references
		for mapped, ref := range remap {
			// for i, adr := range stmt.References() {
			if ns[ref].String() == "" {
				return fmt.Errorf("couldn't find resource for table name: %s", ref)
			}
			d, err := dataset.LoadDataset(r.store, ns[ref])
			if err != nil {
				return err
			}
			ds.Resources[mapped] = d
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
		o.Format = dataset.JsonDataFormat
	})
	if err != nil {
		fmt.Println("exec error", err)
		return err
	}

	// TODO - move this into setting on the dataset outparam
	ds.Structure = structure
	ds.Length = len(results)

	ds.Data, err = r.store.Put(results)
	if err != nil {
		fmt.Println("error putting results in store:", err)
		return err
	}

	dspath, err := ds.Save(r.store)
	if err != nil {
		fmt.Println("error putting dataset in store:", err)
		return err
	}

	rgraph.AddResult(dspath, dspath)
	err = r.repo.SaveQueryResults(rgraph)
	if err != nil {
		return err
	}

	// rqgraph, err := r.repo.ResourceQueries()
	// if err != nil {
	// 	return err
	// }

	// TODO - restore
	// for _, key := range ds.Resources {
	// 	rqgraph.AddQuery(key, dspath)
	// }
	// err = r.repo.SaveResourceQueries(rqgraph)
	// if err != nil {
	// 	return err
	// }

	*res = *ds
	return nil
}
