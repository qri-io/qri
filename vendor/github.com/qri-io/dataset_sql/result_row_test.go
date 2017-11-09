package dataset_sql

import (
	"github.com/qri-io/dataset"
	"testing"
)

func TestResultRowGenerator(t *testing.T) {
	_, datasets, err := makeTestStore()
	if err != nil {
		t.Errorf("error making test data: %s", err.Error())
		return
	}

	resources := map[string]*dataset.Structure{
		"t1": datasets["t1"].Structure,
	}

	stmt, err := Parse("select * from t1")
	if err != nil {
		t.Errorf("error parsing statement", err.Error())
		return
	}

	if err := PrepareStatement(stmt, resources); err != nil {
		t.Errorf("error remapping statement: %s", err.Error())
		return
	}
	cols := CollectColNames(stmt)

	rg, err := NewResultRowGenerator(stmt.(*Select), &dataset.Structure{})
	if err != nil {
		t.Errorf("error creating row generator: %s", err.Error())
		return
	}

	sr := SourceRow{
		"t1": [][]byte{[]byte("Sun Dec 25 09:25:46 2016"), []byte("test_title"), []byte("68882"), []byte("0.6893978118896484"), []byte("no notes")},
	}

	if err := SetSourceRow(cols, sr); err != nil {
		t.Errorf("error setting source row: %s", err.Error())
		return
	}

	row, err := rg.GenerateRow()
	if err != nil {
		t.Errorf("error generating row: %s", err.Error())
		return
	}

	if len(row) != 5 {
		t.Errorf("result row length mismatch. expected: %d, got: %d", 5, len(row))
	}

	// for _, c := range row {
	// 	fmt.Println(string(c))
	// }
}
