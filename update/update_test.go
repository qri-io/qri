package update

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/update/cron"
)

func TestJobFromDataset(t *testing.T) {
	ds := &dataset.Dataset{
		Peername: "b5",
		Name:     "libp2p_node_count",
		Commit: &dataset.Commit{
			// last update was Jan 1 2019
			Timestamp: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Meta: &dataset.Meta{
			// update once a week
			AccrualPeriodicity: "R/P1W",
		},
	}

	_, err := DatasetToJob(ds, "", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestJobFromShellScript(t *testing.T) {
	// ShellScriptToJob(qfs.NewMemfileBytes("test.sh", nil)
}

func TestDatasetJobToCmd(t *testing.T) {
	dsj := &cron.Job{
		Type: cron.JTDataset,
		Name: "me/foo",
		Options: &cron.DatasetOptions{
			Title:    "title",
			BodyPath: "body/path.csv",
			FilePaths: []string{
				"file/path/0.json",
				"file/path/1.json",
			},
			Message: "message",

			// TODO (b5): we only provide one boolean flag here that affects output.
			// b/c map iteration is randomized, adding more would produce inconsistent
			// strings. full bool flag support should be written as a separate test
			Publish:      true,
			ShouldRender: true,
		},
	}
	streams := ioes.NewDiscardIOStreams()
	cmd := JobToCmd(streams, dsj)

	expect := "qri save me/foo --title=title --message=message --body=body/path.csv --file=file/path/0.json --file=file/path/1.json --publish"
	got := strings.Join(cmd.Args, " ")
	if got != expect {
		t.Errorf("job string mismatch. expected:\n'%s'\ngot:\n'%s'", expect, got)
	}
}

func TestShellScriptJobToCmd(t *testing.T) {
	dsj := &cron.Job{
		Type: cron.JTShellScript,
		Name: "path/to/shell/script.sh",
	}
	streams := ioes.NewDiscardIOStreams()
	cmd := JobToCmd(streams, dsj)

	expect := "path/to/shell/script.sh"
	got := strings.Join(cmd.Args, " ")
	if got != expect {
		t.Errorf("job string mismatch. expected:\n'%s'\ngot:\n'%s'", expect, got)
	}
}

func TestShellScriptToJob(t *testing.T) {
	if _, err := ShellScriptToJob("", "", nil); err == nil {
		t.Errorf("expected error")
	}

	if _, err := ShellScriptToJob("testdata/hello.sh", "R/P1Y", nil); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestStart(t *testing.T) {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*200))
	defer done()

	// call factory here to ensure we can create a factory with this context
	Factory(ctx)

	if err := Start(ctx, "", &config.Update{Type: "mem"}, false); err != nil {
		t.Error(err)
	}
}
