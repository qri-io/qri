package core

import (
	"fmt"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	sql "github.com/qri-io/dataset_sql"
	"github.com/qri-io/qri/repo"
)

type QueryRequests struct {
	repo repo.Repo
}

func NewQueryRequests(r repo.Repo) *QueryRequests {
	return &QueryRequests{
		repo: r,
	}
}

func (d *QueryRequests) List(p *ListParams, res *[]*repo.DatasetRef) error {
	results, err := d.repo.GetQueryLogs(p.Limit, p.Offset)
	if err != nil {
		return fmt.Errorf("error getting query logs: %s", err.Error())
	}

	for _, ref := range results {
		if ds, err := dsfs.LoadDataset(d.repo.Store(), ref.Path); err == nil {
			ref.Dataset = ds
		}
	}
	*res = results
	return nil
}

type GetQueryParams struct {
	Path string
	Name string
	Hash string
	Save string
}

func (d *QueryRequests) Get(p *GetQueryParams, res *dataset.Dataset) error {
	// TODO - huh? do we even need to load queries
	q, err := dsfs.LoadDataset(d.repo.Store(), datastore.NewKey(p.Path))
	if err != nil {
		return fmt.Errorf("error loading dataset: %s", err.Error())
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

func (r *QueryRequests) Run(p *RunParams, res *repo.DatasetRef) error {
	var (
		store     = r.repo.Store()
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
		return fmt.Errorf("error getting statement table names: %s", err.Error())
	}
	// ds.QueryString = sqlstr

	if ds.Resources == nil {
		ds.Resources = map[string]*dataset.Dataset{}
		// collect table references
		for _, name := range names {
			path, err := r.repo.GetPath(name)
			if err != nil {
				return fmt.Errorf("error getting path to dataset %s: %s", name, err.Error())
			}
			// for i, adr := range stmt.References() {
			// if ns[name].String() == "" {
			// 	return fmt.Errorf("couldn't find resource for table name: %s", name)
			// }
			d, err := dsfs.LoadDataset(store, path)
			if err != nil {
				return fmt.Errorf("error loading dataset: %s", err.Error())
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
	// 	resource, err = core.GetStructure(store, cache[0])
	// 	if err != nil {
	// 		results, err = core.GetStructuredData(store, resource.Path)
	// 	}
	// }

	// TODO - detect data format from passed-in results structure
	structure, results, err = sql.Exec(store, ds, func(o *sql.ExecOpt) {
		o.Format = dataset.CsvDataFormat
	})
	if err != nil {
		return fmt.Errorf("error executing query: %s", err.Error())
	}

	// TODO - move this into setting on the dataset outparam
	ds.Structure = structure
	ds.Length = len(results)

	ds.Data, err = store.Put(memfs.NewMemfileBytes("data."+ds.Structure.Format.String(), results), false)
	if err != nil {
		return fmt.Errorf("error putting results in store: %s", err.Error())
	}

	pin := p.SaveName != ""

	dspath, err := dsfs.SaveDataset(store, ds, pin)
	if err != nil {
		return fmt.Errorf("error putting dataset in store: %s", err.Error())
	}

	if p.SaveName != "" {
		if err := r.repo.PutName(p.SaveName, dspath); err != nil {
			return fmt.Errorf("error saving dataset name: %s", err.Error())
		}
	}

	if err := dsfs.DerefDatasetStructure(store, ds); err != nil {
		return fmt.Errorf("error dereferencing dataset structure: %s", err.Error())
	}
	if err := dsfs.DerefDatasetQuery(store, ds); err != nil {
		return fmt.Errorf("error dereferencing dataset query: %s", err.Error())
	}

	ref := &repo.DatasetRef{Name: p.SaveName, Path: dspath, Dataset: ds}

	if err := r.repo.LogQuery(ref); err != nil {
		return fmt.Errorf("error logging query to repo: %s", err.Error())
	}

	*res = *ref
	return nil
}
