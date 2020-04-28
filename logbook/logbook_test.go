package logbook

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook/oplog"
)

func Example() {
	// background context to play with
	ctx := context.Background()

	// logbooks are encrypted at rest, we need a private key to interact with
	// them, including to create a new logbook. This is a dummy Private Key
	// you should never, ever use in real life. demo only folks.
	testPk := `CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`
	data, err := base64.StdEncoding.DecodeString(testPk)
	if err != nil {
		panic(err)
	}
	pk, err := crypto.UnmarshalPrivateKey(data)
	if err != nil {
		panic(fmt.Errorf("error unmarshaling private key: %s", err.Error()))
	}

	// logbook relies on a qfs.Filesystem for read & write. create an in-memory
	// filesystem we can play with
	fs := qfs.NewMemFS()

	// Create a new journal for b5, passing in:
	//  * the author private key to encrypt & decrypt the logbook
	//  * author's current username
	//  * a qfs.Filesystem for reading & writing the logbook
	//  * a base path on the filesystem to read & write the logbook to
	// Initializing a logbook ensures the author has an user opset that matches
	// their current state. It will error if a stored book can't be decrypted
	book, err := NewJournal(pk, "b5", fs, "/mem/logset")
	if err != nil {
		panic(err) // real programs don't panic
	}

	// create a name to store dataset versions in. NameInit will create a new
	// log under the logbook author's namespace with the given name, and an opset
	// that tracks operations by this author within that new namespace.
	// The entire logbook is persisted to the filestore after each operation
	initID, err := book.WriteDatasetInit(ctx, "world_bank_population")
	if err != nil {
		panic(err)
	}

	// pretend we've just created a dataset, these are the only fields the log
	// will care about
	ds := &dataset.Dataset{
		Peername: "b5",
		Name:     "world_bank_population",
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path:         "QmHashOfVersion1",
		PreviousPath: "",
		// TODO (b5) - at some point we may want to log parent versions as well,
		// need to model those properly first.
	}

	// create a log record of the version of a dataset. In practice this'll be
	// part of the overall save routine that created the above ds variable
	if err := book.WriteVersionSave(ctx, initID, ds); err != nil {
		panic(err)
	}

	// sometime later, we create another version
	ds2 := &dataset.Dataset{
		Peername: "b5",
		Name:     "world_bank_population",
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			Title:     "added body data",
		},
		Structure: &dataset.Structure{
			Length: 100,
		},
		Path:         "QmHashOfVersion2",
		PreviousPath: "QmHashOfVersion1",
	}

	// once again, write to the log
	if err := book.WriteVersionSave(ctx, initID, ds2); err != nil {
		panic(err)
	}

	ref := dsref.Ref{
		Username: "b5",
		Name:     "world_bank_population",
	}

	// pretend we just published both saved versions of the dataset to the
	// registry we log that here. Providing a revisions arg of 2 means we've
	// published two consecutive revisions from head: the latest version, and the
	// one before it. "registry.qri.cloud" indicates we published to a single
	// destination with that name.
	if err := book.WritePublish(ctx, initID, 2, "registry.qri.cloud"); err != nil {
		panic(err)
	}

	// pretend the user just deleted a dataset version, well, we need to log it!
	// VersionDelete accepts an argument of number of versions back from HEAD
	// more complex deletes that remove pieces of history may require either
	// composing multiple log operations
	book.WriteVersionDelete(ctx, initID, 1)

	// create another version
	ds3 := &dataset.Dataset{
		Peername: "b5",
		Name:     "world_bank_population",
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
			Title:     "added meta info",
		},
		Structure: &dataset.Structure{
			Length: 100,
		},
		Path: "QmHashOfVersion3",
		// note that we're referring to version 1 here. version 2 no longer exists
		// this is happening outside of the log, but the log should reflect
		// contiguous history
		PreviousPath: "QmHashOfVersion1",
	}

	// once again, write to the log
	if err := book.WriteVersionSave(ctx, initID, ds3); err != nil {
		panic(err)
	}

	// now for the fun bit. When we ask for the state of the log, it will
	// play our opsets forward and get us the current state of tne log
	// we can also get the state of a log from the book:
	log, err := book.Items(ctx, ref, 0, 100)
	if err != nil {
		panic(err)
	}

	for _, info := range log {
		fmt.Println(info.SimpleRef().String())
	}

	// Output:
	// b5/world_bank_population@QmHashOfVersion3
	// b5/world_bank_population@QmHashOfVersion1
}

func TestNewJournal(t *testing.T) {
	pk := testPrivKey(t)
	fs := qfs.NewMemFS()

	if _, err := NewJournal(nil, "b5", nil, "/mem/logset"); err == nil {
		t.Errorf("expected missing private key arg to error")
	}
	if _, err := NewJournal(pk, "", nil, "/mem/logset"); err == nil {
		t.Errorf("expected missing author arg to error")
	}
	if _, err := NewJournal(pk, "b5", nil, "/mem/logset"); err == nil {
		t.Errorf("expected missing filesystem arg to error")
	}
	if _, err := NewJournal(pk, "b5", fs, ""); err == nil {
		t.Errorf("expected missing location arg to error")
	}

	_, err := NewJournal(pk, "b5", fs, "/mem/logset")
	if err != nil {
		t.Fatal(err)
	}
}

func TestNilCallable(t *testing.T) {
	var (
		book   *Book
		initID = ""
		ctx    = context.Background()
		err    error
	)

	if _, err = book.ActivePeerID(ctx); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.MergeLog(ctx, nil, &oplog.Log{}); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.RemoveLog(ctx, nil, dsref.Ref{}); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.ConstructDatasetLog(ctx, dsref.Ref{}, nil); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteAuthorRename(ctx, ""); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if _, err = book.WriteDatasetInit(ctx, ""); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteDatasetRename(ctx, initID, ""); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteDatasetDelete(ctx, initID); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WritePublish(ctx, initID, 0); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteUnpublish(ctx, initID, 0); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteVersionAmend(ctx, initID, nil); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteVersionDelete(ctx, initID, 0); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteVersionSave(ctx, initID, nil); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if _, err = book.ResolveRef(ctx, nil); err != dsref.ErrNotFound {
		t.Errorf("expected '%s', got: %v", dsref.ErrNotFound, err)
	}
}

func TestResolveRef(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	if _, err := (*Book)(nil).ResolveRef(tr.Ctx, nil); err != dsref.ErrNotFound {
		t.Errorf("book ResolveRef must be nil-callable. expected: %q, got %v", dsref.ErrNotFound, err)
	}

	if _, err := tr.Book.ResolveRef(tr.Ctx, &dsref.Ref{Username: "username", Name: "does_not_exist"}); err != dsref.ErrNotFound {
		t.Errorf("expeted standard error resolving nonexistent ref: %q, got: %q", dsref.ErrNotFound, err)
	}

	tr.WriteWorldBankExample(t)

	resolveMe := dsref.Ref{
		Username: tr.WorldBankRef().Username,
		Name:     tr.WorldBankRef().Name,
	}

	source, err := tr.Book.ResolveRef(tr.Ctx, &resolveMe)
	if err != nil {
		t.Error(err)
	}

	// logbook resolves locally, expect the empty string
	expectSource := ""

	if diff := cmp.Diff(expectSource, source); diff != "" {
		t.Errorf("source mismatch (-want +got):\n%s", diff)
	}

	expect := dsref.Ref{
		Username: tr.WorldBankRef().Username,
		Name:     tr.WorldBankRef().Name,
		Path:     "QmHashOfVersion3",
		ID:       tr.WorldBankID(),
	}

	if diff := cmp.Diff(expect, resolveMe); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}

	resolveMe = dsref.Ref{
		Username: "me",
		Name:     tr.WorldBankRef().Name,
	}

	if _, err := tr.Book.ResolveRef(tr.Ctx, &resolveMe); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(expect, resolveMe); diff != "" {
		t.Errorf("'me' shortcut result mismatch. (-want +got):\n%s", diff)
	}
}

func TestBookLogEntries(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)

	entries, err := tr.Book.LogEntries(tr.Ctx, tr.WorldBankRef(), 0, 30)
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, len(entries))
	for i, entry := range entries {
		// convert timestamps to UTC for consistent output
		entry.Timestamp = entry.Timestamp.UTC()
		got[i] = entry.String()
		t.Log(got[i])
	}

	expect := []string{
		"12:02AM\ttest_author\tinit branch\tmain",
		"12:00AM\ttest_author\tsave commit\tinitial commit",
		"12:00AM\ttest_author\tsave commit\tadded body data",
		"12:00AM\ttest_author\tpublish\t",
		"12:00AM\ttest_author\tunpublish\t",
		"12:00AM\ttest_author\tremove commit\t",
		"12:00AM\ttest_author\tamend commit\tadded meta info",
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestUserDatasetRef(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteRenameExample(t)

	if _, err := tr.Book.UserDatasetRef(tr.Ctx, dsref.Ref{}); err == nil {
		t.Error("expected LogBytes with empty ref to fail")
	}
	if _, err := tr.Book.UserDatasetRef(tr.Ctx, dsref.Ref{Username: tr.Username}); err == nil {
		t.Error("expected LogBytes with empty name ref to fail")
	}
	if _, err := tr.Book.UserDatasetRef(tr.Ctx, tr.RenameRef()); err != nil {
		t.Errorf("expected LogBytes with proper ref to not produce an error. got: %s", err)
	}
}

func TestLogBytes(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteRenameExample(t)
	log, err := tr.Book.UserDatasetRef(tr.Ctx, tr.RenameRef())
	if err != nil {
		t.Error(err)
	}
	data, err := tr.Book.LogBytes(log)
	if err != nil {
		t.Error(err)
	}

	if len(data) < 1 {
		t.Errorf("expected data to be populated")
	}
}

func TestDsRefAliasForLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)
	tr.WriteRenameExample(t)
	egRef := tr.RenameRef()
	log, err := tr.Book.UserDatasetRef(tr.Ctx, egRef)
	if err != nil {
		t.Error(err)
	}

	if _, err := DsrefAliasForLog(nil); err == nil {
		t.Error("expected nil ref to error")
	}

	wrongModelLog, err := tr.Book.store.HeadRef(tr.Ctx, tr.Username)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DsrefAliasForLog(wrongModelLog); err == nil {
		t.Error("expected converting log of wrong model to error")
	}

	ambiguousLog, err := tr.Book.store.HeadRef(tr.Ctx, tr.Username)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DsrefAliasForLog(ambiguousLog); err == nil {
		t.Error("expected converting ambiguous logs to error")
	}

	ref, err := DsrefAliasForLog(log)
	if err != nil {
		t.Error(err)
	}

	expect := dsref.Ref{
		Username: egRef.Username,
		Name:     egRef.Name,
	}

	if diff := cmp.Diff(expect, ref); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}

func TestDatasetLogNaming(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()
	var err error

	if _, err = tr.Book.WriteDatasetInit(tr.Ctx, ""); err == nil {
		t.Errorf("expected initializing with an empty name to error")
	}
	firstInitID, err := tr.Book.WriteDatasetInit(tr.Ctx, "airport_codes")
	if err != nil {
		t.Fatalf("unexpected error writing valid dataset name: %s", err)
	}
	if _, err = tr.Book.WriteDatasetInit(tr.Ctx, "airport_codes"); err == nil {
		t.Error("expected initializing a name that already exists to error")
	}
	if err = tr.Book.WriteDatasetRename(tr.Ctx, firstInitID, "iata_airport_codes"); err != nil {
		t.Errorf("unexpected error renaming dataset: %s", err)
	}
	if _, err = tr.Book.RefToInitID(dsref.Ref{Username: "test_peer", Name: "airport_codes"}); err == nil {
		t.Error("expected finding the original name to error")
	}
	// Init another dataset with the old name, which is now available due to rename.
	if _, err = tr.Book.WriteDatasetInit(tr.Ctx, "airport_codes"); err != nil {
		t.Fatalf("unexpected error writing recently freed-up dataset name: %s", err)
	}
	if err = tr.Book.WriteDatasetDelete(tr.Ctx, firstInitID); err != nil {
		t.Errorf("unexpected error deleting first dataset: %s", err)
	}
	_, err = tr.Book.WriteDatasetInit(tr.Ctx, "iata_airport_codes")
	if err != nil {
		t.Errorf("expected initializing new name with deleted dataset to not error: %s", err)
	}

	const (
		profileID = "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"
		authorID  = "tz7ffwfj6e6z2xvdqgh2pf6gjkza5nzlncbjrj54s5s5eh46ma3q"
	)

	expect := []PlainLog{
		{
			Ops: []PlainOp{
				{Type: "init", Model: "user", Name: "test_author", AuthorID: profileID, Timestamp: mustTime("1999-12-31T19:00:00-05:00")},
			},
			Logs: []PlainLog{
				{
					Ops: []PlainOp{
						{Type: "init", Model: "dataset", Name: "airport_codes", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:01:00-05:00")},
						{Type: "amend", Model: "dataset", Name: "iata_airport_codes", Timestamp: mustTime("1999-12-31T19:03:00-05:00")},
						{Type: "remove", Model: "dataset", Timestamp: mustTime("1999-12-31T19:06:00-05:00")},
					},
					Logs: []PlainLog{
						{
							Ops: []PlainOp{
								{Type: "init", Model: "branch", Name: "main", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:02:00-05:00")},
							},
						},
					},
				},
				{
					Ops: []PlainOp{
						{Type: "init", Model: "dataset", Name: "airport_codes", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:04:00-05:00")},
					},
					Logs: []PlainLog{
						{
							Ops: []PlainOp{
								{Type: "init", Model: "branch", Name: "main", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:05:00-05:00")},
							},
						},
					},
				},
				{
					Ops: []PlainOp{
						{Type: "init", Model: "dataset", Name: "iata_airport_codes", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:07:00-05:00")},
					},
					Logs: []PlainLog{
						{
							Ops: []PlainOp{
								{Type: "init", Model: "branch", Name: "main", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:08:00-05:00")},
							},
						},
					},
				},
			},
		},
	}

	got, err := tr.Book.PlainLogs(tr.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestBookRawLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)

	got, err := tr.Book.PlainLogs(tr.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	// data, err := json.MarshalIndent(got, "", "  ")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// t.Logf("%s", string(data))

	expect := []PlainLog{
		{
			Ops: []PlainOp{
				{
					Type:      "init",
					Model:     "user",
					Name:      "test_author",
					AuthorID:  "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
					Timestamp: mustTime("1999-12-31T19:00:00-05:00"),
				},
			},
			Logs: []PlainLog{
				PlainLog{
					Ops: []PlainOp{
						{
							Type:      "init",
							Model:     "dataset",
							Name:      "world_bank_population",
							AuthorID:  "tz7ffwfj6e6z2xvdqgh2pf6gjkza5nzlncbjrj54s5s5eh46ma3q",
							Timestamp: mustTime("1999-12-31T19:01:00-05:00"),
						},
					},
					Logs: []PlainLog{
						PlainLog{
							Ops: []PlainOp{
								{
									Type:      "init",
									Model:     "branch",
									Name:      "main",
									AuthorID:  "tz7ffwfj6e6z2xvdqgh2pf6gjkza5nzlncbjrj54s5s5eh46ma3q",
									Timestamp: mustTime("1999-12-31T19:02:00-05:00"),
								},
								{
									Type:      "init",
									Model:     "commit",
									Ref:       "QmHashOfVersion1",
									Timestamp: mustTime("1999-12-31T19:00:00-05:00"),
									Note:      "initial commit",
								},
								{
									Type:      "init",
									Model:     "commit",
									Ref:       "QmHashOfVersion2",
									Prev:      "QmHashOfVersion1",
									Timestamp: mustTime("2000-01-01T19:00:00-05:00"),
									Note:      "added body data",
								},
								{
									Type:  "init",
									Model: "publication",
									Relations: []string{
										"registry.qri.cloud",
									},
									Timestamp: mustTime("1969-12-31T19:00:00-05:00"),
									Size:      2,
								},
								{
									Type:      "remove",
									Model:     "publication",
									Relations: []string{"registry.qri.cloud"},
									Timestamp: mustTime("1969-12-31T19:00:00-05:00"),
									Size:      2,
								},
								{
									Type:      "remove",
									Model:     "commit",
									Timestamp: mustTime("1969-12-31T19:00:00-05:00"),
									Size:      1,
								},
								{
									Type:      "amend",
									Model:     "commit",
									Ref:       "QmHashOfVersion3",
									Prev:      "QmHashOfVersion1",
									Timestamp: mustTime("2000-01-02T19:00:00-05:00"),
									Note:      "added meta info",
								},
							},
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestLogTransfer(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)
	tr.WriteRenameExample(t)

	log, err := tr.Book.UserDatasetRef(tr.Ctx, tr.WorldBankRef())
	if err != nil {
		t.Error(err)
	}

	if len(log.Logs) != 1 {
		t.Errorf("expected UserDatasetRef to only return one dataset log. got: %d", len(log.Logs))
	}

	pk2 := testPrivKey2(t)
	fs2 := qfs.NewMemFS()
	book2, err := NewJournal(pk2, "user2", fs2, "/mem/fs2_location")
	if err != nil {
		t.Fatal(err)
	}

	if err := book2.MergeLog(tr.Ctx, tr.Book.Author(), log); err == nil {
		t.Error("expected Merging unsigned log to fail")
	}

	if err := log.Sign(tr.Book.pk); err != nil {
		t.Error(err)
	}

	if err := book2.MergeLog(tr.Ctx, tr.Book.Author(), log); err != nil {
		t.Fatal(err)
	}

	revs, err := book2.Items(tr.Ctx, tr.WorldBankRef(), 0, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) == 0 {
		t.Errorf("expected book 2 to now have versions for world bank ref")
	}
}

func TestRenameAuthor(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)

	// fetching dataset for original author should work
	if _, err := tr.Book.BranchRef(tr.Ctx, tr.WorldBankRef()); err != nil {
		t.Fatalf("fetching %s should work. got: %s", tr.WorldBankRef(), err)
	}

	rename := "changed_username"
	if err := tr.Book.WriteAuthorRename(tr.Ctx, rename); err != nil {
		t.Fatalf("error renaming author: %s", err)
	}

	if rename != tr.Book.AuthorName() {
		t.Errorf("authorname mismatch. expected: %s, got: %s", rename, tr.Book.AuthorName())
	}

	// fetching dataset for original author should NOT work
	if _, err := tr.Book.BranchRef(tr.Ctx, tr.WorldBankRef()); err == nil {
		t.Fatalf("fetching %s must fail. got: %s", tr.WorldBankRef(), err)
	}

	r := dsref.Ref{Username: rename, Name: "world_bank_population"}
	if _, err := tr.Book.BranchRef(tr.Ctx, r); err != nil {
		t.Fatalf("fetching new ref shouldn't fail. got: %s", err)
	}

}

func TestRenameDataset(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteRenameExample(t)

	if _, err := tr.Book.LogEntries(tr.Ctx, tr.RenameInitialRef(), 0, 30); err == nil {
		t.Error("expected fetching renamed dataset to error")
	}

	entries, err := tr.Book.LogEntries(tr.Ctx, tr.RenameRef(), 0, 30)
	// entries, err := tr.Book.Logs(tr.RenameInitialRef(), 0, 30)
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, len(entries))
	for i, entry := range entries {
		// convert timestamps to UTC for consistent output
		entry.Timestamp = entry.Timestamp.UTC()
		got[i] = entry.String()
		t.Log(got[i])
	}

	expect := []string{
		"12:02AM\ttest_author\tinit branch\tmain",
		"12:00AM\ttest_author\tsave commit\tinitial commit",
		"12:00AM\ttest_author\tsave commit\tadded meta info",
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestItems(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	initID := tr.WriteWorldBankExample(t)
	tr.WriteMoreWorldBankCommits(t, initID)
	book := tr.Book

	items, err := book.Items(tr.Ctx, tr.WorldBankRef(), 0, 10)
	if err != nil {
		t.Error(err)
	}

	expect := []DatasetLogItem{
		{
			VersionInfo: dsref.VersionInfo{
				Username:   "test_author",
				Name:       "world_bank_population",
				Path:       "QmHashOfVersion5",
				CommitTime: mustTime("2000-01-04T19:00:00-05:00"),
			},
			CommitTitle: "v5",
		},
		{
			VersionInfo: dsref.VersionInfo{
				Username:   "test_author",
				Name:       "world_bank_population",
				Path:       "QmHashOfVersion4",
				CommitTime: mustTime("2000-01-03T19:00:00-05:00"),
			},
			CommitTitle: "v4",
		},
		{
			VersionInfo: dsref.VersionInfo{
				Username:   "test_author",
				Name:       "world_bank_population",
				Path:       "QmHashOfVersion3",
				CommitTime: mustTime("2000-01-02T19:00:00-05:00"),
			},
			CommitTitle: "added meta info",
		},
	}

	if diff := cmp.Diff(expect, items); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}

	items, err = book.Items(tr.Ctx, tr.WorldBankRef(), 1, 1)
	if err != nil {
		t.Error(err)
	}

	expect = []DatasetLogItem{
		{
			VersionInfo: dsref.VersionInfo{
				Username:   "test_author",
				Name:       "world_bank_population",
				Path:       "QmHashOfVersion4",
				CommitTime: mustTime("2000-01-03T19:00:00-05:00"),
			},
			CommitTitle: "v4",
		},
	}
	if diff := cmp.Diff(expect, items); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestConstructDatasetLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	book := tr.Book
	name := "to_reconstruct"
	ref := dsref.Ref{Username: tr.Username, Name: name}
	history := []*dataset.Dataset{
		&dataset.Dataset{
			Peername: tr.Username,
			Name:     name,
			Commit: &dataset.Commit{
				Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Title:     "initial commit",
			},
			Path: "HashOfVersion1",
		},
		&dataset.Dataset{
			Peername: tr.Username,
			Name:     name,
			Commit: &dataset.Commit{
				Timestamp: time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
				Title:     "commit 2",
			},
			Path:         "HashOfVersion2",
			PreviousPath: "HashOfVersion1",
		},
		&dataset.Dataset{
			Peername: tr.Username,
			Name:     name,
			Commit: &dataset.Commit{
				Timestamp: time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
				Title:     "commit 2",
			},
			Path:         "HashOfVersion3",
			PreviousPath: "HashOfVersion2",
		},
	}

	if err := book.ConstructDatasetLog(tr.Ctx, ref, history); err != nil {
		t.Errorf("error constructing history: %s", err)
	}

	if err := book.ConstructDatasetLog(tr.Ctx, ref, history); err == nil {
		t.Error("expected second call to reconstruct to error")
	}

	// now for the fun bit. When we ask for the state of the log, it will
	// play our opsets forward and get us the current state of tne log
	// we can also get the state of a log from the book:
	items, err := book.Items(tr.Ctx, ref, 0, 100)
	if err != nil {
		t.Errorf("getting items: %s", err)
	}

	if len(items) != 3 {
		t.Errorf("expected 3 dslog items to return from history. got: %d", len(items))
	}
}

func mustTime(str string) time.Time {
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		panic(err)
	}
	return t
}

type testRunner struct {
	Ctx      context.Context
	Username string
	Book     *Book
	Fs       qfs.Filesystem
	Pk       crypto.PrivKey
	Tick     int
}

func newTestRunner(t *testing.T) (tr *testRunner, cleanup func()) {
	ctx := context.Background()
	authorName := "test_author"
	pk := testPrivKey(t)
	fs := qfs.NewMemFS()
	prevTs := NewTimestamp
	tr = &testRunner{
		Ctx:      ctx,
		Username: authorName,
	}
	NewTimestamp = tr.newTimestamp

	var err error
	tr.Book, err = NewJournal(pk, authorName, fs, "/mem/logset")
	if err != nil {
		t.Fatalf("creating book: %s", err.Error())
	}

	cleanup = func() {
		NewTimestamp = prevTs
	}

	return tr, cleanup
}

func (tr *testRunner) newTimestamp() int64 {
	defer func() { tr.Tick++ }()
	t := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	return t.Add(time.Minute * time.Duration(tr.Tick)).UnixNano()
}

func (tr *testRunner) WorldBankRef() dsref.Ref {
	return dsref.Ref{Username: tr.Username, Name: "world_bank_population"}
}

func (tr *testRunner) WorldBankID() string {
	return "crwd4wku64be6uxu3wbfqj7z65vtps4jt5ayx5dpjq4e2k72ks7q"
}

func (tr *testRunner) WriteWorldBankExample(t *testing.T) {
	book := tr.Book
	name := "world_bank_population"

	initID, err := book.WriteDatasetInit(tr.Ctx, name)
	if err != nil {
		panic(err)
	}

	// pretend we've just created a dataset, these are the only fields the log
	// will care about
	ds := &dataset.Dataset{
		Peername: tr.Username,
		Name:     name,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path:         "QmHashOfVersion1",
		PreviousPath: "",
	}

	if err := book.WriteVersionSave(tr.Ctx, initID, ds); err != nil {
		panic(err)
	}

	// sometime later, we create another version
	ds.Commit = &dataset.Commit{
		Timestamp: time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
		Title:     "added body data",
	}
	ds.Path = "QmHashOfVersion2"
	ds.PreviousPath = "QmHashOfVersion1"

	if err := book.WriteVersionSave(tr.Ctx, initID, ds); err != nil {
		t.Fatal(err)
	}

	if err := book.WritePublish(tr.Ctx, initID, 2, "registry.qri.cloud"); err != nil {
		t.Fatal(err)
	}

	if err := book.WriteUnpublish(tr.Ctx, initID, 2, "registry.qri.cloud"); err != nil {
		t.Fatal(err)
	}

	book.WriteVersionDelete(tr.Ctx, initID, 1)

	ds.Commit.Timestamp = time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)
	ds.Commit.Title = "added meta info"
	ds.Path = "QmHashOfVersion3"
	ds.PreviousPath = "QmHashOfVersion1"

	if err := book.WriteVersionAmend(tr.Ctx, initID, ds); err != nil {
		t.Fatal(err)
	}

	return initID
}

func (tr *testRunner) WriteMoreWorldBankCommits(t *testing.T, initID string) {
	book := tr.Book
	name := "world_bank_population"
	ds := &dataset.Dataset{
		Peername: tr.Username,
		Name:     name,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC),
			Title:     "v4",
		},
		Path:         "QmHashOfVersion4",
		PreviousPath: "QmHashOfVersion3",
	}

	if err := book.WriteVersionSave(tr.Ctx, initID, ds); err != nil {
		panic(err)
	}

	ds = &dataset.Dataset{
		Peername: tr.Username,
		Name:     name,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 5, 0, 0, 0, 0, time.UTC),
			Title:     "v5",
		},
		Path:         "QmHashOfVersion5",
		PreviousPath: "QmHashOfVersion4",
	}

	if err := book.WriteVersionSave(tr.Ctx, initID, ds); err != nil {
		panic(err)
	}
}

func (tr *testRunner) RenameInitialRef() dsref.Ref {
	return dsref.Ref{Username: tr.Book.AuthorName(), Name: "dataset"}
}

func (tr *testRunner) RenameRef() dsref.Ref {
	return dsref.Ref{Username: tr.Book.AuthorName(), Name: "renamed_dataset"}
}

func (tr *testRunner) WriteRenameExample(t *testing.T) {
	book := tr.Book
	name := "dataset"
	rename := "renamed_dataset"

	initID, err := book.WriteDatasetInit(tr.Ctx, name)
	if err != nil {
		panic(err)
	}

	// pretend we've just created a dataset, these are the only fields the log
	// will care about
	ds := &dataset.Dataset{
		Peername: tr.Username,
		Name:     name,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path:         "QmHashOfVersion1",
		PreviousPath: "",
	}

	if err := book.WriteVersionSave(tr.Ctx, initID, ds); err != nil {
		panic(err)
	}

	ds.Commit.Timestamp = time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)
	ds.Commit.Title = "added meta info"
	ds.Path = "QmHashOfVersion2"
	ds.PreviousPath = "QmHashOfVersion1"

	if err := book.WriteVersionSave(tr.Ctx, initID, ds); err != nil {
		t.Fatal(err)
	}

	if err := book.WriteDatasetRename(tr.Ctx, initID, rename); err != nil {
		t.Fatal(err)
	}
}

func testPrivKey(t *testing.T) crypto.PrivKey {
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

func testPrivKey2(t *testing.T) crypto.PrivKey {
	// id: "QmTqawxrPeTRUKS4GSUURaC16o4etPSJv7Akq6a9xqGZUh"
	testPk := `CAASpwkwggSjAgEAAoIBAQDACiqtbAeIR0gKZZfWuNgDssXnQnEQNrAlISlNMrtULuCtsLBk2tZ4C508T4/JQHfvbazZ/aPvkhr9KBaH8AzDU3FngHQnWblGtfm/0FAXbXPfn6DZ1rbA9rx9XpVZ+pUBDve0YxTSPOo5wOOR9u30JEvO47n1R/bF+wtMRHvDyRuoy4H86XxwMR76LYbgSlJm6SSKnrAVoWR9zqjXdaF1QljO77VbivnR5aS9vQ5Sd1mktwgb3SYUMlEGedtcMdLd3MPVCLFzq6cdjhSwVAxZ3RowR2m0hSEE/p6CKH9xz4wkMmjVrADfQTYU7spym1NBaNCrW1f+r4ScDEqI1yynAgMBAAECggEAWuJ04C5IQk654XHDMnO4h8eLsa7YI3w+UNQo38gqr+SfoJQGZzTKW3XjrC9bNTu1hzK4o1JOy4qyCy11vE/3Olm7SeiZECZ+cOCemhDUVsIOHL9HONFNHHWpLwwcUsEs05tpz400xWrezwZirSnX47tpxTgxQcwVFg2Bg07F5BntepqX+Ns7s2XTEc7YO8o77viYbpfPSjrsToahWP7ngIL4ymDjrZjgWTPZC7AzobDbhjTh5XuVKh60eUz0O7/Ezj2QK00NNkkD7nplU0tojZF10qXKCbECPn3pocVPAetTkwB1Zabq2tC2Y10dYlef0B2fkktJ4PAJyMszx4toQQKBgQD+69aoMf3Wcbw1Z1e9IcOutArrnSi9N0lVD7X2B6HHQGbHkuVyEXR10/8u4HVtbM850ZQjVnSTa4i9XJAy98FWwNS4zFh3OWVhgp/hXIetIlZF72GEi/yVFBhFMcKvXEpO/orEXMOJRdLb/7kNpMvl4MQ/fGWOmQ3InkKxLZFJ+wKBgQDA2jUTvSjjFVtOJBYVuTkfO1DKRGu7QQqNeF978ZEoU0b887kPu2yzx9pK0PzjPffpfUsa9myDSu7rncBX1FP0gNmSIAUja2pwMvJDU2VmE3Ua30Z1gVG1enCdl5ZWufum8Q+0AUqVkBdhPxw+XDJStA95FUArJzeZ2MTwbZH0RQKBgDG188og1Ys36qfPW0C6kNpEqcyAfS1I1rgLtEQiAN5GJMTOVIgF91vy11Rg2QVZrp9ryyOI/HqzAZtLraMCxWURfWn8D1RQkQCO5HaiAKM2ivRgVffvBHZd0M3NglWH/cWhxZW9MTRXtWLJX2DVvh0504s9yuAf4Jw6oG7EoAx5AoGBAJluAURO/jSMTTQB6cAmuJdsbX4+qSc1O9wJpI3LRp06hAPDM7ycdIMjwTw8wLVaG96bXCF7ZCGggCzcOKanupOP34kuCGiBkRDqt2tw8f8gA875S+k4lXU4kFgQvf8JwHi02LVxQZF0LeWkfCfw2eiKcLT4fzDV5ppzp1tREQmxAoGAGOXFomnMU9WPxJp6oaw1ZimtOXcAGHzKwiwQiw7AAWbQ+8RrL2AQHq0hD0eeacOAKsh89OXsdS9iW9GQ1mFR3FA7Kp5srjCMKNMgNSBNIb49iiG9O6P6UcO+RbYGg3CkSTG33W8l2pFIjBrtGktF5GoJudAPR4RXhVsRYZMiGag=`
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
