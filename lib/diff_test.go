package lib

import (
	"reflect"
	"testing"
	"time"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestDatasetRequestsDiff(t *testing.T) {
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Time{} }
	defer func() { dsfs.Timestamp = prevTs }()

	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	req := NewDatasetRequests(node, nil)

	// File 1
	fp1, err := dstest.BodyFilepath("testdata/jobs_by_automation")
	if err != nil {
		t.Errorf("getting data filepath: %s", err.Error())
		return
	}

	dsRef1 := repo.DatasetRef{}
	initParams := &SaveParams{
		Ref:      "me/jobs_ranked_by_automation_prob",
		BodyPath: fp1,
	}
	err = req.Save(initParams, &dsRef1)
	if err != nil {
		t.Errorf("couldn't init file 1: %s", err.Error())
		return
	}

	// File 2
	fp2, err := dstest.BodyFilepath("testdata/jobs_by_automation_2")
	if err != nil {
		t.Errorf("getting data filepath: %s", err.Error())
		return
	}
	dsRef2 := repo.DatasetRef{}
	initParams = &SaveParams{
		Ref:      "me/jobs_ranked_by_automation_prob",
		BodyPath: fp2,
	}
	err = req.Save(initParams, &dsRef2)
	if err != nil {
		t.Errorf("couldn't load second file: %s", err.Error())
		return
	}

	// need selected refs for tests
	mr.SetSelectedRefs([]repo.DatasetRef{dsRef2})

	successCases := []struct {
		description string
		Left, Right string
		Selector    string
		Stat        *DiffStat
		DeltaLen    int
	}{
		{"two fully qualified references",
			dsRef1.String(), dsRef2.String(),
			"",
			&DiffStat{Left: 42, Right: 44, LeftWeight: 2570, RightWeight: 2807, Inserts: 1, Updates: 7, Deletes: 0, Moves: 0},
			8,
		},
		{"fill left path from history",
			"", dsRef2.AliasString(),
			"",
			&DiffStat{Left: 42, Right: 44, LeftWeight: 2570, RightWeight: 2807, Inserts: 1, Updates: 7, Deletes: 0, Moves: 0},
			8,
		},
		{"populate from selected references",
			"", "",
			"",
			&DiffStat{Left: 42, Right: 44, LeftWeight: 2570, RightWeight: 2807, Inserts: 1, Updates: 7, Deletes: 0, Moves: 0},
			8,
		},
		{"two local file paths",
			"testdata/jobs_by_automation/body.csv", "testdata/jobs_by_automation_2/body.csv",
			"",
			&DiffStat{Left: 156, Right: 156, LeftWeight: 3897, RightWeight: 3909, Inserts: 0, Updates: 3, Deletes: 0, Moves: 0},
			3,
		},
		{"diff local csv & json file",
			"testdata/now_tf/input.dataset.json", "testdata/jobs_by_automation/body.csv",
			"",
			&DiffStat{Left: 17, Right: 156, LeftWeight: 250, RightWeight: 3897, Inserts: 156, Updates: 0, Deletes: 156, Moves: 0},
			2,
		},
	}

	// execute
	for i, c := range successCases {
		p := &DiffParams{
			LeftPath:  c.Left,
			RightPath: c.Right,
			Selector:  c.Selector,
		}
		res := &DiffResponse{}
		err := req.Diff(p, res)
		if err != nil {
			t.Errorf("%d. %s error: %s", i, c.description, err.Error())
			continue
		}

		if !reflect.DeepEqual(c.Stat, res.Stat) {
			t.Errorf("%d %s diffStat mismatch.\nwant: %v\ngot: %v\n", i, c.description, c.Stat, res.Stat)
		}

		if len(res.Diff) != c.DeltaLen {
			t.Errorf("%d %s delta length mismatch. want: %d got: %d", i, c.description, c.DeltaLen, len(res.Diff))
		}
	}
}
