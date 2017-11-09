package dataset_sql

import (
	"fmt"
	"github.com/qri-io/dataset"
)

// ResultRowGenerator makes rows from SourceRows
// calling eval on a set of select expressions from a given
// SourceRow
type ResultRowGenerator struct {
	exprs SelectExprs
	aggs  []AggFunc
	st    *dataset.Structure
}

func NewResultRowGenerator(sel *Select, result *dataset.Structure) (rg *ResultRowGenerator, err error) {
	rg = &ResultRowGenerator{
		exprs: sel.SelectExprs,
		st:    result,
	}

	rg.aggs, err = AggregateFuncs(sel)
	if err != nil {
		return nil, err
	}
	return
}

var (
	ErrAggStmt   = fmt.Errorf("this statement only generates an aggregate result row")
	ErrTableStmt = fmt.Errorf("this statement doesn't generate an aggregate result row")
)

// GenerateRow generates a row
func (rg *ResultRowGenerator) GenerateRow() ([][]byte, error) {
	row := make([][]byte, len(rg.exprs))
	for i, expr := range rg.exprs {
		_, data, err := expr.Eval()
		if err != nil {
			return nil, err
		}
		row[i] = data
	}

	if !rg.HasAggregates() {
		return row, nil
	}
	return nil, ErrAggStmt
}

func (rg *ResultRowGenerator) HasAggregates() bool {
	return len(rg.aggs) > 0
}

func (rg *ResultRowGenerator) GenerateAggregateRow() ([][]byte, error) {
	if rg.HasAggregates() {
		// TODO - this is currently relying on order of returned results
		// matching the resulting schema, which is a horrible idea.
		row := make([][]byte, len(rg.aggs))
		for i, agg := range rg.aggs {
			row[i] = agg.Value()
		}
		return row, nil
	}
	return nil, ErrTableStmt
}

func (rg *ResultRowGenerator) Structure() *dataset.Structure {
	return rg.st
}
