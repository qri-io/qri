package lib

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/p2p"
	reporef "github.com/qri-io/qri/repo/ref"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestHistoryRequestsLog(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

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
		ds := r.Dataset
		items[i].VersionInfo = reporef.ConvertToVersionInfo(&r)
		items[i].MetaTitle = ""
		items[i].BodyRows = 0
		items[i].NumErrors = 0
		items[i].BodyFormat = ""
		if ds != nil && ds.Commit != nil {
			items[i].CommitTitle = ds.Commit.Title
			items[i].CommitMessage = ds.Commit.Message
		}
	}

	cases := []struct {
		description string
		p           *LogParams
		refs        []DatasetLogItem
		err         string
	}{
		{"log list - empty",
			&LogParams{}, []DatasetLogItem{}, `"" is not a valid dataset reference: empty reference`},
		{"log list - bad path",
			&LogParams{Ref: "/badpath"}, []DatasetLogItem{}, `"/badpath" is not a valid dataset reference: unexpected character at position 0: '/'`},
		{"log list - default",
			&LogParams{Ref: firstRef}, items, ""},
		{"log list - offset 0 limit 3",
			&LogParams{Ref: firstRef, ListParams: ListParams{Offset: 0, Limit: 3}}, items[:3], ""},
		{"log list - offset 3 limit 3",
			&LogParams{Ref: firstRef, ListParams: ListParams{Offset: 3, Limit: 3}}, items[3:], ""},
		{"log list - offset 6 limit 3",
			&LogParams{Ref: firstRef, ListParams: ListParams{Offset: 6, Limit: 3}}, nil, "repo: no history"},
	}

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewLogMethods(inst)
	for _, c := range cases {
		got := []DatasetLogItem{}
		err := m.Log(c.p, &got)

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
			t.Logf("expect: %d got: %d", v.BodySize, got[i].BodySize)
		}

		if diff := cmp.Diff(c.refs, got, cmpopts.IgnoreFields(dsref.VersionInfo{}, "CommitTime", "ProfileID"), cmpopts.IgnoreFields(dsref.Ref{}, "Path")); diff != "" {
			t.Errorf("case '%s' result mismatch (-want +got):\n%s", c.description, diff)
		}
	}
}

func TestHistoryRequestsLogEntries(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

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
	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewLogMethods(inst)

	if err = m.Logbook(&RefListParams{}, nil); err == nil {
		t.Errorf("expected empty reference param to error")
	}

	res := []LogEntry{}
	if err = m.Logbook(&RefListParams{Ref: firstRef, Limit: 30}, &res); err != nil {
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
