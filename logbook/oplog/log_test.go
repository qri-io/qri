package oplog

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/logbook/oplog/logfb"
)

var allowUnexported = cmp.AllowUnexported(
	Book{},
	Log{},
)

func TestBookFlatbuffer(t *testing.T) {
	pk := testPrivKey(t)
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

	book := &Book{
		pk:         pk,
		authorname: "must_preserve",
		logs:       []*Log{log},
	}

	data := book.flatbufferBytes()
	logsetfb := logfb.GetRootAsBook(data, 0)

	got := &Book{
		// re-provide private key, unmarshal flatbuffer must preserve this key
		// through the unmarshaling call
		pk: pk,
	}
	if err := got.unmarshalFlatbuffer(logsetfb); err != nil {
		t.Fatalf("unmarshalling flatbuffer bytes: %s", err.Error())
	}

	// TODO (b5) - need to ignore log.parent here. causes a stack overflow in cmp.Diff
	// we should file an issue with a test that demonstrates the error
	ignoreCircularPointers := cmpopts.IgnoreUnexported(Log{})

	if diff := cmp.Diff(book, got, allowUnexported, cmp.Comparer(comparePrivKeys), ignoreCircularPointers); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestBookCiphertext(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	lg := tr.RandomLog(Op{
		Type:  OpTypeInit,
		Model: 0x1,
		Name:  "apples",
	}, 10)

	book := tr.Book
	book.AppendLog(lg)

	gotcipher, err := book.FlatbufferCipher()
	if err != nil {
		t.Fatalf("calculating flatbuffer cipher: %s", err.Error())
	}

	plaintext := book.flatbufferBytes()
	if bytes.Equal(gotcipher, plaintext) {
		t.Errorf("plaintext bytes & ciphertext bytes can't be equal")
	}

	// TODO (b5) - we should confirm the ciphertext isn't readable, but
	// this'll panic with out-of-bounds slice access...
	// ciphertextAsBook := logfb.GetRootAsBook(gotcipher, 0)
	// if err := book.unmarshalFlatbuffer(ciphertextAsBook); err == nil {
	// 	t.Errorf("ciphertext as book should not have worked")
	// }

	if err = book.UnmarshalFlatbufferCipher(tr.Ctx, gotcipher); err != nil {
		t.Errorf("book.UnmarhsalFlatbufferCipher unexpected error: %s", err.Error())
	}
}

func TestBookSignLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	lg := tr.RandomLog(Op{
		Type:  OpTypeInit,
		Model: 0x1,
		Name:  "apples",
	}, 400)

	pk := tr.Book.pk
	data, err := lg.SignedFlatbufferBytes(pk)
	if err != nil {
		t.Fatal(err)
	}

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

	tr.AddAuthorLogTree()

	got, err := tr.Book.Log("nonsense")
	if err != ErrNotFound {
		t.Errorf("expected not-found error for missing ID. got: %s", err)
	}

	root := tr.Book.Logs()[0]
	got, err = tr.Book.Log(root.ID())
	if err != nil {
		t.Errorf("unexpected error fetching root ID: %s", err)
	} else if !got.Head().Equal(root.Head()) {
		t.Errorf("returned log mismatch. Heads are different")
	}

	child := root.Logs[0]
	got, err = tr.Book.Log(child.ID())
	if err != nil {
		t.Errorf("unexpected error fetching child ID: %s", err)
	}
	if !got.Head().Equal(child.Head()) {
		t.Errorf("returned log mismatch. Heads are different")
	}
}

func TestNameTracking(t *testing.T) {
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

func TestLogMerge(t *testing.T) {
	left := &Log{
		Signature: []byte{1, 2, 3},
		Ops: []Op{
			Op{
				Type:     OpTypeInit,
				Model:    0x1,
				AuthorID: "author",
				Name:     "root",
			},
		},
		Logs: []*Log{
			{
				Ops: []Op{
					Op{
						Type:     OpTypeInit,
						Model:    0x0002,
						AuthorID: "author",
						Name:     "child_a",
					},
					Op{
						Type:  OpTypeInit,
						Model: 0x0456,
					},
				},
			},
		},
	}

	right := &Log{
		Ops: []Op{
			Op{
				Type:     OpTypeInit,
				Model:    0x1,
				AuthorID: "author",
				Name:     "root",
			},
			Op{
				Type:  OpTypeInit,
				Model: 0x0011,
			},
		},
		Logs: []*Log{
			{
				Ops: []Op{
					Op{
						Type:     OpTypeInit,
						Model:    0x0002,
						AuthorID: "author",
						Name:     "child_a",
					},
				},
			},
			{
				Ops: []Op{
					Op{
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
			Op{
				Type:     OpTypeInit,
				Model:    0x1,
				AuthorID: "author",
				Name:     "root",
			},
			Op{
				Type:  OpTypeInit,
				Model: 0x0011,
			},
		},
		Logs: []*Log{
			{
				Ops: []Op{
					Op{
						Type:     OpTypeInit,
						Model:    0x0002,
						AuthorID: "author",
						Name:     "child_a",
					},
					Op{
						Type:  OpTypeInit,
						Model: 0x0456,
					},
				},
			},
			{
				Ops: []Op{
					Op{
						Type:     OpTypeInit,
						Model:    0x0002,
						AuthorID: "buthor",
						Name:     "child_b",
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expect, left, allowUnexported); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestLogTraversal(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.AddAuthorLogTree()

	if _, err := tr.Book.HeadRef(); err == nil {
		t.Errorf("expected not providing a name to error")
	}

	if _, err := tr.Book.HeadRef("this", "isn't", "a", "thing"); err != ErrNotFound {
		t.Errorf("expected asking for nonexistent log to return ErrNotFound. got: %v", err)
	}

	got, err := tr.Book.HeadRef("root", "b", "bazinga")
	if err != nil {
		t.Error(err)
	}

	t.Logf("%#v", tr.Book.Logs()[0])

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

	tr.AddAuthorLogTree()

	if err := tr.Book.RemoveLog(); err == nil {
		t.Errorf("expected no name remove to error")
	}

	if err := tr.Book.RemoveLog("root", "b", "bazinga"); err != nil {
		t.Error(err)
	}

	if log, err := tr.Book.HeadRef("root", "b", "bazinga"); err != ErrNotFound {
		t.Errorf("expected RemoveLog to remove log at path root/b/bazinga. got: %v. log: %v", err, log)
	}

	if err := tr.Book.RemoveLog("root", "b"); err != nil {
		t.Error(err)
	}

	if _, err := tr.Book.HeadRef("root", "b"); err != ErrNotFound {
		t.Error("expected RemoveLog to remove log at path root/b")
	}

	if err := tr.Book.RemoveLog("root"); err != nil {
		t.Error(err)
	}

	if _, err := tr.Book.HeadRef("root"); err != ErrNotFound {
		t.Error("expected RemoveLog to remove log at path root")
	}

	if err := tr.Book.RemoveLog("nonexistent"); err != ErrNotFound {
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
	AuthorID   string
	Book       *Book
	gen        *opGenerator
}

type testFailer interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func newTestRunner(t testFailer) (tr testRunner, cleanup func()) {
	ctx := context.Background()
	authorName := "test_author"
	authorID := "QmTestAuthorID"
	pk := testPrivKey(t)

	book, err := NewBook(pk, authorName, authorID)
	if err != nil {
		t.Fatalf("creating book: %s", err.Error())
	}

	tr = testRunner{
		Ctx:        ctx,
		AuthorName: authorName,
		AuthorID:   authorID,
		Book:       book,
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

func (tr *testRunner) AddAuthorLogTree() {
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

	tr.Book.AppendLog(tree)
}
