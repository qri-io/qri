package lib

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
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

	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	firstRef := refs[0].String()

	items := make([]dsref.VersionInfo, len(refs))
	for i, r := range refs {
		ds := r.Dataset
		items[i] = reporef.ConvertToVersionInfo(&r)
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
		p           *ActivityParams
		refs        []dsref.VersionInfo
		err         string
	}{
		{"log list - empty",
			&ActivityParams{}, nil, `activity: ref required`},
		{"log list - bad path",
			&ActivityParams{Ref: "/badpath"}, nil, `"/badpath" is not a valid dataset reference: unexpected character at position 0: '/'`},
		{"log list - default",
			&ActivityParams{Ref: firstRef}, items, ""},
		{"log list - offset 0 limit 3",
			&ActivityParams{Ref: firstRef, ListParams: ListParams{Offset: 0, Limit: 3}}, items[:3], ""},
		{"log list - offset 3 limit 3",
			&ActivityParams{Ref: firstRef, ListParams: ListParams{Offset: 3, Limit: 3}}, items[3:], ""},
		{"log list - offset 6 limit 3",
			&ActivityParams{Ref: firstRef, ListParams: ListParams{Offset: 6, Limit: 3}}, nil, "repo: no history"},
	}

	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	m := inst.Dataset()
	for _, c := range cases {
		got, err := m.Activity(ctx, c.p)

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

	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	firstRef := refs[0].String()
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	if _, err = inst.Log().Log(ctx, &RefListParams{}); err == nil {
		t.Errorf("expected empty reference param to error")
	}

	res, err := inst.Log().Log(ctx, &RefListParams{Ref: firstRef, Limit: 30})
	if err != nil {
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
