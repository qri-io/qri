package qds

import (
	"bytes"
	"context"
	"testing"

	"github.com/cube2222/octosql/app"
	octocfg "github.com/cube2222/octosql/config"
	csvoutput "github.com/cube2222/octosql/output/csv"
	"github.com/cube2222/octosql/parser"
	"github.com/cube2222/octosql/parser/sqlparser"
	"github.com/cube2222/octosql/physical"
	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestQriDatasource(t *testing.T) {

	tr, cleanup := newTestRunner(t)
	defer cleanup()

	cfg := &octocfg.Config{
		DataSources: []octocfg.DataSourceConfig{
			{Type: CfgTypeString, Name: "me_movies",
				Config: map[string]interface{}{
					"ref": "me/movies",
				},
			},
		},
	}

	res := tr.MustRun(t, "select t1.title from me_movies t1 limit 1", cfg)

	expect := "t1.title\n'Avatar '\n"
	if diff := cmp.Diff(expect, res); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}

type testRunner struct {
	ctx  context.Context
	repo repo.Repo
}

func newTestRunner(t *testing.T) (*testRunner, func()) {
	r, err := repotest.NewTestRepo()
	if err != nil {
		t.Fatal(err)
	}

	tr := &testRunner{
		ctx:  context.Background(),
		repo: r,
	}

	cleanup := func() {

	}

	return tr, cleanup
}

func (tr *testRunner) MustRun(t *testing.T, query string, cfg *octocfg.Config) string {
	fac := NewDataSourceBuilderFactory(tr.repo)
	ff := func(dbConfig map[string]interface{}) (physical.DataSourceBuilderFactory, error) {
		return fac, nil
	}

	dataSourceRespository, err := physical.CreateDataSourceRepositoryFromConfig(
		map[string]physical.Factory{
			"qri": ff,
		},
		cfg,
	)

	if err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}

	app := app.NewApp(cfg, dataSourceRespository, csvoutput.NewOutput(',', out), false)

	// Parse query
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		t.Fatalf("couldn't parse query: %s", err)
	}
	typed, ok := stmt.(sqlparser.SelectStatement)
	if !ok {
		t.Fatalf("statement must be a select statement")
	}

	plan, err := parser.ParseNode(typed)
	if err != nil {
		t.Fatal("couldn't parse query: ", err)
	}

	// Run query
	app.RunPlan(tr.ctx, plan)
	return out.String()
}
