package queries

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
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

func (d *Requests) List(p *ListParams, res *[]*repo.DatasetRef) error {
	results, err := d.repo.GetQueryLogs(p.Limit, p.Offset)
	if err != nil {
		return err
	}

	for _, ref := range results {
		if ds, err := dsfs.LoadDataset(d.store, ref.Path); err == nil {
			ref.Dataset = ds
		}
	}
	*res = results
	return nil
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

type RunParams struct {
	// embed all execution options
	sql.ExecOpt
	SaveName string
	Dataset  *dataset.Dataset
}

func (r *Requests) Run(p *RunParams, res *repo.DatasetRef) error {
	var (
		structure *dataset.Structure
		results   []byte
		err       error
		ds        = p.Dataset
	)

	if ds == nil {
		return fmt.Errorf("dataset is required")
	}

	ds.Timestamp = time.Now()

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
				fmt.Println("load dataset error:", err.Error())
				return err
			}
			ds.Resources[name] = d
		}
	}

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

	ds.Data, err = r.store.Put(memfs.NewMemfileBytes("data."+ds.Structure.Format.String(), results), false)
	if err != nil {
		fmt.Println("error putting results in store:", err)
		return err
	}

	pin := p.SaveName != ""

	dspath, err := dsfs.SaveDataset(r.store, ds, pin)
	if err != nil {
		fmt.Println("error putting dataset in store:", err)
		return err
	}

	if p.SaveName != "" {
		if err := r.repo.PutName(p.SaveName, dspath); err != nil {
			fmt.Println("error saving dataset name:", err)
			return err
		}
	}

	if err := dsfs.DerefDatasetStructure(r.store, ds); err != nil {
		return err
	}
	if err := dsfs.DerefDatasetQuery(r.store, ds); err != nil {
		return err
	}

	ref := &repo.DatasetRef{Name: p.SaveName, Path: dspath, Dataset: ds}

	if err := r.repo.LogQuery(ref); err != nil {
		return err
	}

	*res = *ref
	return nil
}
