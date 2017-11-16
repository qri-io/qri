package dataset_sql

import (
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
)

func TestPrepare(t *testing.T) {
	ds := &dataset.Query{
		Abstract: &dataset.AbstractQuery{
			Statement: "select * from t1",
		},
		Resources: map[string]*dataset.Dataset{
			"t1": &dataset.Dataset{
				Data: datastore.NewKey("t1/data/path"),
				Structure: &dataset.Structure{
					Format: dataset.CsvDataFormat,
					Schema: &dataset.Schema{
						Fields: []*dataset.Field{
							&dataset.Field{Name: "one", Type: datatypes.String},
							&dataset.Field{Name: "two", Type: datatypes.Boolean},
						},
					},
				},
			},
		},
	}

	prep, err := Prepare(ds, &ExecOpt{Format: dataset.CsvDataFormat})
	if err != nil {
		t.Errorf("unexpected error from Prepare: %s", err.Error())
		return
	}

	str := String(prep.stmt)
	expect := "select t1.a as a, t1.b as b from t1"
	if expect != str {
		t.Errorf("statement error, expected: '%s', got: '%s'", expect, str)
		return
	}

	if prep.paths["t1"].String() != "/t1/data/path" {
		t.Errorf("data path error, expected %s, got %s", "/t1/data/path", prep.paths["a"].String())
	}
}

func TestPrepareStatement(t *testing.T) {
	stmt, err := Parse("select * from t1")
	if err != nil {
		t.Errorf("error parsing statement: %s", err.Error())
		return
	}

	resources := map[string]*dataset.Structure{
		"t1": &dataset.Structure{
			Format: dataset.CsvDataFormat,
			Schema: &dataset.Schema{
				Fields: []*dataset.Field{
					&dataset.Field{Name: "a", Type: datatypes.Integer},
					&dataset.Field{Name: "b", Type: datatypes.Float},
				},
			},
		},
	}

	if err := PrepareStatement(stmt, resources); err != nil {
		t.Errorf("error remapping statement: %s", err.Error())
		return
	}

	sel := stmt.(*Select)
	if len(resources["t1"].Schema.Fields) != len(sel.SelectExprs) {
		t.Errorf("select expressions length mismatch. expected %d, got %d", len(resources["t1"].Schema.Fields), len(sel.SelectExprs))
	}
}
