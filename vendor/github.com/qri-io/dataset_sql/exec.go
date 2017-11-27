package dataset_sql

import (
	"fmt"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
)

type ExecOpt struct {
	Format dataset.DataFormat
}

func opts(options ...func(*ExecOpt)) *ExecOpt {
	o := &ExecOpt{
		Format: dataset.CsvDataFormat,
	}
	for _, option := range options {
		option(o)
	}
	return o
}

func Exec(store cafs.Filestore, q *dataset.Query, options ...func(o *ExecOpt)) (result *dataset.Structure, resultBytes []byte, err error) {
	opts := &ExecOpt{
		Format: dataset.CsvDataFormat,
	}
	for _, option := range options {
		option(opts)
	}

	if q.Syntax != "sql" {
		return nil, nil, fmt.Errorf("Invalid syntax: '%s' sql_dataset only supports sql syntax. ", q.Syntax)
	}

	prep, err := Prepare(q, opts)
	if err != nil {
		return nil, nil, err
	}

	return prep.stmt.exec(store, prep)
}

// CollectColNames grabs a slice of pointers to all columns
// in a given SQL statement.
func CollectColNames(stmt Statement) (cols []*ColName) {
	stmt.WalkSubtree(func(node SQLNode) (bool, error) {
		if col, ok := node.(*ColName); ok && node != nil {
			cols = append(cols, col)
		}
		return true, nil
	})
	return
}

// SetSourceRow sets ColName values to the current SourceRow
// value for evaluation
func SetSourceRow(cols []*ColName, sr SourceRow) error {
	for _, col := range cols {
		if col.Metadata.TableName == "" {
			return fmt.Errorf("col missing metadata: %#v", col)
		}
		if col.Metadata.ColIndex > len(sr[col.Metadata.TableName])-1 {
			return fmt.Errorf("index out of range to set column value: %s.%d", col.Metadata.TableName, col.Metadata.ColIndex)
		}
		col.Value = sr[col.Metadata.TableName][col.Metadata.ColIndex]
	}
	return nil
}

func (stmt *Select) exec(store cafs.Filestore, prep preparedQuery) (result *dataset.Structure, resultBytes []byte, err error) {
	q := prep.q
	absq := q.Abstract
	cols := CollectColNames(stmt)
	buf, err := NewResultBuffer(stmt, absq.Structure)
	if err != nil {
		return result, nil, err
	}
	srg, err := NewSourceRowGenerator(store, prep.paths, absq.Structures)
	if err != nil {
		return result, nil, err
	}
	srf, err := NewSourceRowFilter(stmt, buf)
	if err != nil {
		return result, nil, err
	}
	rrg, err := NewResultRowGenerator(stmt, absq.Structure)
	if err != nil {
		return result, nil, err
	}

	for srg.Next() && !srf.Done() {
		sr, err := srg.Row()
		if err != nil {
			return result, nil, err
		}

		if err := SetSourceRow(cols, sr); err != nil {
			return result, nil, err
		}

		if srf.Match() {
			row, err := rrg.GenerateRow()
			if err == ErrAggStmt {
				continue
			} else if err != nil {
				return result, nil, err
			}

			if srf.ShouldWriteRow(row) {
				if err := buf.WriteRow(row); err != nil {
					return result, nil, err
				}
			}

		}
	}

	if rrg.HasAggregates() {
		row, err := rrg.GenerateAggregateRow()
		if err != nil {
			return result, nil, err
		}
		buf.WriteRow(row)
	}

	if err := buf.Close(); err != nil {
		return result, nil, err
	}

	// TODO - rename / deref result var
	result = prep.result
	resultBytes = buf.Bytes()
	return
}

func (node *Union) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("union statements")
}
func (node *Insert) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("insert statements")
}
func (node *Update) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("update statements")
}
func (node *Delete) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("delete statements")
}
func (node *Set) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("set statements")
}
func (node *DDL) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("ddl statements")
}
func (node *ParenSelect) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("ParenSelect statements")
}
func (node *Show) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("Show statements")
}
func (node *Use) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("Use statements")
}
func (node *OtherRead) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("OtherRead statements")
}
func (node *OtherAdmin) exec(store cafs.Filestore, prep preparedQuery) (*dataset.Structure, []byte, error) {
	return nil, nil, NotYetImplemented("OtherAdmin statements")
}
