package queries

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfile"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsgraph"
	sql "github.com/qri-io/dataset_sql"
	"github.com/qri-io/qri/repo"
	"time"
)

func NewRequests(store cafs.Filestore, r repo.Repo) *Requests {
	return &Requests{
		store: store,
		repo:  r,
	}
}

type Requests struct {
	store cafs.Filestore
	repo  repo.Repo
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

func (d *Requests) List(p *ListParams, res *dsgraph.QueryResults) error {
	// TODO - finish, need to restore query results graph
	// qr, err := repo.DatasetsQuery(d.repo, query.Query{})
	// qr, err := d.repo.QueryResults()
	// if err != nil {
	// 	return err
	// }
	// *res = qr
	return fmt.Errorf("listing queries is not yet finished")
}

type GetParams struct {
	Path string
	Name string
	Hash string
	Save string
}

func (d *Requests) Get(p *GetParams, res *dataset.Dataset) error {
	// TODO - huh? do we even need to load queries
	q, err := dsfs.LoadDataset(d.store, datastore.NewKey(p.Path))
	if err != nil {
		return err
	}

	*res = *q
	return nil
}

func (r *Requests) Run(ds *dataset.Dataset, res *repo.DatasetRef) error {
	var (
		structure *dataset.Structure
		results   []byte
		err       error
	)

	ds.Timestamp = time.Now()

	// ns, err := r.repo.Namespace()
	// if err != nil {
	// 	return err
	// }

	// TODO - make format output the parsed statement as well
	// to avoid triple-parsing
	// sqlstr, _, remap, err := sql.Format(ds.QueryString)
	// if err != nil {
	// 	return err
	// }
	names, err := sql.StatementTableNames(ds.QueryString)
	if err != nil {
		return err
	}
	// ds.QueryString = sqlstr

	if ds.Resources == nil {
		ds.Resources = map[string]*dataset.Dataset{}
		// collect table references
		for _, name := range names {
			path, err := r.repo.GetPath(name)
			if err != nil {
				// return fmt.Errorf("couldn't find resource for table name: %s", name)
				return err
			}
			// for i, adr := range stmt.References() {
			// if ns[name].String() == "" {
			// 	return fmt.Errorf("couldn't find resource for table name: %s", name)
			// }
			d, err := dsfs.LoadDataset(r.store, path)
			if err != nil {
				return err
			}
			ds.Resources[name] = d
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

	// TODO - restore query results graph
	// rgraph, err := r.repo.QueryResults()
	// if err != nil {
	// 	return err
	// }

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

	// TODO - move this into setting on the dataset outparam
	ds.Structure = structure
	ds.Length = len(results)

	ds.Data, err = r.store.Put(memfile.NewMemfileBytes("data."+ds.Structure.Format.String(), results), false)
	if err != nil {
		fmt.Println("error putting results in store:", err)
		return err
	}

	dspath, err := dsfs.SaveDataset(r.store, ds, false)
	if err != nil {
		fmt.Println("error putting dataset in store:", err)
		return err
	}

	// TODO - need to re-load dataset here to get a dereferenced version
	// lds, err := dsfs.LoadDataset(r.store, dspath)
	// if err != nil {
	// 	return err
	// }

	*res = repo.DatasetRef{
		Dataset: ds,
		Path:    dspath,
	}
	return nil
}
