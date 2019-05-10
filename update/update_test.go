package update

import (
	"testing"
	"time"

	"github.com/qri-io/dataset"
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
