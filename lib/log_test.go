package lib

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestHistoryRequestsLog(t *testing.T) {
	mr, refs, err := testrepo.NewTestRepoWithHistory()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
		return
	}

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	firstRef := refs[0].String()

	items := make([]DatasetLogItem, len(refs))
	for i, r := range refs {
		t.Logf("%d: %d %s\t%s\t%s", i, r.Dataset.Structure.Length, r.Dataset.Commit.Title, r.Path, r.Dataset.PreviousPath)
		items[i] = DatasetLogItem{
			Ref:         repo.ConvertToDsref(r),
			Published:   r.Published,
			CommitTitle: r.Dataset.Commit.Title,
			Local:       true,
			Size:        int64(r.Dataset.Structure.Length),
		}
	}

	cases := []struct {
		description string
		p           *LogParams
		refs        []DatasetLogItem
		err         string
	}{
		{"log list - empty",
			&LogParams{}, []DatasetLogItem{}, "repo: empty dataset reference"},
		{"log list - bad path",
			&LogParams{Ref: "/badpath"}, nil, "repo: not found"},
		{"log list - default",
			&LogParams{Ref: firstRef}, items, ""},
		{"log list - offset 0 limit 3",
			&LogParams{Ref: firstRef, ListParams: ListParams{Offset: 0, Limit: 3}}, items[:3], ""},
		{"log list - offset 3 limit 3",
			&LogParams{Ref: firstRef, ListParams: ListParams{Offset: 3, Limit: 3}}, items[3:], ""},
		{"log list - offset 6 limit 3",
			&LogParams{Ref: firstRef, ListParams: ListParams{Offset: 6, Limit: 3}}, nil, "repo: no history"},
	}

	req := NewLogRequests(node, nil)
	for _, c := range cases {
		got := []DatasetLogItem{}
		err := req.Log(c.p, &got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case '%s' error mismatch: expected: %s, got: %s", c.description, c.err, err)
			continue
		}

		if len(c.refs) != len(got) {
			t.Errorf("case '%s' expected log length error. expected: %d, got: %d", c.description, len(c.refs), len(got))
			continue
		}

		t.Logf("-- %s", c.description)
		for i, v := range c.refs {
			t.Logf("expect: %d got: %d", v.Size, got[i].Size)
		}

		if diff := cmp.Diff(c.refs, got, cmpopts.IgnoreFields(DatasetLogItem{}, "Timestamp"), cmpopts.IgnoreFields(dsref.Ref{}, "Path")); diff != "" {
			t.Errorf("case '%s' result mismatch (-want +got):\n%s", c.description, diff)
		}
	}
}

func TestHistoryRequestsLogEntries(t *testing.T) {
	mr, refs, err := testrepo.NewTestRepoWithHistory()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
		return
	}

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	firstRef := refs[0].String()
	req := NewLogRequests(node, nil)

	if err = req.Logbook(&RefListParams{}, nil); err == nil {
		t.Errorf("expected empty reference param to error")
	}

	res := []LogEntry{}
	if err = req.Logbook(&RefListParams{Ref: firstRef, Limit: 30}, &res); err != nil {
		t.Fatal(err)
	}

	result := make([]string, len(res))
	for i := range res {
		// set response times to zero for consistent results
		res[i].Timestamp = time.Time{}
		result[i] = res[i].String()
	}

	expect := []string{
		`12:00AM	peer	init branch	main`,
		`12:00AM	peer	save commit	initial commit`,
		`12:00AM	peer	save commit	initial commit`,
		`12:00AM	peer	save commit	initial commit`,
		`12:00AM	peer	save commit	initial commit`,
		`12:00AM	peer	save commit	initial commit`,
	}

	if diff := cmp.Diff(expect, result); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
