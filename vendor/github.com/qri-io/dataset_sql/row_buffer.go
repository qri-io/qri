package dataset_sql

import (
	"bytes"
	"fmt"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
	"github.com/qri-io/dataset/dsio"
	"sort"
)

// NewResultBuffer returns either a *RowBuffer or *dsio.Buffer depending on
// which is required. RowBuffer is (much) more expensive but supports introspection
// into already-written rows
func NewResultBuffer(stmt Statement, st *dataset.Structure) (dsio.RowReadWriter, error) {
	if needsRowBuffer(stmt) {
		return NewRowBuffer(stmt, st)
	}
	return dsio.NewBuffer(st)
}

// Checks to see if we need a RowBuffer at all. Statements that don't contain
// DISTINCT or ORDER BY clauses don't require row buffering
func needsRowBuffer(stmt Statement) bool {
	sel, ok := stmt.(*Select)
	if !ok {
		// TODO - remove this.
		// for now anything that isn't a select statement is a candidate for
		// row buffering
		return true
	}

	return len(sel.OrderBy) > 0 || sel.Distinct != ""
}

// RowBuffer keeps raw row data for ORDER BY & DISTINCT statements
type RowBuffer struct {
	rows [][][]byte
	less *func(i, j int) bool
	buf  *dsio.Buffer
	err  error
}

// NewRowBuffer allocates a RowBuffer from a statement
func NewRowBuffer(stmt Statement, st *dataset.Structure) (*RowBuffer, error) {
	buf, err := dsio.NewBuffer(st)
	if err != nil {
		return nil, err
	}
	rb := &RowBuffer{
		buf: buf,
	}
	rb.less, rb.err = makeLessFunc(rb, stmt, st)

	return rb, nil
}

func (rb *RowBuffer) Structure() dataset.Structure {
	return rb.buf.Structure()
}

func (rb *RowBuffer) ReadRow() ([][]byte, error) {
	return nil, fmt.Errorf("cannot read rows from a *RowBuffer")
}

func (rb *RowBuffer) WriteRow(row [][]byte) error {
	rb.rows = append(rb.rows, row)
	return nil
}

func (rb *RowBuffer) Close() error {
	if rb.err != nil {
		return rb.err
	}
	if rb.less == nil {

	}
	sort.Sort(rb)
	return nil
}

func (rb *RowBuffer) Bytes() []byte {
	for _, row := range rb.rows {
		if err := rb.buf.WriteRow(row); err != nil {
			return nil
		}
	}
	if err := rb.buf.Close(); err != nil {
		return nil
	}
	return rb.buf.Bytes()
}

func (rb *RowBuffer) HasRow(row [][]byte) bool {
ROWS:
	for _, r := range rb.rows {
		if len(r) != len(row) {
			return false
		}
		for i, cell := range row {
			if !bytes.Equal(r[i], cell) {
				continue ROWS
			}
		}
		return true
	}
	return false
}

// Len is the number of elements in the collection.
func (rb *RowBuffer) Len() int {
	return len(rb.rows)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (rb *RowBuffer) Less(i, j int) bool {
	less := *rb.less
	return less(i, j)
}

// Swap swaps the elements with indexes i and j.
func (rb *RowBuffer) Swap(i, j int) {
	rb.rows[i], rb.rows[j] = rb.rows[j], rb.rows[i]
}

func makeLessFunc(rb *RowBuffer, stmt Statement, st *dataset.Structure) (*func(i, j int) bool, error) {
	sel, ok := stmt.(*Select)
	if !ok {
		// TODO - need to implement this for all types of statements
		// need to add SelectExprs() SelectExprs and Orders() Orders
		// on Statement interface
		return nil, NotYetImplemented("non-select row ordering")
	}

	type order struct {
		idx  int
		desc bool
		dt   datatypes.Type
	}

	orderby := sel.OrderBy
	orders := []order{}
	if len(orderby) > 0 {
		for _, o := range orderby {
			// TODO - horrible hack, will break when sorting on multiple tables, or with non-abstract
			// statements.
			str := String(o.Expr)
			str = string(bytes.TrimPrefix([]byte(str), []byte("t1.")))
			idx := st.StringFieldIndex(str)
			if idx < 0 {
				return nil, fmt.Errorf("couldn't find sort index: %s", String(o.Expr))
			}
			orders = append(orders, order{
				idx:  idx,
				desc: o.Direction == "desc",
				dt:   st.Schema.Fields[idx].Type,
			})
		}
	}

	less := func(i, j int) bool {
		for _, o := range orders {
			l, err := datatypes.CompareTypeBytes(rb.rows[i][o.idx], rb.rows[j][o.idx], o.dt)
			if err != nil {
				continue
			}
			if (o.desc && l < 0) || l > 0 {
				continue
			}
			return true
		}
		return false
	}

	return &less, nil
}
