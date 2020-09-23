package qds

import (
	"context"
	"errors"
	"fmt"

	"github.com/cube2222/octosql"
	"github.com/cube2222/octosql/config"
	"github.com/cube2222/octosql/execution"
	"github.com/cube2222/octosql/physical"
	"github.com/cube2222/octosql/physical/metadata"
	golog "github.com/ipfs/go-log"
	perrors "github.com/pkg/errors"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/tabular"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	qrierr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/repo"
)

// CfgTypeString is the string constant that indicates the qri data source as
// configuration
const CfgTypeString = "qri"

var log = golog.Logger("qds")

var availableFilters = map[physical.FieldType]map[physical.Relation]struct{}{
	physical.Primary:   make(map[physical.Relation]struct{}),
	physical.Secondary: make(map[physical.Relation]struct{}),
}

// DataSource implements a qri dataset as an octosql.DataSource
type DataSource struct {
	r     repo.Repo
	alias string
	ref   dsref.Ref
	ds    *dataset.Dataset
}

// NewDataSourceBuilderFactory is a factory function for qri data source
// builders
func NewDataSourceBuilderFactory(r repo.Repo, loadDataset dsref.ParseResolveLoad) physical.DataSourceBuilderFactory {
	return physical.NewDataSourceBuilderFactory(
		func(ctx context.Context, matCtx *physical.MaterializationContext, dbConfig map[string]interface{}, filter physical.Formula, alias string) (execution.Node, error) {
			refstr, err := config.GetString(dbConfig, "ref")
			if err != nil {
				return nil, perrors.Wrap(err, "couldn't get path")
			}

			ds, err := loadDataset(ctx, refstr)
			if err != nil {
				return nil, err
			}

			// TODO(b5) - we need an easy way to get a reference from a dataset
			ref := dsref.Ref{
				// TODO(b5) - get an ID field on dataset
				// InitID: ds.ID,
				Username: ds.Peername,
				Name:     ds.Name,
				Path:     ds.Path,
			}

			return &DataSource{
				r:     r,
				alias: alias,
				ref:   ref,
			}, nil
		},
		nil,
		availableFilters,
		metadata.BoundedFitsInLocalStorage,
	)
}

// Get implements octosql's execution.Node interface, returning a RecordStream
func (qds *DataSource) Get(ctx context.Context, variables octosql.Variables) (execution.RecordStream, error) {
	ref := qds.ref
	ds, err := dsfs.LoadDataset(ctx, qds.r.Store(), ref.Path)
	if err != nil {
		log.Debugf("buildSource: dsfs.LoadDataset '%s': %s", qds.ref, err)
		return nil, perrors.Wrap(err, "preparing SQL data source: couldn't load dataset")
	}

	if ds.Structure == nil {
		return nil, fmt.Errorf("dataset %s has no Structure component", qds.ref)
	}

	if ds.Structure.Format != dataset.CSVDataFormat.String() {
		return nil, errors.New("sql queries only support CSV-formatted data")
	}

	if err = base.OpenDataset(ctx, qds.r.Filesystem(), ds); err != nil {
		log.Debugf("buildSource: base.OpenDataset '%s': %s", qds.ref, err)
		return nil, perrors.Wrap(err, "couldn't open ")
	}

	r, err := dsio.NewEntryReader(ds.Structure, ds.BodyFile())
	if err != nil {
		return nil, err
	}

	aliasedFields, err := initializeColumns(qds.alias, qds.ref, ds.Structure)
	if err != nil {
		return nil, perrors.Wrap(err, "couldn't initialize columns for record stream")
	}

	return &RecordStream{
		alias:         qds.alias,
		ds:            ds,
		r:             r,
		isDone:        false,
		aliasedFields: aliasedFields,
	}, nil
}

// RecordStream connects a qri dataset to an octosql.RecordStream interface
type RecordStream struct {
	ds            *dataset.Dataset
	r             dsio.EntryReader
	isDone        bool
	alias         string
	aliasedFields []octosql.VariableName
}

// Close finalizes the stream
func (rs *RecordStream) Close() error {
	if err := rs.r.Close(); err != nil {
		return perrors.Wrap(err, "couldn't close dataset entry reader")
	}

	return nil
}

func initializeColumns(alias string, ref dsref.Ref, st *dataset.Structure) ([]octosql.VariableName, error) {
	cols, _, err := tabular.ColumnsFromJSONSchema(st.Schema)
	if err != nil {
		// the tabular package emits nice errors we can use as user-facing messages
		// so we wrap in a qri error
		err = fmt.Errorf("cannot use '%s' as sql table.\n%w", ref, err)
		return nil, qrierr.New(err, err.Error())
	}

	if err := cols.ValidMachineTitles(); err != nil {
		err = fmt.Errorf("cannot use '%s' as sql table.\n%w", ref, err)
		return nil, qrierr.New(err, err.Error())
	}

	titles := cols.Titles()

	fields := make([]octosql.VariableName, len(titles))
	for i, t := range titles {
		fields[i] = octosql.NewVariableName(fmt.Sprintf("%s.%s", alias, t))
	}

	return fields, nil
}

// Next reads the next execution record in a stream
func (rs *RecordStream) Next(ctx context.Context) (*execution.Record, error) {
	if rs.isDone {
		return nil, execution.ErrEndOfStream
	}

	ent, err := rs.r.ReadEntry()
	if err != nil {
		if err.Error() == "EOF" {
			rs.isDone = true
			rs.r.Close()
			return nil, execution.ErrEndOfStream
		}
		log.Debug(err)
		return nil, err
	}

	aliasedRecord := make(map[octosql.VariableName]octosql.Value)
	if rec, ok := ent.Value.([]interface{}); ok {
		for i, x := range rec[:len(rs.aliasedFields)] {
			switch v := x.(type) {
			case string:
				aliasedRecord[rs.aliasedFields[i]] = octosql.MakeString(v)
			case int:
				aliasedRecord[rs.aliasedFields[i]] = octosql.MakeInt(v)
			case int64:
				aliasedRecord[rs.aliasedFields[i]] = octosql.MakeInt(int(v))
			case float64:
				aliasedRecord[rs.aliasedFields[i]] = octosql.MakeFloat(v)
			case bool:
				aliasedRecord[rs.aliasedFields[i]] = octosql.MakeBool(v)
			default:
				aliasedRecord[rs.aliasedFields[i]] = octosql.MakeNull()
			}
		}
	} else {
		log.Debugf("returned record is not an array type. got: %q", ent)
		return nil, fmt.Errorf("returned record is not an array type. got: %q", ent)
	}

	return execution.NewRecord(rs.aliasedFields, aliasedRecord), nil
}
