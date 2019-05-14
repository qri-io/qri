package lib

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/update/cron"
)

func TestUpdateMethods(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "update_methods")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfigForTesting()
	cfg.Update = &config.Update{Type: "mem"}
	cfg.Repo = &config.Repo{Type: "mem", Middleware: []string{}}
	cfg.Store = &config.Store{Type: "map"}

	inst, err := NewInstance(tmpDir, OptConfig(cfg), OptIOStreams(ioes.NewDiscardIOStreams()))
	if err != nil {
		t.Fatal(err)
	}

	m := NewUpdateMethods(inst)

	shellJob := &ScheduleParams{
		// this'll create type ShellScript with the .sh extension
		Name: "testdata/hello.sh",
		// run one time after one second
		Periodicity: "R1/PT1S",
	}
	shellRes := &Job{}
	if err := m.Schedule(shellJob, shellRes); err != nil {
		t.Fatal(err)
	}

	// TODO - test repo currently doesn't have a configured profile, so this isn't
	// working
	ref := addNowTransformDataset(t, inst.Node())

	dsJob := &ScheduleParams{
		Name: ref.AliasString(),
		// run one time after ten seconds
		// TODO (b5) - currently we *don't* want this code to run, because it'll
		// invoke the compiled qri command. need to figure out a way to test this,
		// possibly by overriding the "qri" cmd with some call to a binary that'll
		// accept any argument and return a zero exit code
		Periodicity: "R1/PT10S",
		SaveParams: &SaveParams{
			BodyPath:  "testdata/cities_2/body.csv",
			FilePaths: []string{"testdata/component_files/meta.json"},
			Title:     "hallo",
		},
	}
	dsRes := &Job{}
	if err := m.Schedule(dsJob, dsRes); err != nil {
		t.Fatal(err)
	}
	dsName := dsRes.Name
	t.Log(dsName, dsRes)

	// run the service for one second to generate updates
	// sorry tests, y'all gotta run a little slower :/
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer done()
	if err := inst.cron.(*cron.Cron).Start(ctx); err != nil {
		t.Fatal(err)
	}

	res := []*Job{}
	if err := m.Logs(&ListParams{Offset: 0, Limit: -1}, &res); err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Errorf("expected 1 log entry, got: %d", len(res))
	}

	if err := m.List(&ListParams{Offset: 0, Limit: -1}, &res); err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Errorf("expected 2 list entries, got: %d", len(res))
	}

	jobRes := &Job{}
	if err := m.Job(&dsName, jobRes); err != nil {
		t.Error(err)
	}

	shName := "testdata/hello.sh"
	var fin bool
	if err := m.Unschedule(&shName, &fin); err != nil {
		t.Error(err)
	}

}

func TestUpdateServiceStart(t *testing.T) {
	inst := &Instance{}
	m := NewUpdateMethods(inst)

	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(time.Second/4))
	defer done()

	p := &UpdateServiceStartParams{
		Ctx:       ctx,
		Daemonize: false,

		RepoPath:  "",
		UpdateCfg: &config.Update{Type: "mem"},
	}

	var res bool
	if err := m.ServiceStart(p, &res); err != nil {
		t.Fatal(err)
	}
}

func TestDatasetMethodsRun(t *testing.T) {
	node := newTestQriNode(t)
	inst := &Instance{node: node}

	m := NewUpdateMethods(inst)
	res := &repo.DatasetRef{}
	if err := m.Run(&Job{Name: "me/bad_dataset", Type: cron.JTDataset}, res); err == nil {
		t.Error("expected update to nonexistent dataset to error")
	}

	ref := addNowTransformDataset(t, node)
	res = &repo.DatasetRef{}
	if err := m.Run(&Job{Name: ref.AliasString(), Type: cron.JTDataset /* Recall: "tf", ReturnBody: true */}, res); err != nil {
		t.Errorf("update error: %s", err)
	}

	metaPath := tempDatasetFile(t, "*-methods-meta.json", &dataset.Dataset{
		Meta: &dataset.Meta{Title: "an updated title"},
	})
	defer func() {
		os.RemoveAll(metaPath)
	}()

	dsm := NewDatasetRequests(inst.node, nil)
	// run a manual save to lose the transform
	err := dsm.Save(&SaveParams{
		Ref:       res.AliasString(),
		FilePaths: []string{metaPath},
	}, res)
	if err != nil {
		t.Error(err)
	}

	// update should grab the transform from 2 commits back
	if err := m.Run(&Job{Name: res.AliasString(), Type: cron.JTDataset /* ReturnBody: true */}, res); err != nil {
		t.Error(err)
	}
}
