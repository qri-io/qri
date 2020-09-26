package qds

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/cube2222/octosql/app"
	octocfg "github.com/cube2222/octosql/config"
	csvoutput "github.com/cube2222/octosql/output/csv"
	"github.com/cube2222/octosql/parser"
	"github.com/cube2222/octosql/parser/sqlparser"
	"github.com/cube2222/octosql/physical"
	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/dsref"
	qerr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestQriDatasourceSelect(t *testing.T) {

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

func TestQriDatasourceJoin(t *testing.T) {

	tr, cleanup := newTestRunner(t)
	defer cleanup()

	cfg := &octocfg.Config{
		DataSources: []octocfg.DataSourceConfig{
			{Type: CfgTypeString, Name: "me_movies",
				Config: map[string]interface{}{
					"ref": "me/movies",
				},
			},
			{Type: CfgTypeString, Name: "me_movies_directors",
				Config: map[string]interface{}{
					"ref": "me/movies_directors",
				},
			},
		},
	}

	res := tr.MustRun(t, "select t1.director, t2.duration from me_movies_directors t1 inner join me_movies t2 on t1.title = t2.title", cfg)

	expect := "t1.director,t2.duration\n'James Cameron',178\n'Christopher Nolan',164\n"
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
	fac := NewDataSourceBuilderFactory(tr.repo, tr.loadDatasetFunc())
	ff := func(dbConfig map[string]interface{}) (physical.DataSourceBuilderFactory, error) {
		return fac, nil
	}

	dataSourceRepository, err := physical.CreateDataSourceRepositoryFromConfig(
		map[string]physical.Factory{
			"qri": ff,
		},
		cfg,
	)

	if err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}

	app := app.NewApp(cfg, dataSourceRepository, csvoutput.NewOutput(',', out), false)

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

func (tr *testRunner) loadDatasetFunc() dsref.ParseResolveLoad {
	pro, _ := tr.repo.Profile()
	loader := base.NewLocalDatasetLoader(tr.repo)
	return newParseResolveLoadFunc(pro.Peername, tr.repo, loader)
}

// newParseResolveLoadFunc composes a username, resolver, and loader into a
// higher-order function that converts strings to full datasets
// pass the empty string as a username to disable the "me" keyword in references
func newParseResolveLoadFunc(username string, resolver dsref.Resolver, loader dsref.Loader) dsref.ParseResolveLoad {
	return func(ctx context.Context, refStr string) (*dataset.Dataset, error) {
		ref, err := dsref.Parse(refStr)
		if err != nil {
			return nil, err
		}

		if username == "" && ref.Username == "me" {
			msg := fmt.Sprintf(`Can't use the "me" keyword to refer to a dataset in this context.
Replace "me" with your username for the reference:
%s`, refStr)
			return nil, qerr.New(fmt.Errorf("invalid contextual reference"), msg)
		} else if username != "" && ref.Username == "me" {
			ref.Username = username
		}

		source, err := resolver.ResolveRef(ctx, &ref)
		if err != nil {
			return nil, err
		}

		return loader.LoadDataset(ctx, ref, source)
	}
}
