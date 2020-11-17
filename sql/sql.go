// +build !arm

package sql

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/cube2222/octosql/app"
	octosqlcfg "github.com/cube2222/octosql/config"
	"github.com/cube2222/octosql/output"
	csvoutput "github.com/cube2222/octosql/output/csv"
	jsonoutput "github.com/cube2222/octosql/output/json"
	"github.com/cube2222/octosql/output/table"
	"github.com/cube2222/octosql/parser"
	"github.com/cube2222/octosql/parser/sqlparser"
	"github.com/cube2222/octosql/physical"
	golog "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/qri-io/qri/dsref"
	qrierr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/sql/preprocess"
	"github.com/qri-io/qri/sql/qds"
)

var log = golog.Logger("sql")

// Service executes SQL queries against qri datasets
type Service struct {
	r           repo.Repo
	loadDataset dsref.ParseResolveLoad
}

// New creates an SQL service
func New(r repo.Repo, loadDataset dsref.ParseResolveLoad) *Service {
	return &Service{
		r:           r,
		loadDataset: loadDataset,
	}
}

// Exec runs an SQL query against a given dataset mapping
func (svc *Service) Exec(ctx context.Context, w io.Writer, outFormat, query string) error {
	processedQuery, sources, err := preprocess.Query(query)
	if err != nil {
		log.Errorf("mapping query: %s", err)
		return err
	}

	// Configuration
	cfg := &octosqlcfg.Config{}
	for name, refStr := range sources {
		cfg.DataSources = append(cfg.DataSources, octosqlcfg.DataSourceConfig{
			Type: qds.CfgTypeString,
			Name: name,
			Config: map[string]interface{}{
				"ref": refStr,
			},
		})
	}

	ff := func(dbConfig map[string]interface{}) (physical.DataSourceBuilderFactory, error) {
		return qds.NewDataSourceBuilderFactory(svc.r, svc.loadDataset), nil
	}

	dataSourceRepository, err := physical.CreateDataSourceRepositoryFromConfig(
		map[string]physical.Factory{
			"qri": ff,
		},
		cfg,
	)
	if err != nil {
		log.Error(err)
		return err
	}

	var out output.Output
	switch outFormat {
	case "table":
		out = table.NewOutput(w, false)
	case "table_row_separated":
		out = table.NewOutput(w, true)
	case "json":
		out = jsonoutput.NewOutput(w)
	case "csv":
		out = csvoutput.NewOutput(',', w)
	case "tabbed":
		out = csvoutput.NewOutput('\t', w)
	default:
		err = fmt.Errorf("invalid output type: %s", w)
		log.Error(err)
		return err
	}

	app := app.NewApp(cfg, dataSourceRepository, out, false)

	// Parse query
	stmt, err := sqlparser.Parse(processedQuery)
	if err != nil {
		log.Debugf("couldn't parse query: %s", err)
		return qrierr.New(err, fmt.Sprintf("Parsing SQL:\n%s", err.Error()))
	}
	typed, ok := stmt.(sqlparser.SelectStatement)
	if !ok {
		log.Debugf("%v is not a select statement", reflect.TypeOf(stmt))
		err := fmt.Errorf("invalid statement type, wanted sqlparser.SelectStatement got %v", reflect.TypeOf(stmt))
		return qrierr.New(err, "only SELECT statements are supported")
	}
	plan, err := parser.ParseNode(typed)
	if err != nil {
		log.Debugf("couldn't generate plan: ", err)
		msg := `Qri was able to parse your SQL statement, but can't execute it. 
Some SQL functions and features are not yet implemented. 
Check our issue tracker for SQL support & feature requests:
  https://github.com/qri-io/qri/issues?q=is:issue+label:SQL

Error:
%s`
		return qrierr.New(err, fmt.Sprintf(msg, err.Error()))
	}

	// Run query
	err = app.RunPlan(ctx, plan)
	return unwrapErr(err)
}

// octosql uses the errors package, which doesn't support errors.Unwrap,
// so we unwrap before returning
func unwrapErr(err error) error {
	if err != nil {
		switch e := errors.Cause(err).(type) {
		case qrierr.Error:
			return qrierr.New(err, e.Message())
		default:
			return err
		}
	}
	return nil
}
