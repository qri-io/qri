package dataset_sql

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
	"testing"
)

func TestSourceRowGenerator(t *testing.T) {
	store, resources, err := makeTestStore()
	if err != nil {
		t.Errorf("error creating test data: %s", err.Error())
		return
	}

	datamap := map[string]datastore.Key{}
	st := map[string]*dataset.Structure{}
	for _, name := range []string{"t1", "t2"} {
		datamap[name] = resources[name].Data
		st[name] = resources[name].Structure
	}

	srg, err := NewSourceRowGenerator(store, datamap, st)
	if err != nil {
		t.Errorf("error creating generator: %s", err.Error())
		return
	}

	count := 0
	for srg.Next() {
		count++
		// TODO - check that rows are iterating the right values
		_, err := srg.Row()
		if err != nil {
			t.Errorf("row %d unexpected error: %s", count, err.Error())
			return
		}
	}

	if count != 100 {
		t.Errorf("wrong number of iterations. expected %d, got %d", 100, count)
	}
}

func TestSourceRowFilter(t *testing.T) {
	// TODO - need to test limit / offset clauses
	stmt, err := Parse("select * from t1 where t1.a > 5")
	if err != nil {
		t.Errorf("statement parse error: %s", err.Error())
		return
	}

	srg, err := NewSourceRowFilter(stmt, nil)
	if err != nil {
		t.Errorf("errog creating source row filter: %s", err.Error())
		return
	}

	resources := map[string]*dataset.Structure{
		"t1": &dataset.Structure{
			Format: dataset.CsvDataFormat,
			Schema: &dataset.Schema{
				Fields: []*dataset.Field{
					&dataset.Field{Name: "a", Type: datatypes.Integer},
				},
			},
		},
	}

	if err := PrepareStatement(stmt, resources); err != nil {
		t.Errorf("error preparing statement: %s", err.Error())
		return
	}
	cols := CollectColNames(stmt)

	cases := []struct {
		sr     SourceRow
		expect bool
	}{
		{SourceRow{"t1": [][]byte{[]byte("0")}}, false},
		{SourceRow{"t1": [][]byte{[]byte("5")}}, false},
		{SourceRow{"t1": [][]byte{[]byte("6")}}, true},
		{SourceRow{"t1": [][]byte{[]byte("10")}}, true},
		{SourceRow{"t1": [][]byte{[]byte("200")}}, true},
		{SourceRow{"t1": [][]byte{[]byte("7")}}, true},
	}

	for i, c := range cases {
		if err := SetSourceRow(cols, c.sr); err != nil {
			t.Errorf("case %d error setting source row: %s", i, err.Error())
			return
		}

		got := srg.Match()
		if got != c.expect {
			t.Errorf("case %d fail %t != %t", i, c.expect, got)
		}
	}
}
