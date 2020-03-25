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
	qrierr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/sql/preprocess"
	"github.com/qri-io/qri/sql/qds"
)

var log = golog.Logger("sql")

// Service executes SQL queries against qri datasets
type Service struct {
	r repo.Repo
}

// New creates an SQL service
func New(r repo.Repo) *Service {
	return &Service{
		r: r,
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
		return qds.NewDataSourceBuilderFactory(svc.r), nil
	}

	dataSourceRespository, err := physical.CreateDataSourceRepositoryFromConfig(
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

	app := app.NewApp(cfg, dataSourceRespository, out, false)

	// Parse query
	stmt, err := sqlparser.Parse(processedQuery)
	if err != nil {
		log.Errorf("couldn't parse query: %s", err)
		return fmt.Errorf("couldn't parse query: %s", err)
	}
	typed, ok := stmt.(sqlparser.SelectStatement)
	if !ok {
		err := fmt.Errorf("invalid statement type, wanted sqlparser.SelectStatement got %v", reflect.TypeOf(stmt))
		log.Error(err)
		return err
	}
	plan, err := parser.ParseNode(typed)
	if err != nil {
		log.Fatal("couldn't parse query: ", err)
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
