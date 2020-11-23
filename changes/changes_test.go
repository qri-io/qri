package changes

import (
	"context"
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/tabular"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
	"github.com/qri-io/qri/stats"
)

func newTestService(t *testing.T, r repo.Repo, workDir string) *Service {
	cache, err := stats.NewLocalCache(workDir, 1000<<8)
	if err != nil {
		t.Fatal(err)
	}
	statsSvc := stats.New(cache)
	loader := base.NewLocalDatasetLoader(r.Filesystem())

	return New(loader, statsSvc)
}

func updateDataset(t *testing.T, r repo.Repo, ds *dataset.Dataset, newBody string) dsref.Ref {
	ctx := context.Background()
	currRef := dsref.ConvertDatasetToVersionInfo(ds).SimpleRef()

	ds.SetBodyFile(qfs.NewMemfileBytes("body.csv", []byte(newBody)))
	ds.PreviousPath = currRef.Path

	// force recalculate structure as that is what we rely on for the change reports
	ds.Structure = nil
	if err := base.InferStructure(ds); err != nil {
		t.Fatal(err.Error())
	}

	res, err := base.CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), ds, nil, dsfs.SaveSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err.Error())
	}
	return dsref.ConvertDatasetToVersionInfo(res).SimpleRef()
}

func getBaseCols() []*ChangeReportDeltaComponent {
	return []*ChangeReportDeltaComponent{
		&ChangeReportDeltaComponent{
			ChangeReportComponent: ChangeReportComponent{
				Left: EmptyObject{
					"count":  float64(5),
					"max":    float64(65.25),
					"min":    float64(44.4),
					"mean":   float64(52.04),
					"median": float64(50.65),
					"histogram": map[string]interface{}{
						"bins": []interface{}{
							float64(44.4),
							float64(50.65),
							float64(55.5),
							float64(65.25),
							float64(66.25),
						},
						"frequencies": []interface{}{
							float64(2),
							float64(1),
							float64(1),
							float64(1),
						},
					},
					"type": "numeric",
				},
				Right: EmptyObject{
					"count":  float64(5),
					"max":    float64(5000.65),
					"min":    float64(44),
					"mean":   float64(1238.06),
					"median": float64(440.4),
					"histogram": map[string]interface{}{
						"bins": []interface{}{
							float64(44),
							float64(55),
							float64(440.4),
							float64(650.25),
							float64(5000.65),
							float64(5001.65),
						},
						"frequencies": []interface{}{
							float64(1),
							float64(1),
							float64(1),
							float64(1),
							float64(1),
						},
					},
					"type": "numeric",
				},
				About: map[string]interface{}{
					"status": fsi.STChange,
				},
			},
			Title: "avg_age",
			Delta: map[string]interface{}{
				"count":  float64(0),
				"max":    float64(4935.4),
				"mean":   float64(1186.02),
				"median": float64(389.75),
				"min":    float64(-0.3999999999999986),
			},
		},
		&ChangeReportDeltaComponent{
			ChangeReportComponent: ChangeReportComponent{
				Left: EmptyObject{
					"count":     float64(5),
					"maxLength": float64(8),
					"minLength": float64(7),
					"unique":    float64(5),
					"frequencies": map[string]interface{}{
						"chatham":  float64(1),
						"chicago":  float64(1),
						"new york": float64(1),
						"raleigh":  float64(1),
						"toronto":  float64(1),
					},
					"type": "string",
				},
				Right: EmptyObject{
					"count":     float64(5),
					"maxLength": float64(8),
					"minLength": float64(7),
					"unique":    float64(5),
					"frequencies": map[string]interface{}{
						"chatham":  float64(1),
						"chicago":  float64(1),
						"new york": float64(1),
						"raleigh":  float64(1),
						"toronto":  float64(1),
					},
					"type": "string",
				},
				About: map[string]interface{}{
					"status": fsi.STUnmodified,
				},
			},
			Title: "city",
			Delta: map[string]interface{}{
				"count":     float64(0),
				"maxLength": float64(0),
				"minLength": float64(0),
				"unique":    float64(0),
			},
		},
		&ChangeReportDeltaComponent{
			ChangeReportComponent: ChangeReportComponent{
				Left: EmptyObject{
					"count":      float64(5),
					"falseCount": float64(1),
					"trueCount":  float64(4),
					"type":       "boolean",
				},
				Right: EmptyObject{
					"count":      float64(5),
					"falseCount": float64(5),
					"trueCount":  float64(0),
					"type":       "boolean",
				},
				About: map[string]interface{}{
					"status": fsi.STUnmodified,
				},
			},
			Title: "in_usa",
			Delta: map[string]interface{}{
				"count":      float64(0),
				"falseCount": float64(4),
				"trueCount":  float64(-4),
			},
		},
		&ChangeReportDeltaComponent{
			ChangeReportComponent: ChangeReportComponent{
				Left: EmptyObject{
					"count":  float64(5),
					"max":    float64(40000000),
					"min":    float64(35000),
					"mean":   float64(9817000),
					"median": float64(300000),
					"histogram": map[string]interface{}{
						"bins": []interface{}{
							float64(35000),
							float64(250000),
							float64(300000),
							float64(8500000),
							float64(40000000),
							float64(40000001),
						},
						"frequencies": []interface{}{
							float64(1),
							float64(1),
							float64(1),
							float64(1),
							float64(1),
						},
					},
					"type": "numeric",
				},
				Right: EmptyObject{
					"count":  float64(5),
					"max":    float64(4000000),
					"min":    float64(3500),
					"mean":   float64(981700),
					"median": float64(30000),
					"histogram": map[string]interface{}{
						"bins": []interface{}{
							float64(3500),
							float64(25000),
							float64(30000),
							float64(850000),
							float64(4000000),
							float64(4000001),
						},
						"frequencies": []interface{}{
							float64(1),
							float64(1),
							float64(1),
							float64(1),
							float64(1),
						},
					},
					"type": "numeric",
				},
				About: map[string]interface{}{
					"status": fsi.STChange,
				},
			},
			Title: "pop",
			Delta: map[string]interface{}{
				"count":  float64(0),
				"max":    float64(-36000000),
				"mean":   float64(-8835300),
				"median": float64(-270000),
				"min":    float64(-31500),
			},
		},
	}
}

func TestStatsDiff(t *testing.T) {
	ctx := context.Background()

	workDir, err := ioutil.TempDir("", "qri_test_changes_service")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)

	mr, err := repotest.NewTestRepo()
	if err != nil {
		t.Fatal(err)
	}

	svc := newTestService(t, mr, workDir)

	ref := dsref.MustParse("peer/cities")
	if _, err := mr.ResolveRef(ctx, &ref); err != nil {
		t.Fatal(err)
	}

	ds, err := dsfs.LoadDataset(ctx, mr.Filesystem(), ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err = base.OpenDataset(ctx, mr.Filesystem(), ds); err != nil {
		t.Fatal(err)
	}

	ds.Name = "cities"
	leftDs := *ds

	// alter body file
	const alteredBodyData = `city,pop,avg_age,in_usa
toronto,4000000,55.0,false
new york,850000,44.0,false
chicago,30000,440.4,false
chatham,3500,650.25,false
raleigh,25000,5000.65,false`

	updateDataset(t, mr, ds, alteredBodyData)

	res, err := svc.statsDiff(ctx, &leftDs, ds)
	if err != nil {
		t.Fatal(err)
	}

	// output order is not strict so we need to enfore it here
	sort.SliceStable(res.Columns, func(i, j int) bool {
		return res.Columns[i].Title < res.Columns[j].Title
	})

	expect := &StatsChangeComponent{
		Summary: &ChangeReportDeltaComponent{
			ChangeReportComponent: ChangeReportComponent{
				Left:  StatsChangeSummaryFields{Entries: 5, Columns: 4, TotalSize: 155},
				Right: StatsChangeSummaryFields{Entries: 5, Columns: 4, TotalSize: 157},
			},
			Delta: StatsChangeSummaryFields{
				Entries:   0,
				Columns:   0,
				TotalSize: 2,
			},
		},
		Columns: getBaseCols(),
	}

	if diff := cmp.Diff(res, expect); diff != "" {
		t.Errorf("stat component result mismatch. (-want +got):%s\n", diff)
	}
}

func TestParseColumns(t *testing.T) {
	ctx := context.Background()

	workDir, err := ioutil.TempDir("", "qri_test_changes_service")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)

	mr, err := repotest.NewTestRepo()
	if err != nil {
		t.Fatal(err)
	}

	svc := newTestService(t, mr, workDir)

	ref := dsref.MustParse("peer/cities")
	if _, err := mr.ResolveRef(ctx, &ref); err != nil {
		t.Fatal(err)
	}

	ds, err := dsfs.LoadDataset(ctx, mr.Filesystem(), ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err = base.OpenDataset(ctx, mr.Filesystem(), ds); err != nil {
		t.Fatal(err)
	}

	var colItems tabular.Columns
	summary, err := svc.parseColumns(&colItems, ds)
	if err != nil {
		t.Fatal(err)
	}

	expectColItems := tabular.Columns{
		tabular.Column{
			Title: "city",
			Type:  &tabular.ColType{"string"},
		},
		tabular.Column{
			Title: "pop",
			Type:  &tabular.ColType{"integer"},
		},
		tabular.Column{
			Title: "avg_age",
			Type:  &tabular.ColType{"number"},
		},
		tabular.Column{
			Title: "in_usa",
			Type:  &tabular.ColType{"boolean"},
		},
	}

	if diff := cmp.Diff(colItems, expectColItems); diff != "" {
		t.Errorf("column items result mismatch. (-want +got):%s\n", diff)
	}

	expectSummary := StatsChangeSummaryFields{
		Entries:   5,
		Columns:   4,
		TotalSize: 155,
	}

	if diff := cmp.Diff(summary, expectSummary); diff != "" {
		t.Errorf("stats summary result mismatch. (-want +got):%s\n", diff)
	}
}

func TestMaybeLoadStats(t *testing.T) {
	ctx := context.Background()

	workDir, err := ioutil.TempDir("", "qri_test_changes_service")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)

	mr, err := repotest.NewTestRepo()
	if err != nil {
		t.Fatal(err)
	}

	svc := newTestService(t, mr, workDir)

	ref := dsref.MustParse("peer/cities")
	if _, err := mr.ResolveRef(ctx, &ref); err != nil {
		t.Fatal(err)
	}

	ds, err := dsfs.LoadDataset(ctx, mr.Filesystem(), ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err = base.OpenDataset(ctx, mr.Filesystem(), ds); err != nil {
		t.Fatal(err)
	}

	if ds.Stats == nil {
		t.Fatal("stats are nil")
	}

	ds.Stats = nil

	svc.maybeLoadStats(ctx, ds)
	if ds.Stats == nil {
		t.Fatal("stats are nil")
	}
}

func TestMatchColumns(t *testing.T) {
	ctx := context.Background()

	workDir, err := ioutil.TempDir("", "qri_test_changes_service")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)

	mr, err := repotest.NewTestRepo()
	if err != nil {
		t.Fatal(err)
	}

	svc := newTestService(t, mr, workDir)

	ref := dsref.MustParse("peer/cities")
	if _, err := mr.ResolveRef(ctx, &ref); err != nil {
		t.Fatal(err)
	}

	ds, err := dsfs.LoadDataset(ctx, mr.Filesystem(), ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err = base.OpenDataset(ctx, mr.Filesystem(), ds); err != nil {
		t.Fatal(err)
	}

	ds.Name = "cities"
	leftDs := *ds

	// alter body file
	const alteredBodyData = `city,pop,avg_age,in_usa
toronto,4000000,55.0,false
new york,850000,44.0,false
chicago,30000,440.4,false
chatham,3500,650.25,false
raleigh,25000,5000.65,false`

	updateDataset(t, mr, ds, alteredBodyData)

	var leftColItems tabular.Columns
	_, err = svc.parseColumns(&leftColItems, &leftDs)
	if err != nil {
		t.Fatal(err)
	}
	leftStats, err := svc.parseStats(&leftDs)
	if err != nil {
		t.Fatal(err)
	}

	var rightColItems tabular.Columns
	_, err = svc.parseColumns(&rightColItems, ds)
	if err != nil {
		t.Fatal(err)
	}
	rightStats, err := svc.parseStats(ds)
	if err != nil {
		t.Fatal(err)
	}

	report, err := svc.matchColumns(4, 4, leftColItems, rightColItems, leftStats, rightStats)
	if err != nil {
		t.Fatal(err)
	}

	// output order is not strict so we need to enfore it here
	sort.SliceStable(report, func(i, j int) bool {
		return report[i].Title < report[j].Title
	})

	expect := getBaseCols()

	if diff := cmp.Diff(report, expect); diff != "" {
		t.Errorf("column items result mismatch. (-want +got):%s\n", diff)
	}

	// alter body file - remove column
	const alteredBodyDataColumns1 = `city,avg_age,in_usa
toronto,55.0,false
new york,44.0,false
chicago,440.4,false
chatham,650.25,false
raleigh,5000.65,false`

	ds.Name = "cities"

	updateDataset(t, mr, ds, alteredBodyDataColumns1)
	if err = base.OpenDataset(ctx, mr.Filesystem(), ds); err != nil {
		t.Fatal(err)
	}

	_, err = svc.parseColumns(&rightColItems, ds)
	if err != nil {
		t.Fatal(err)
	}
	rightStats, err = svc.parseStats(ds)
	if err != nil {
		t.Fatal(err)
	}

	report, err = svc.matchColumns(4, 3, leftColItems, rightColItems, leftStats, rightStats)
	if err != nil {
		t.Fatal(err)
	}

	// output order is not strict so we need to enfore it here
	sort.SliceStable(report, func(i, j int) bool {
		return report[i].Title < report[j].Title
	})

	expect = getBaseCols()
	expect[3] = &ChangeReportDeltaComponent{
		ChangeReportComponent: ChangeReportComponent{
			Left: EmptyObject{
				"count":  float64(5),
				"max":    float64(40000000),
				"min":    float64(35000),
				"mean":   float64(9817000),
				"median": float64(300000),
				"histogram": map[string]interface{}{
					"bins": []interface{}{
						float64(35000),
						float64(250000),
						float64(300000),
						float64(8500000),
						float64(40000000),
						float64(40000001),
					},
					"frequencies": []interface{}{
						float64(1),
						float64(1),
						float64(1),
						float64(1),
						float64(1),
					},
				},
				"type": "numeric",
			},
			Right: EmptyObject{},
			About: map[string]interface{}{
				"status": fsi.STRemoved,
			},
		},
		Title: "pop",
		Delta: map[string]interface{}{
			"count":  float64(-5),
			"max":    float64(-40000000),
			"mean":   float64(-9817000),
			"median": float64(-300000),
			"min":    float64(-35000),
		},
	}

	if diff := cmp.Diff(report, expect); diff != "" {
		t.Errorf("column items result mismatch. (-want +got):%s\n", diff)
	}

	// alter body file - add column
	const alteredBodyDataColumns2 = `city,pop,avg_age,in_usa,score
toronto,4000000,55.0,false,1
new york,850000,44.0,false,2
chicago,30000,440.4,false,3
chatham,3500,650.25,false,4
raleigh,25000,5000.65,false,5`

	ds.Name = "cities"

	updateDataset(t, mr, ds, alteredBodyDataColumns2)

	_, err = svc.parseColumns(&rightColItems, ds)
	if err != nil {
		t.Fatal(err)
	}
	rightStats, err = svc.parseStats(ds)
	if err != nil {
		t.Fatal(err)
	}

	report, err = svc.matchColumns(4, 5, leftColItems, rightColItems, leftStats, rightStats)
	if err != nil {
		t.Fatal(err)
	}

	// output order is not strict so we need to enfore it here
	sort.SliceStable(report, func(i, j int) bool {
		return report[i].Title < report[j].Title
	})

	expect = getBaseCols()
	expect = append(expect, &ChangeReportDeltaComponent{
		ChangeReportComponent: ChangeReportComponent{
			Left: EmptyObject{},
			Right: EmptyObject{
				"count":  float64(5),
				"max":    float64(5),
				"min":    float64(1),
				"mean":   float64(3),
				"median": float64(3),
				"histogram": map[string]interface{}{
					"bins": []interface{}{
						float64(1),
						float64(2),
						float64(3),
						float64(4),
						float64(5),
						float64(6),
					},
					"frequencies": []interface{}{
						float64(1),
						float64(1),
						float64(1),
						float64(1),
						float64(1),
					},
				},
				"type": "numeric",
			},
			About: map[string]interface{}{
				"status": fsi.STAdd,
			},
		},
		Title: "score",
		Delta: map[string]interface{}{
			"count":  float64(5),
			"max":    float64(5),
			"mean":   float64(3),
			"median": float64(3),
			"min":    float64(1),
		},
	})

	if diff := cmp.Diff(report, expect); diff != "" {
		t.Errorf("column items result mismatch. (-want +got):%s\n", diff)
	}
}
