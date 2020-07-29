package oplog

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/logbook/oplog/logfb"
)

var allowUnexported = cmp.AllowUnexported(
	Journal{},
	Log{},
)

func TestJournalID(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	expect := ""
	got := tr.Journal.ID()
	if expect != got {
		t.Errorf("empty ID mismatch. expected: %q, got: %q", expect, got)
	}

	if err := tr.Journal.SetID(tr.Ctx, "test_id"); err == nil {
		t.Errorf("expected setting an ID that doesn't exist to fail. got nil")
	}

	l := tr.AddAuthorLogTree(t)

	if err := tr.Journal.SetID(tr.Ctx, l.ID()); err != nil {
		t.Errorf("expected setting ID to an author log to not fail. got: %q", err)
	}

	expect = l.ID()
	got = tr.Journal.ID()
	if expect != got {
		t.Errorf("set ID mismatch. expected: %q, got: %q", expect, got)
	}
}

func TestJournalMerge(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()
	ctx := tr.Ctx

	if err := tr.Journal.MergeLog(ctx, &Log{}); err == nil {
		t.Error("exceted adding an empty log to fail")
	}

	a := &Log{Ops: []Op{
		{Type: OpTypeInit, AuthorID: "a"},
	}}
	if err := tr.Journal.MergeLog(ctx, a); err != nil {
		t.Error(err)
	}

	expectLen := 1
	gotLen := len(tr.Journal.logs)
	if expectLen != gotLen {
		t.Errorf("top level log length mismatch. expected: %d, got: %d", expectLen, gotLen)
	}

	a = &Log{
		Ops: []Op{
			{Type: OpTypeInit, AuthorID: "a"},
		},
		Logs: []*Log{
			{
				Ops: []Op{
					{Type: OpTypeInit, Model: 1, AuthorID: "a"},
				},
			},
		},
	}

	if err := tr.Journal.MergeLog(ctx, a); err != nil {
		t.Error(err)
	}

	if expectLen != gotLen {
		t.Errorf("top level log length shouldn't change after merging a child log. expected: %d, got: %d", expectLen, gotLen)
	}

	got, err := tr.Journal.Get(ctx, a.ID())
	if err != nil {
		t.Error(err)
	}

	if !got.Logs[0].Ops[0].Equal(a.Logs[0].Ops[0]) {
		t.Errorf("expected returned ops to be equal")
	}
}

func TestJournalFlatbuffer(t *testing.T) {
	log := InitLog(Op{
		Type:      OpTypeInit,
		Model:     0x1,
		Ref:       "QmRefHash",
		Prev:      "QmPrevHash",
		Relations: []string{"a", "b", "c"},
		Name:      "steve",
		AuthorID:  "QmSteveHash",
		Timestamp: 1,
		Size:      2,
		Note:      "note!",
	})
	log.Signature = []byte{1, 2, 3}

	log.AddChild(InitLog(Op{
		Type:      OpTypeInit,
		Model:     0x0002,
		Ref:       "QmRefHash",
		Name:      "steve",
		AuthorID:  "QmSteveHash",
		Timestamp: 2,
		Size:      2500000,
		Note:      "note?",
	}))

	j := &Journal{
		logs: []*Log{log},
	}

	data := j.flatbufferBytes()
	logsetfb := logfb.GetRootAsBook(data, 0)

	got := &Journal{}
	if err := got.unmarshalFlatbuffer(logsetfb); err != nil {
		t.Fatalf("unmarshalling flatbuffer bytes: %s", err.Error())
	}

	// TODO (b5) - need to ignore log.parent here. causes a stack overflow in cmp.Diff
	// we should file an issue with a test that demonstrates the error
	ignoreCircularPointers := cmpopts.IgnoreUnexported(Log{})

	if diff := cmp.Diff(j, got, allowUnexported, cmp.Comparer(comparePrivKeys), ignoreCircularPointers); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestJournalCiphertext(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	lg := tr.RandomLog(Op{
		Type:  OpTypeInit,
		Model: 0x1,
		Name:  "apples",
	}, 10)

	j := tr.Journal
	if err := j.MergeLog(tr.Ctx, lg); err != nil {
		t.Fatal(err)
	}

	gotcipher, err := j.FlatbufferCipher(tr.PrivKey)
	if err != nil {
		t.Fatalf("calculating flatbuffer cipher: %s", err.Error())
	}

	plaintext := j.flatbufferBytes()
	if bytes.Equal(gotcipher, plaintext) {
		t.Errorf("plaintext bytes & ciphertext bytes can't be equal")
	}

	// TODO (b5) - we should confirm the ciphertext isn't readable, but
	// this'll panic with out-of-bounds slice access...
	// ciphertextAsBook := logfb.GetRootAsBook(gotcipher, 0)
	// if err := book.unmarshalFlatbuffer(ciphertextAsBook); err == nil {
	// 	t.Errorf("ciphertext as book should not have worked")
	// }

	if err = j.UnmarshalFlatbufferCipher(tr.Ctx, tr.PrivKey, gotcipher); err != nil {
		t.Errorf("book.UnmarhsalFlatbufferCipher unexpected error: %s", err.Error())
	}
}

func TestJournalSignLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	lg := tr.RandomLog(Op{
		Type:  OpTypeInit,
		Model: 0x1,
		Name:  "apples",
	}, 400)

	pk := tr.PrivKey
	if err := lg.Sign(pk); err != nil {
		t.Fatal(err)
	}
	data := lg.FlatbufferBytes()

	received, err := FromFlatbufferBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	if err := received.Verify(pk.GetPublic()); err != nil {
		t.Fatal(err)
	}
}

func TestLogHead(t *testing.T) {
	l := &Log{}
	if !l.Head().Equal(Op{}) {
		t.Errorf("expected empty log head to equal empty Op")
	}

	l = &Log{Ops: []Op{Op{}, Op{Model: 4}, Op{Model: 5, AuthorID: "foo"}}}
	if !l.Head().Equal(l.Ops[2]) {
		t.Errorf("expected log head to equal last Op")
	}
}

func TestLogGetID(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.AddAuthorLogTree(t)
	ctx := tr.Ctx

	got, err := tr.Journal.Get(ctx, "nonsense")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected not-found error for missing ID. got: %s", err)
	}

	root := tr.Journal.logs[0]
	got, err = tr.Journal.Get(ctx, root.ID())
	if err != nil {
		t.Errorf("unexpected error fetching root ID: %s", err)
	} else if !got.Head().Equal(root.Head()) {
		t.Errorf("returned log mismatch. Heads are different")
	}

	child := root.Logs[0]
	got, err = tr.Journal.Get(ctx, child.ID())
	if err != nil {
		t.Errorf("unexpected error fetching child ID: %s", err)
	}
	if !got.Head().Equal(child.Head()) {
		t.Errorf("returned log mismatch. Heads are different")
	}
}

func TestLogNameTracking(t *testing.T) {
	lg := InitLog(Op{
		Type:     OpTypeInit,
		Model:    0x01,
		Name:     "apples",
		AuthorID: "authorID",
	})

	changeOp := Op{
		Type:     OpTypeAmend,
		Model:    0x01,
		Name:     "oranges",
		AuthorID: "authorID2",
	}
	lg.Append(changeOp)

	if lg.Name() != "oranges" {
		t.Logf("name mismatch. expected 'oranges', got: '%s'", lg.Name())
	}

	if lg.Author() != "authorID2" {
		t.Logf("name mismatch. expected 'authorID2', got: '%s'", lg.Author())
	}
}

// NB: This test currently doesn't / can't confirm merging sets Log.parent.
// the cmp package can't deal with cyclic references
func TestLogMerge(t *testing.T) {
	left := &Log{
		Signature: []byte{1, 2, 3},
		Ops: []Op{
			{
				Type:     OpTypeInit,
				Model:    0x1,
				AuthorID: "author",
				Name:     "root",
			},
		},
		Logs: []*Log{
			{
				Ops: []Op{
					{
						Type:     OpTypeInit,
						Model:    0x0002,
						AuthorID: "author",
						Name:     "child_a",
					},
					{
						Type:  OpTypeInit,
						Model: 0x0456,
					},
				},
			},
		},
	}

	right := &Log{
		Ops: []Op{
			{
				Type:     OpTypeInit,
				Model:    0x1,
				AuthorID: "author",
				Name:     "root",
			},
			{
				Type:  OpTypeInit,
				Model: 0x0011,
			},
		},
		Logs: []*Log{
			{
				Ops: []Op{
					{
						Type:     OpTypeInit,
						Model:    0x0002,
						AuthorID: "author",
						Name:     "child_a",
					},
				},
			},
			{
				Ops: []Op{
					{
						Type:     OpTypeInit,
						Model:    0x0002,
						AuthorID: "buthor",
						Name:     "child_b",
					},
				},
			},
		},
	}

	left.Merge(right)

	expect := &Log{
		Ops: []Op{
			{
				Type:     OpTypeInit,
				Model:    0x1,
				AuthorID: "author",
				Name:     "root",
			},
			{
				Type:  OpTypeInit,
				Model: 0x0011,
			},
		},
		Logs: []*Log{
			{
				Ops: []Op{
					{
						Type:     OpTypeInit,
						Model:    0x0002,
						AuthorID: "author",
						Name:     "child_a",
					},
					{
						Type:  OpTypeInit,
						Model: 0x0456,
					},
				},
			},
			{
				ParentID: "adguqcqnrpc2rwxdykvsvengsccd5kew3x7jhs52rspg2f5nbina",
				Ops: []Op{
					{
						Type:     OpTypeInit,
						Model:    0x0002,
						AuthorID: "buthor",
						Name:     "child_b",
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expect, left, allowUnexported, cmpopts.IgnoreUnexported(Log{})); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestHeadRefRemoveTracking(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	ctx := tr.Ctx

	l := &Log{
		Ops: []Op{
			{Type: OpTypeInit, Model: 1, Name: "a"},
		},
		Logs: []*Log{
			{
				Ops: []Op{
					{Type: OpTypeInit, Model: 2, Name: "a"},
				},
			},
			{
				Ops: []Op{
					{Type: OpTypeRemove, Model: 2, Name: "b"}, // "pre-deleted log"
				},
			},
		},
	}
	if err := tr.Journal.MergeLog(ctx, l); err != nil {
		t.Fatal(err)
	}

	aLog, err := tr.Journal.HeadRef(ctx, "a")
	if err != nil {
		t.Errorf("expected no error fetching head ref for a. got: %v", err)
	}
	if _, err = tr.Journal.HeadRef(ctx, "a", "a"); err != nil {
		t.Errorf("expected no error fetching head ref for a/a. got: %v", err)
	}
	if _, err = tr.Journal.HeadRef(ctx, "a", "b"); err != ErrNotFound {
		t.Errorf("expected removed log to be not found. got: %v", err)
	}

	// add a remove operation to "a":
	aLog.Ops = append(aLog.Ops, Op{Type: OpTypeRemove, Model: 1, Name: "a"})

	if _, err = tr.Journal.HeadRef(ctx, "a"); err != ErrNotFound {
		t.Errorf("expected removed log to be not found. got: %v", err)
	}
	if _, err = tr.Journal.HeadRef(ctx, "a", "a"); err != ErrNotFound {
		t.Errorf("expected child of removed log to be not found. got: %v", err)
	}

	expectLogs := []*Log{
		{
			Ops: []Op{
				{Type: OpTypeInit, Model: 1, Name: "a"},
				{Type: OpTypeRemove, Model: 1, Name: "a"},
			},
			Logs: []*Log{
				{
					Ops: []Op{
						{Type: OpTypeInit, Model: 2, Name: "a"},
					},
				},
				{
					Ops: []Op{
						{Type: OpTypeRemove, Model: 2, Name: "b"},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expectLogs, tr.Journal.logs, allowUnexported); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestLogTraversal(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.AddAuthorLogTree(t)
	ctx := tr.Ctx

	if _, err := tr.Journal.HeadRef(ctx); err == nil {
		t.Errorf("expected not providing a name to error")
	}

	if _, err := tr.Journal.HeadRef(ctx, "this", "isn't", "a", "thing"); err != ErrNotFound {
		t.Errorf("expected asking for nonexistent log to return ErrNotFound. got: %v", err)
	}

	got, err := tr.Journal.HeadRef(ctx, "root", "b", "bazinga")
	if err != nil {
		t.Error(err)
	}

	// t.Logf("%#v", tr.Book.logs[0])

	expect := &Log{
		Ops: []Op{
			{Type: OpTypeInit, Model: 0x0002, AuthorID: "buthor", Name: "bazinga"},
		},
	}

	if diff := cmp.Diff(expect, got, allowUnexported); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestRemoveLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.AddAuthorLogTree(t)
	ctx := tr.Ctx

	if err := tr.Journal.RemoveLog(ctx); err == nil {
		t.Errorf("expected no name remove to error")
	}

	if err := tr.Journal.RemoveLog(ctx, "root", "b", "bazinga"); err != nil {
		t.Error(err)
	}

	if log, err := tr.Journal.HeadRef(ctx, "root", "b", "bazinga"); err != ErrNotFound {
		t.Errorf("expected RemoveLog to remove log at path root/b/bazinga. got: %v. log: %v", err, log)
	}

	if err := tr.Journal.RemoveLog(ctx, "root", "b"); err != nil {
		t.Error(err)
	}

	if _, err := tr.Journal.HeadRef(ctx, "root", "b"); err != ErrNotFound {
		t.Error("expected RemoveLog to remove log at path root/b")
	}

	if err := tr.Journal.RemoveLog(ctx, "root"); err != nil {
		t.Error(err)
	}

	if _, err := tr.Journal.HeadRef(ctx, "root"); err != ErrNotFound {
		t.Error("expected RemoveLog to remove log at path root")
	}

	if err := tr.Journal.RemoveLog(ctx, "nonexistent"); err != ErrNotFound {
		t.Error("expected RemoveLog for nonexistent path to error")
	}
}

func TestLogID(t *testing.T) {
	l := &Log{}
	got := l.ID()
	if "" != got {
		t.Errorf("expected op hash of empty log to give the empty string, got: %s", got)
	}

	l = &Log{
		Ops: []Op{Op{Name: "hello"}},
	}
	got = l.ID()
	expect := "z7ghdteiybt7mopm5ysntbdr6ewiq5cfjlfev2v3ekbfbay6bp5q"
	if expect != got {
		t.Errorf("result mismatch, expect: %s, got: %s", expect, got)
	}

	// changing a feature like a timestamp should affect output hash
	l = &Log{
		Ops: []Op{Op{Name: "hello", Timestamp: 2}},
	}
	got = l.ID()
	expect = "7ixp5z4h2dzjyljkjn7sbnsu6vg22gpgozmcl7wpg33pl5qfs3ra"
	if expect != got {
		t.Errorf("result mismatch, expect: %s, got: %s", expect, got)
	}
}

type testRunner struct {
	Ctx        context.Context
	AuthorName string
	PrivKey    crypto.PrivKey
	Journal    *Journal
	gen        *opGenerator
}

type testFailer interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func newTestRunner(t testFailer) (tr testRunner, cleanup func()) {
	ctx := context.Background()
	authorName := "test_author"
	pk := testPrivKey(t)

	tr = testRunner{
		Ctx:        ctx,
		AuthorName: authorName,
		PrivKey:    pk,
		Journal:    &Journal{},
		gen:        &opGenerator{ctx: ctx, NoopProb: 60},
	}
	cleanup = func() {
		// noop
	}

	return tr, cleanup
}

func (tr testRunner) RandomLog(init Op, opCount int) *Log {
	lg := InitLog(init)
	for i := 0; i < opCount; i++ {
		lg.Append(tr.gen.Gen())
	}
	return lg
}

func testPrivKey(t testFailer) crypto.PrivKey {
	// logbooks are encrypted at rest, we need a private key to interact with
	// them, including to create a new logbook. This is a dummy Private Key
	// you should never, ever use in real life. demo only folks.
	testPk := `CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`
	data, err := base64.StdEncoding.DecodeString(testPk)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := crypto.UnmarshalPrivateKey(data)
	if err != nil {
		t.Fatalf("error unmarshaling private key: %s", err.Error())
	}
	return pk
}

func comparePrivKeys(a, b crypto.PrivKey) bool {
	if a == nil && b != nil || a != nil && b == nil {
		return false
	}

	abytes, err := a.Bytes()
	if err != nil {
		return false
	}

	bbytes, err := b.Bytes()
	if err != nil {
		return false
	}

	return string(abytes) == string(bbytes)
}

func (tr *testRunner) AddAuthorLogTree(t testFailer) *Log {
	tree := &Log{
		Ops: []Op{
			Op{
				Type:     OpTypeInit,
				Model:    0x1,
				AuthorID: "author",
				Name:     "root",
			},
			Op{
				Type:  OpTypeInit,
				Model: 0x11,
			},
		},
		Logs: []*Log{
			{
				Ops: []Op{
					Op{
						Type:     OpTypeInit,
						Model:    0x2,
						AuthorID: "author",
						Name:     "a",
					},
					Op{
						Type:  OpTypeInit,
						Model: 0x456,
					},
				},
			},
			{
				Ops: []Op{
					Op{
						Type:     OpTypeInit,
						Model:    0x2,
						AuthorID: "buthor",
						Name:     "b",
					},
				},
				Logs: []*Log{
					{
						Ops: []Op{
							{Type: OpTypeInit, Model: 0x0002, AuthorID: "buthor", Name: "bazinga"},
						},
					},
				},
			},
		},
	}

	if err := tr.Journal.MergeLog(tr.Ctx, tree); err != nil {
		t.Fatal(err)
	}

	return tree
}
