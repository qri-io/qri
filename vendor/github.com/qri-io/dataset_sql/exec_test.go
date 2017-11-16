package dataset_sql

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
	"github.com/qri-io/dataset/dsfs"
	dmp "github.com/sergi/go-diff/diffmatchpatch"
)

func TestSelectFields(t *testing.T) {
	store, resources, err := makeTestStore()
	if err != nil {
		t.Errorf("error creating test data: %s", err.Error())
		return
	}

	t1f := resources["t1"].Structure.Schema.Fields
	created, title, views, rating, notes := t1f[0], t1f[1], t1f[2], t1f[3], t1f[4]

	cases := []execTestCase{
		{"select * from t1", nil, []*dataset.Field{created, title, views, rating, notes}, "ratings/t1.csv"},
		{"select created, title, views, rating, notes from t1", nil, []*dataset.Field{created, title, views, rating, notes}, "ratings/t1.csv"},
		{"select created from t1", nil, []*dataset.Field{created}, "ratings/t1_created.csv"},
		{"select t1.created, t1.title, t1.views, t1.rating, t1.notes from t1 limit 1 offset 1", nil, []*dataset.Field{created, title, views, rating, notes}, "ratings/t1_row_2.csv"},
		{"select created, t1.title, t1.views, rating, notes from t1 where title = 'title_2'", nil, []*dataset.Field{created, title, views, rating, notes}, "ratings/t1_row_2.csv"},

		{"select * from t2 where title = 'test_title' order by title", nil, []*dataset.Field{created, title, views, rating, notes}, ""},
		{"select * from t2 where title = 'test_title_two'", nil, []*dataset.Field{created, title, views, rating, notes}, ""},

		{"select * from t2 order by rating", nil, []*dataset.Field{created, title, views, rating, notes}, "ratings/t2_order_rating.csv"},
		{"select sum(views), avg(views), count(views), max(views), min(views) from t1", nil, []*dataset.Field{
			&dataset.Field{Name: "sum", Type: datatypes.Float},
			&dataset.Field{Name: "avg", Type: datatypes.Float},
			&dataset.Field{Name: "count", Type: datatypes.Float},
			&dataset.Field{Name: "max", Type: datatypes.Float},
			&dataset.Field{Name: "min", Type: datatypes.Float},
		}, "ratings/t1_agg.csv"},

		{"select * from t3 order by rating", nil, []*dataset.Field{created, title, views, rating, notes}, "ratings/t3_order_rating.csv"},
		{"select sum(views), avg(views), count(views), max(views), min(views) from t3", nil, []*dataset.Field{
			&dataset.Field{Name: "sum", Type: datatypes.Float},
			&dataset.Field{Name: "avg", Type: datatypes.Float},
			&dataset.Field{Name: "count", Type: datatypes.Float},
			&dataset.Field{Name: "max", Type: datatypes.Float},
			&dataset.Field{Name: "min", Type: datatypes.Float},
		}, "ratings/t3_agg.csv"},

		{"select * from t3 where views > 5", nil, []*dataset.Field{created, title, views, rating, notes}, "empty.csv"},
		{"select * from t3 where views < 3", nil, []*dataset.Field{created, title, views, rating, notes}, "ratings/t3_views_less_than_3.csv"},
	}

	runCases(store, resources, cases, t)
}

func TestSelectJoin(t *testing.T) {
	store, resources, err := makeTestStore()
	if err != nil {
		t.Errorf("error creating test data: %s", err.Error())
		return
	}

	t1f := resources["t1"].Structure.Schema.Fields
	created, title, views, rating, notes := t1f[0], t1f[1], t1f[2], t1f[3], t1f[4]

	cases := []execTestCase{
		{"select * from t1, t2 where t1.title = t2.title order by t1.views desc", nil, []*dataset.Field{created, title, views, rating, notes, created, title, views, rating, notes}, "ratings/t1_t2_join.csv"},
		{`SELECT t1.views as v, t2.notes as n 
			FROM t1 LEFT JOIN t2 
			ON t1.title = t2.title`, nil, []*dataset.Field{&dataset.Field{Name: "v", Type: datatypes.Integer}, &dataset.Field{Name: "n", Type: datatypes.String}}, ""},
		// {"select * from t1, t2 where t1.notes = t2.notes", nil, []*dataset.Field{created, title, views, rating, notes, created, title, views, rating, notes}, 1, ""},
		// {"select t1.title, t2.title from t1, t2 where t1.notes = t2.notes", nil, []*dataset.Field{title, title}, 1, ""},

		// TODO - need to check result structure name on this one:
		// {"select * from a as aa, b as bb where a.created = b.created", nil, []*dataset.Field{created, title, views, rating, notes, created, title, views, rating, notes}, 2, ""},
		// {"select 1 from a", nil, []*dataset.Field{&dataset.Field{Name: "result", Type: datatypes.Integer}}, 1, ""},
	}

	runCases(store, resources, cases, t)
}

type execTestCase struct {
	statement  string
	expect     error
	fields     []*dataset.Field
	resultpath string
}

func runCases(store cafs.Filestore, ns map[string]*dataset.Dataset, cases []execTestCase, t *testing.T) {
	for i, c := range cases {

		q := &dataset.Query{
			Syntax: "sql",
			Abstract: &dataset.AbstractQuery{
				Statement: c.statement,
			},
			Resources: ns,
		}

		results, data, err := Exec(store, q, func(o *ExecOpt) {
			o.Format = dataset.CsvDataFormat
		})
		if err != c.expect {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.expect, err.Error())
			continue
		}

		if len(results.Schema.Fields) != len(c.fields) {
			t.Errorf("case %d field length mismatch. expected: %d, got: %d", i, len(c.fields), len(results.Schema.Fields))
			continue
		}

		for j, f := range c.fields {
			if results.Schema.Fields[j].Name != f.Name {
				t.Errorf("case %d field %d name mismatch. expected: %s, got: %s", i, j, f.Name, results.Schema.Fields[j].Name)
				break
			}
			if results.Schema.Fields[j].Type != f.Type {
				t.Errorf("case %d field %d type mismatch. expected: %s, got: %s", i, j, f.Type, results.Schema.Fields[j].Type)
				break
			}
		}

		if c.resultpath != "" {
			expect, err := loadTestdata(c.resultpath)
			if err != nil {
				t.Errorf("case %d error loading result data: %s", i, err.Error())
			}

			if !bytes.Equal(expect, data) {
				dmp := dmp.New()
				diffs := dmp.DiffMain(string(expect), string(data), true)
				if len(diffs) == 0 {
					t.Logf("case %d bytes were unequal but computed no difference between results")
					continue
				}

				t.Errorf("case %d mismatch:\n%s", i, dmp.DiffPrettyText(diffs))
				if len(expect) < 50 {
					t.Errorf("expected: %s, got: %s", string(expect), string(data))
				}
			}

		}
	}
}

func makeTestStore() (store cafs.Filestore, datasets map[string]*dataset.Dataset, err error) {
	store = memfs.NewMapstore()
	datasets = map[string]*dataset.Dataset{}
	testData := []struct {
		name, dspath, datapath string
	}{
		{"t1", "ratings/dataset.json", "ratings/t1.csv"},
		{"t2", "ratings/dataset.json", "ratings/t2.csv"},
		{"t3", "ratings/dataset.json", "ratings/t3.csv"},
	}

	for _, td := range testData {
		var (
			ds   *dataset.Dataset
			data []byte
		)

		ds, err = loadTestDataset(td.dspath)
		if err != nil {
			return
		}
		data, err = loadTestdata(td.datapath)
		if err != nil {
			return
		}

		datapath, err := store.Put(memfs.NewMemfileBytes(td.datapath, data), true)
		if err != nil {
			return nil, nil, err
		}
		ds.Data = datapath
		dspath, err := dsfs.SaveDataset(store, ds, true)
		if err != nil {
			return nil, nil, err
		}

		datasets[td.name], err = dsfs.LoadDataset(store, dspath)
		if err != nil {
			return nil, nil, err
		}
	}

	return store, datasets, nil
}

func loadTestdata(path string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join("testdata", path))
}

func loadTestDataset(path string) (*dataset.Dataset, error) {
	dsdata, err := loadTestdata(path)
	if err != nil {
		return nil, err
	}
	ds := &dataset.Dataset{}
	if err = ds.UnmarshalJSON(dsdata); err != nil {
		return nil, err
	}
	return ds, nil
}
