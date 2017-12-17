package core

import (
	"fmt"
	"net/rpc"
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
	cli  *rpc.Client
}

func (d QueryRequests) CoreRequestsName() string { return "queries" }

func NewQueryRequests(r repo.Repo, cli *rpc.Client) *QueryRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewQueryRequests"))
	}

	return &QueryRequests{
		repo: r,
		cli:  cli,
	}
}

func (d *QueryRequests) List(p *ListParams, res *[]*repo.DatasetRef) error {
	if d.cli != nil {
		return d.cli.Call("QueryRequests.List", p, res)
	}

	items, err := d.repo.ListQueryLogs(p.Limit, p.Offset)
	if err != nil {
		return fmt.Errorf("error getting query logs: %s", err.Error())
	}

	results := make([]*repo.DatasetRef, len(items))
	for i, item := range items {
		results[i] = &repo.DatasetRef{Path: item.DatasetPath}
		if ds, err := dsfs.LoadDataset(d.repo.Store(), item.DatasetPath); err == nil {
			results[i].Name = ds.Transform.Data
			results[i].Dataset = ds
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
	if d.cli != nil {
		return d.cli.Call("QueryRequests.Get", p, res)
	}

	// TODO - huh? do we even need to load query datasets?
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
	if r.cli != nil {
		return r.cli.Call("QueryRequests.Run", p, res)
	}

	var (
		store   = r.repo.Store()
		abst    *dataset.Transform
		results []byte
		err     error
		ds      = p.Dataset
	)

	if ds == nil {
		return fmt.Errorf("dataset is required")
	}

	ds.Timestamp = time.Now()

	q := ds.Transform
	if q == nil {
		q = &dataset.Transform{
			Syntax: "sql",
			Data:   ds.QueryString,
		}
	}

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

	q2 := &dataset.Transform{}
	q2.Assign(q)
	qrpath, err := sql.QueryRecordPath(store, q2, func(o *sql.ExecOpt) {
		o.Format = dataset.CSVDataFormat
	})
	if err != nil {
		return fmt.Errorf("error calculating query hash: %s", err.Error())
	}

	// TODO - currently broken. Fix & Add Tests
	// if qi, err := r.repo.QueryLogItem(&repo.QueryLogItem{Key: qrpath}); err != nil && err != repo.ErrNotFound {
	// 	return fmt.Errorf("error checking for existing query: %s", err.Error())
	// } else if err != repo.ErrNotFound {
	// 	if ds, err := dsfs.LoadDataset(store, qi.DatasetPath); err == nil {
	// 		// ref := &repo.QueryLogItem{Name: p.SaveName, Query: q.Data, Key: dsp, Dataset: dsp}
	// 		// if err := r.repo.LogQuery(ref); err != nil {
	// 		// 	return fmt.Errorf("error logging query to repo: %s", err.Error())
	// 		// }
	// 		*res = repo.DatasetRef{
	// 			Path:    qi.DatasetPath,
	// 			Dataset: ds,
	// 		}
	// 		return nil
	// 	}
	// }

	// TODO - detect data format from passed-in results structure
	abst, results, err = sql.Exec(store, q, func(o *sql.ExecOpt) {
		o.Format = dataset.CSVDataFormat
	})
	if err != nil {
		return fmt.Errorf("error executing query: %s", err.Error())
	}

	// TODO - move this into setting on the dataset outparam
	ds.Structure = q.Structure
	ds.Length = len(results)
	ds.Transform = q
	ds.AbstractTransform = abst

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
	if err := dsfs.DerefDatasetTransform(store, ds); err != nil {
		return fmt.Errorf("error dereferencing dataset query: %s", err.Error())
	}

	ref := &repo.DatasetRef{Name: p.SaveName, Path: dspath, Dataset: ds}
	item := &repo.QueryLogItem{
		Query:       ds.QueryString,
		Name:        p.SaveName,
		Key:         qrpath,
		DatasetPath: dspath,
		Time:        time.Now(),
	}
	if err := r.repo.LogQuery(item); err != nil {
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
	if r.cli != nil {
		return r.cli.Call("QueryRequests.DatasetQueries", p, res)
	}

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
