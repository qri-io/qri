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

	// TODO - clean this up, this is a hack to prevent null datasets from
	// being sent back as responses.
	// Warning - this could throw off pagination :/
	final := []*repo.DatasetRef{}
	for _, ref := range results {
		if ref.Dataset != nil {
			final = append(final, ref)
		}
	}
	*res = final
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
		transform *dataset.Transform
		results   []byte
		err       error
		ds        = p.Dataset
	)

	if ds == nil {
		return fmt.Errorf("dataset is required")
	}
	// fmt.Println("running query: %s", p.Dataset.QueryString)

	ds.Timestamp = time.Now()

	q := ds.Transform
	if q == nil {
		q = &dataset.Transform{
			Syntax: "sql",
			Data:   ds.QueryString,
		}
	}

	// if ds.QueryString == "" {
	// 	ds.QueryString = q.Abstract.Statement
	// }

	// TODO - make format output the parsed statement as well
	// to avoid triple-parsing
	// sqlstr, _, remap, err := sql.Format(ds.QueryString)
	// if err != nil {
	// 	return err
	// }
	names, err := sql.StatementTableNames(q.Data)
	if err != nil {
		return fmt.Errorf("error getting statement table names: %s", err.Error())
	}

	if q.Resources == nil {
		q.Resources = map[string]*dataset.Dataset{}
		// collect table references
		for _, name := range names {
			path, err := r.repo.GetPath(name)
			if err != nil {
				return fmt.Errorf("error getting path to dataset %s: %s", name, err.Error())
			}
			d, err := dsfs.LoadDataset(store, path)
			if err != nil {
				return fmt.Errorf("error loading dataset: %s", err.Error())
			}
			q.Resources[name] = d
		}
	}

	// 	func PreparedQueryPath(fs cafs.Filestore, q *dataset.Query, opts *ExecOpt) (datastore.Key, error) {
	// 	q2 := &dataset.Query{}
	// 	q2.Assign(q)
	// 	prep, err := Prepare(q2, opts)
	// 	if err != nil {
	// 		return datastore.NewKey(""), err
	// 	}
	// 	return dsfs.SaveQuery(fs, prep.q, false)
	// }

	q2 := &dataset.Transform{}
	q2.Assign(q)
	_, abst, err := sql.Format(q, func(o *sql.ExecOpt) {
		o.Format = dataset.CSVDataFormat
	})
	if err != nil {
		return fmt.Errorf("formatting error: %s", err.Error())
	}
	qpath, err := dsfs.SaveTransform(store, abst, false)
	if err != nil {
		return fmt.Errorf("error calculating query hash: %s", err.Error())
	}

	// fmt.Println(qpath.String())
	// atb, _ := abst.MarshalJSON()
	// fmt.Println(string(atb))

	// qpath, err := sql.PreparedQueryPath(r.repo.Store(), q, &sql.ExecOpt{Format: dataset.CSVDataFormat})
	// if err != nil {
	// 	return fmt.Errorf("error calculating query hash: %s", err.Error())
	// }

	if dsp, err := repo.DatasetForQuery(r.repo, qpath); err != nil && err != repo.ErrNotFound {
		return fmt.Errorf("error checking for existing query: %s", err.Error())
	} else if err != repo.ErrNotFound {
		if ds, err := dsfs.LoadDataset(store, dsp); err == nil {
			ref := &repo.DatasetRef{Name: p.SaveName, Path: dsp, Dataset: ds}
			if err := r.repo.LogQuery(ref); err != nil {
				return fmt.Errorf("error logging query to repo: %s", err.Error())
			}
			*res = *ref
			return nil
		}
	}

	// TODO - detect data format from passed-in results structure
	transform, results, err = sql.Exec(store, q, func(o *sql.ExecOpt) {
		o.Format = dataset.CSVDataFormat
	})
	if err != nil {
		return fmt.Errorf("error executing query: %s", err.Error())
	}

	// tb, _ := transform.MarshalJSON()
	// fmt.Println(string(tb))

	// TODO - move this into setting on the dataset outparam
	ds.Structure = transform.Structure
	ds.Length = len(results)
	ds.Transform = q
	ds.AbstractTransform = transform

	datakey, err := store.Put(memfs.NewMemfileBytes("data."+ds.Structure.Format.String(), results), false)
	if err != nil {
		return fmt.Errorf("error putting results in store: %s", err.Error())
	}
	ds.Data = datakey.String()

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
	// fmt.Println("result query:", ds.AbstractTransform.Path())
	if err := dsfs.DerefDatasetTransform(store, ds); err != nil {
		return fmt.Errorf("error dereferencing dataset query: %s", err.Error())
	}
	fmt.Println(ds.AbstractTransform.Path().String())

	ref := &repo.DatasetRef{Name: p.SaveName, Path: dspath, Dataset: ds}

	if err := r.repo.LogQuery(ref); err != nil {
		return fmt.Errorf("error logging query to repo: %s", err.Error())
	}

	*res = *ref
	return nil
}

type DatasetQueriesParams struct {
	Path    string
	Orderby string
	Limit   int
	Offset  int
}

func (r *QueryRequests) DatasetQueries(p *DatasetQueriesParams, res *[]*repo.DatasetRef) error {
	if p.Path == "" {
		return fmt.Errorf("path is required")
	}

	store := r.repo.Store()
	_, err := dsfs.LoadDataset(store, datastore.NewKey(p.Path))
	if err != nil {
		return fmt.Errorf("error loading dataset: %s", err.Error())
	}

	nodes, err := r.repo.Graph()
	if err != nil {
		return fmt.Errorf("error loading graph: %s", err.Error())
	}

	dsq := repo.DatasetQueries(nodes)
	list := []*repo.DatasetRef{}

	for dshash, qKey := range dsq {
		if dshash == p.Path {
			ds, err := dsfs.LoadDataset(store, datastore.NewKey(dshash))
			if err != nil {
				return fmt.Errorf("error loading dataset: %s", err.Error())
			}

			list = append(list, &repo.DatasetRef{
				Path:    qKey,
				Dataset: ds,
			})
		}
	}

	*res = list
	return nil
}
