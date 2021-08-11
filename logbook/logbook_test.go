package logbook_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/profile"
	profiletest "github.com/qri-io/qri/profile/test"
)

func Example() {
	ctx := context.Background()

	// logbooks are encrypted at rest, we need a private key to interact with
	// them, including to create a new logbook. This is a dummy Private Key
	// you should never, ever use in real life. demo only folks.
	yolanda := profiletest.GetProfile("yolanda_the_rat")

	// logbook relies on a qfs.Filesystem for read & write. create an in-memory
	// filesystem we can play with
	fs := qfs.NewMemFS()

	// Create a new journal for b5, passing in:
	//  * the author private key to encrypt & decrypt the logbook
	//  * author's current username
	//  * an event bus (not used in this example)
	//  * a qfs.Filesystem for reading & writing the logbook
	//  * a base path on the filesystem to read & write the logbook to
	// Initializing a logbook ensures the author has an user opset that matches
	// their current state. It will error if a stored book can't be decrypted
	book, err := logbook.NewJournal(*yolanda, event.NilBus, fs, "/mem/logbook.qfb")
	if err != nil {
		panic(err) // real programs don't panic
	}

	// create a name to store dataset versions in. NameInit will create a new
	// log under the logbook author's namespace with the given name, and an opset
	// that tracks operations by this author within that new namespace.
	// The entire logbook is persisted to the filestore after each operation
	initID, err := book.WriteDatasetInit(ctx, yolanda, "world_bank_population")
	if err != nil {
		panic(err)
	}

	// pretend we've just created a dataset, these are the only fields the log
	// will care about
	ds := &dataset.Dataset{
		ID:       initID,
		Peername: yolanda.Peername,
		Name:     "world_bank_population",
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path:         "QmHashOfVersion1",
		PreviousPath: "",
	}

	// create a log record of the version of a dataset. In practice this'll be
	// part of the overall save routine that created the above ds variable
	if err := book.WriteVersionSave(ctx, yolanda, ds, nil); err != nil {
		panic(err)
	}

	// sometime later, we create another version
	ds2 := &dataset.Dataset{
		ID:       initID,
		Peername: yolanda.Peername,
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
	if err := book.WriteVersionSave(ctx, yolanda, ds2, nil); err != nil {
		panic(err)
	}

	ref := dsref.Ref{
		Username: yolanda.Peername,
		Name:     "world_bank_population",
	}

	// pretend we just published both saved versions of the dataset to the
	// registry we log that here. Providing a revisions arg of 2 means we've
	// published two consecutive revisions from head: the latest version, and the
	// one before it. "registry.qri.cloud" indicates we published to a remote
	// with that address
	if _, _, err := book.WriteRemotePush(ctx, yolanda, initID, 2, "registry.qri.cloud"); err != nil {
		panic(err)
	}

	// pretend the user just deleted a dataset version, well, we need to log it!
	// VersionDelete accepts an argument of number of versions back from HEAD
	// more complex deletes that remove pieces of history may require either
	// composing multiple log operations
	book.WriteVersionDelete(ctx, yolanda, initID, 1)

	// create another version
	ds3 := &dataset.Dataset{
		ID:       initID,
		Peername: yolanda.Peername,
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
	if err := book.WriteVersionSave(ctx, yolanda, ds3, nil); err != nil {
		panic(err)
	}

	// now for the fun bit. When we ask for the state of the log, it will
	// play our opsets forward and get us the current state of the log
	// we can also get the state of a log from the book:
	log, err := book.Items(ctx, ref, 0, 100)
	if err != nil {
		panic(err)
	}

	for _, info := range log {
		fmt.Println(info.SimpleRef().String())
	}

	// Output:
	// yolanda_the_rat/world_bank_population@QmHashOfVersion3
	// yolanda_the_rat/world_bank_population@QmHashOfVersion1
}

func TestNewJournal(t *testing.T) {
	p := *testProfile(t)
	fs := qfs.NewMemFS()

	if _, err := logbook.NewJournal(p, nil, nil, "/mem/logbook.qfb"); err == nil {
		t.Errorf("expected missing private key arg to error")
	}
	if _, err := logbook.NewJournal(p, nil, nil, "/mem/logbook.qfb"); err == nil {
		t.Errorf("expected missing filesystem arg to error")
	}
	if _, err := logbook.NewJournal(p, nil, fs, ""); err == nil {
		t.Errorf("expected missing location arg to error")
	}
	if _, err := logbook.NewJournal(p, nil, fs, ""); err == nil {
		t.Errorf("expected nil event bus to error")
	}

	_, err := logbook.NewJournal(p, event.NilBus, fs, "/mem/logbook.qfb")
	if err != nil {
		t.Fatal(err)
	}
}

func TestNilCallable(t *testing.T) {
	var (
		book   *logbook.Book
		initID = ""
		ctx    = context.Background()
		err    error
	)

	if err = book.MergeLog(ctx, nil, &oplog.Log{}); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if err = book.RemoveLog(ctx, dsref.Ref{}); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if err = book.ConstructDatasetLog(ctx, nil, dsref.Ref{}, nil); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if err = book.WriteAuthorRename(ctx, nil, ""); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if _, err = book.WriteDatasetInit(ctx, nil, ""); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if err = book.WriteDatasetRename(ctx, nil, initID, ""); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if err = book.WriteDatasetDeleteAll(ctx, nil, initID); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if _, _, err = book.WriteRemotePush(ctx, nil, initID, 0, ""); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if _, _, err = book.WriteRemoteDelete(ctx, nil, initID, 0, ""); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if err = book.WriteVersionAmend(ctx, nil, nil); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if err = book.WriteVersionDelete(ctx, nil, initID, 0); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if err = book.WriteVersionSave(ctx, nil, nil, nil); err != logbook.ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", logbook.ErrNoLogbook, err)
	}
	if _, err = book.ResolveRef(ctx, nil); err != dsref.ErrRefNotFound {
		t.Errorf("expected '%s', got: %v", dsref.ErrRefNotFound, err)
	}
}

func TestResolveRef(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	if _, err := (*logbook.Book)(nil).ResolveRef(tr.Ctx, nil); err != dsref.ErrRefNotFound {
		t.Errorf("book ResolveRef must be nil-callable. expected: %q, got %v", dsref.ErrRefNotFound, err)
	}

	book := tr.Book
	dsrefspec.AssertResolverSpec(t, book, func(ref dsref.Ref, author *profile.Profile, log *oplog.Log) error {
		return book.MergeLog(tr.Ctx, author.PrivKey.GetPublic(), log)
	})
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
		"12:03AM\ttest_author\tpublish\t",
		"12:04AM\ttest_author\tunpublish\t",
		"12:00AM\ttest_author\tremove commit\t",
		"12:00AM\ttest_author\tamend commit\tadded meta info",
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestUserDatasetBranchesLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)
	tr.WriteRenameExample(t)

	if _, err := tr.Book.UserDatasetBranchesLog(tr.Ctx, ""); err == nil {
		t.Error("expected LogBytes with empty ref to fail")
	}

	if _, err := tr.Book.UserDatasetBranchesLog(tr.Ctx, tr.renameInitID); err != nil {
		t.Error(err)
	}

	got, err := tr.Book.UserDatasetBranchesLog(tr.Ctx, tr.worldBankInitID)
	if err != nil {
		t.Fatal(err)
	}

	justWorldBank := logbook.NewPlainLog(got)
	expect := tr.WorldBankPlainLog()

	if diff := cmp.Diff(expect, justWorldBank); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestLogBytes(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteRenameExample(t)
	log, err := tr.Book.UserDatasetBranchesLog(tr.Ctx, tr.RenameRef().InitID)
	if err != nil {
		t.Error(err)
	}
	data, err := tr.Book.LogBytes(log, tr.Owner.PrivKey)
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
	egDatasetInitID := tr.RenameRef().InitID
	log, err := tr.Book.UserDatasetBranchesLog(tr.Ctx, egDatasetInitID)
	if err != nil {
		t.Error(err)
	}

	if _, err := logbook.DsrefAliasForLog(nil); err == nil {
		t.Error("expected nil ref to error")
	}

	wrongModelLog, err := tr.Book.UserDatasetBranchesLog(tr.Ctx, egDatasetInitID)
	if err != nil {
		t.Fatal(err)
	}
	// use dataset oplog instead of user, which is wrong
	wrongModelLog = wrongModelLog.Logs[0]

	if _, err := logbook.DsrefAliasForLog(wrongModelLog); err == nil {
		t.Error("expected converting log of wrong model to error")
	}

	// TODO(b5) - not sure this is a proper test of ambiguity
	ambiguousLog, err := tr.Book.UserDatasetBranchesLog(tr.Ctx, egDatasetInitID)
	if err != nil {
		t.Fatal(err)
	}
	ambiguousLog = ambiguousLog.Logs[0]

	if _, err := logbook.DsrefAliasForLog(ambiguousLog); err == nil {
		t.Error("expected converting ambiguous logs to error")
	}

	ref, err := logbook.DsrefAliasForLog(log)
	if err != nil {
		t.Error(err)
	}

	expect := dsref.Ref{
		Username:  tr.RenameRef().Username,
		Name:      tr.RenameRef().Name,
		ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
	}

	if diff := cmp.Diff(expect, ref); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}

func TestWritePermissions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tr, cleanup := newTestRunner(t)
	defer cleanup()

	otherLogbook := tr.foreignLogbook(t, "janelle")

	initID, log := GenerateExampleOplog(ctx, t, otherLogbook, "atmospheric_particulates", "/ipld/QmExample")

	if err := tr.Book.MergeLog(ctx, otherLogbook.Owner().PubKey, log); err != nil {
		t.Fatal(err)
	}

	author := tr.Owner
	if err := tr.Book.WriteDatasetRename(ctx, author, initID, "foo"); !errors.Is(err, logbook.ErrAccessDenied) {
		t.Errorf("WriteDatasetRename to an oplog the book author doesn't own must return a wrap of logbook.ErrAccessDenied")
	}
	if err := tr.Book.WriteDatasetDeleteAll(ctx, author, initID); !errors.Is(err, logbook.ErrAccessDenied) {
		t.Errorf("WriteDatasetDeleteAll to an oplog the book author doesn't own must return a wrap of logbook.ErrAccessDenied")
	}

	ds := &dataset.Dataset{
		ID:       initID,
		Peername: author.Peername,
		Name:     "atmospheric_particulates",
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path: "HashOfVersion1",
	}
	if err := tr.Book.WriteVersionSave(ctx, author, ds, nil); !errors.Is(err, logbook.ErrAccessDenied) {
		t.Errorf("WriteVersionSave to an oplog the book author doesn't own must return a wrap of logbook.ErrAccessDenied")
	}
	if err := tr.Book.WriteVersionAmend(ctx, author, ds); !errors.Is(err, logbook.ErrAccessDenied) {
		t.Errorf("WriteVersionAmend to an oplog the book author doesn't own must return a wrap of logbook.ErrAccessDenied")
	}
	if err := tr.Book.WriteVersionDelete(ctx, author, initID, 1); !errors.Is(err, logbook.ErrAccessDenied) {
		t.Errorf("WriteVersionDelete to an oplog the book author doesn't own must return a wrap of logbook.ErrAccessDenied")
	}
	if _, _, err := tr.Book.WriteRemotePush(ctx, author, initID, 1, "https://registry.example.com"); !errors.Is(err, logbook.ErrAccessDenied) {
		t.Errorf("WriteRemotePush to an oplog the book author doesn't own must return a wrap of logbook.ErrAccessDenied")
	}
	if _, _, err := tr.Book.WriteRemoteDelete(ctx, author, initID, 1, "https://registry.example.com"); !errors.Is(err, logbook.ErrAccessDenied) {
		t.Errorf("WriteRemoteDelete to an oplog the book author doesn't own must return a wrap of logbook.ErrAccessDenied")
	}
}

func TestPushModel(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(tr.Ctx)
	defer cancel()

	author := tr.Owner

	initID, err := tr.Book.WriteDatasetInit(ctx, author, "publish_test")
	if err != nil {
		t.Fatal(err)
	}

	// TODO (b5) - we should have a check like this:
	// if _, _, err := tr.Book.WriteRemotePush(ctx, initID, 1, "example/remote/address"); err == nil {
	// 	t.Error("expected writing a push with no available versions to fail, got none")
	// }

	err = tr.Book.WriteVersionSave(ctx, author, &dataset.Dataset{
		ID:       initID,
		Peername: author.Peername,
		Name:     "atmospheric_particulates",
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path: "HashOfVersion1",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	lg, rollback, err := tr.Book.WriteRemotePush(ctx, author, initID, 1, "example/remote/address")
	if err != nil {
		t.Errorf("error writing push: %q", err)
	}

	if len(lg.Logs[0].Logs[0].Ops) != 3 {
		t.Errorf("expected branch log to have 3 operations. got: %d", len(lg.Logs[0].Logs[0].Ops))
	}

	t.Log(tr.Book.SummaryString(ctx) + "\n\n")

	if err = rollback(ctx); err != nil {
		t.Errorf("rollback error: %q", err)
	}
	if err = rollback(ctx); err != nil {
		t.Errorf("rollback error: %q", err)
	}

	t.Log(tr.Book.SummaryString(ctx))

	lg, err = tr.Book.UserDatasetBranchesLog(ctx, initID)
	if err != nil {
		t.Fatal(err)
	}

	if len(lg.Logs[0].Logs[0].Ops) != 2 {
		t.Errorf("expected branch log to have 2 operations after rollback. got: %d", len(lg.Logs[0].Logs[0].Ops))
	}

	_, _, err = tr.Book.WriteRemotePush(ctx, author, initID, 1, "example/remote/address")
	if err != nil {
		t.Errorf("error writing push: %q", err)
	}

	lg, rollback, err = tr.Book.WriteRemoteDelete(ctx, author, initID, 1, "example/remote/address")
	if err != nil {
		t.Errorf("error writing delete: %q", err)
	}

	if len(lg.Logs[0].Logs[0].Ops) != 4 {
		t.Errorf("expected branch log to have 4 operations after writing push & delete push. got: %d", len(lg.Logs[0].Logs[0].Ops))
	}
	if err := rollback(ctx); err != nil {
		t.Errorf("rollback error: %q", err)
	}
	if err = rollback(ctx); err != nil {
		t.Errorf("extra calls to rollback should not error. got: %q", err)
	}

	lg, err = tr.Book.UserDatasetBranchesLog(ctx, initID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lg.Logs[0].Logs[0].Ops) != 3 {
		t.Errorf("expected branch log to have 3 operations after writing push & delete push. got: %d", len(lg.Logs[0].Logs[0].Ops))
	}

}

func TestDatasetLogNaming(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()
	var err error
	author := tr.Owner

	if _, err = tr.Book.WriteDatasetInit(tr.Ctx, author, ""); err == nil {
		t.Errorf("expected initializing with an empty name to error")
	}
	firstInitID, err := tr.Book.WriteDatasetInit(tr.Ctx, author, "airport_codes")
	if err != nil {
		t.Fatalf("unexpected error writing valid dataset name: %s", err)
	}

	if err = tr.Book.WriteDatasetRename(tr.Ctx, author, firstInitID, "iata_airport_codes"); err != nil {
		t.Errorf("unexpected error renaming dataset: %s", err)
	}
	if _, err = tr.Book.RefToInitID(dsref.Ref{Username: "test_peer_dataset_log_naming", Name: "airport_codes"}); err == nil {
		t.Error("expected finding the original name to error")
	}
	// Init another dataset with the old name, which is now available due to rename.
	if _, err = tr.Book.WriteDatasetInit(tr.Ctx, author, "airport_codes"); err != nil {
		t.Fatalf("unexpected error writing recently freed-up dataset name: %s", err)
	}
	if err = tr.Book.WriteDatasetDeleteAll(tr.Ctx, author, firstInitID); err != nil {
		t.Errorf("unexpected error deleting first dataset: %s", err)
	}
	_, err = tr.Book.WriteDatasetInit(tr.Ctx, author, "iata_airport_codes")
	if err != nil {
		t.Errorf("expected initializing new name with deleted dataset to not error: %s", err)
	}

	const (
		profileID = "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"
		authorID  = "tz7ffwfj6e6z2xvdqgh2pf6gjkza5nzlncbjrj54s5s5eh46ma3q"
	)

	expect := []logbook.PlainLog{
		{
			Ops: []logbook.PlainOp{
				{Type: "init", Model: "user", Name: "test_author", AuthorID: profileID, Timestamp: mustTime("1999-12-31T19:00:00-05:00")},
			},
			Logs: []logbook.PlainLog{
				{
					Ops: []logbook.PlainOp{
						{Type: "init", Model: "dataset", Name: "airport_codes", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:01:00-05:00")},
						{Type: "amend", Model: "dataset", Name: "iata_airport_codes", Timestamp: mustTime("1999-12-31T19:03:00-05:00")},
						{Type: "remove", Model: "dataset", Timestamp: mustTime("1999-12-31T19:06:00-05:00")},
					},
					Logs: []logbook.PlainLog{
						{
							Ops: []logbook.PlainOp{
								{Type: "init", Model: "branch", Name: "main", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:02:00-05:00")},
							},
						},
					},
				},
				{
					Ops: []logbook.PlainOp{
						{Type: "init", Model: "dataset", Name: "airport_codes", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:04:00-05:00")},
					},
					Logs: []logbook.PlainLog{
						{
							Ops: []logbook.PlainOp{
								{Type: "init", Model: "branch", Name: "main", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:05:00-05:00")},
							},
						},
					},
				},
				{
					Ops: []logbook.PlainOp{
						{Type: "init", Model: "dataset", Name: "iata_airport_codes", AuthorID: authorID, Timestamp: mustTime("1999-12-31T19:07:00-05:00")},
					},
					Logs: []logbook.PlainLog{
						{
							Ops: []logbook.PlainOp{
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

	if _, err = tr.Book.WriteDatasetInit(tr.Ctx, author, "overwrite"); err != nil {
		t.Fatalf("unexpected error writing valid dataset name: %s", err)
	}
	if _, err = tr.Book.WriteDatasetInit(tr.Ctx, author, "overwrite"); err != nil {
		t.Fatalf("unexpected error overwrite an empty dataset history: %s", err)
	}
	err = tr.Book.WriteVersionSave(tr.Ctx, author, &dataset.Dataset{
		ID:       firstInitID,
		Peername: author.Peername,
		Name:     "atmospheric_particulates",
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path: "HashOfVersion1",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = tr.Book.WriteDatasetInit(tr.Ctx, author, "overwrite"); err != nil {
		t.Error("expected initializing a name that exists with a history to error")
	}
}

func TestBookPlainLogs(t *testing.T) {
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

	expect := []logbook.PlainLog{
		tr.WorldBankPlainLog(),
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

	log, err := tr.Book.UserDatasetBranchesLog(tr.Ctx, tr.WorldBankRef().InitID)
	if err != nil {
		t.Error(err)
	}

	if len(log.Logs) != 1 {
		t.Errorf("expected UserDatasetRef to only return one dataset log. got: %d", len(log.Logs))
	}

	pk2 := testPrivKey2(t)
	pro2 := mustProfileFromPrivKey("user_2", pk2)
	fs2 := qfs.NewMemFS()
	book2, err := logbook.NewJournal(*pro2, tr.bus, fs2, "/mem/fs2_location.qfb")
	if err != nil {
		t.Fatal(err)
	}

	if err := book2.MergeLog(tr.Ctx, tr.Book.Owner().PubKey, log); err == nil {
		t.Error("expected Merging unsigned log to fail")
	}

	if err := log.Sign(tr.Book.Owner().PrivKey); err != nil {
		t.Error(err)
	}

	if err := book2.MergeLog(tr.Ctx, tr.Book.Owner().PubKey, log); err != nil {
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

// Test a particularly tricky situation: a user authored and pushed a dataset to a remote. Then,
// they reinitialize their repository with the same profileID. This creates a new logbook entry,
// thus they have the same profileID but a different userCreateID. Then they push again to the
// same remote. Test that a client is able to pull this new dataset, and it will merge into their
// logbook, instead of creating two entries for the same user.
func TestMergeWithDivergentLogbookAuthorID(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()
	_ = tr

	ctx := context.Background()

	ref := dsref.MustParse("test_user/first_ds")
	firstKeyData := testkeys.GetKeyData(0)
	firstProfile := mustProfileFromPrivKey("test_user", firstKeyData.PrivKey)
	firstBook := makeLogbookOneCommit(ctx, t, firstProfile, ref, "first commit", "QmHashOfVersion1")

	ref = dsref.MustParse("test_user/second_ds")
	// NOTE: Purposefully use the same crypto key pairs. This will lead to the same
	// profileID, but different logbook userCreateIDs.
	secondKeyData := testkeys.GetKeyData(0)
	secondProfile := mustProfileFromPrivKey("test_user", secondKeyData.PrivKey)
	secondBook := makeLogbookOneCommit(ctx, t, secondProfile, ref, "second commit", "QmHashOfVersion2")

	// Get the log for the newly pushed dataset by initID.
	secondInitID, err := secondBook.RefToInitID(dsref.MustParse("test_user/second_ds"))
	if err != nil {
		t.Fatal(err)
	}
	secondLog, err := secondBook.UserDatasetBranchesLog(ctx, secondInitID)
	if err != nil {
		t.Error(err)
	}
	if len(secondLog.Logs) != 1 {
		t.Errorf("expected UserDatasetRef to only return one dataset log. got: %d", len(secondLog.Logs))
	}

	if err := secondLog.Sign(secondProfile.PrivKey); err != nil {
		t.Error(err)
	}

	if err := firstBook.MergeLog(ctx, secondBook.Owner().PubKey, secondLog); err != nil {
		t.Fatal(err)
	}

	revs, err := firstBook.PlainLogs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(revs)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)

	// Regex that replaces timestamps with just static text
	fixTs := regexp.MustCompile(`"(timestamp|commitTime)":\s?"[0-9TZ.:+-]*?"`)
	actual := string(fixTs.ReplaceAll([]byte(output), []byte(`"timestamp":"timeStampHere"`)))
	expect := `[{"ops":[{"type":"init","model":"user","name":"test_user","authorID":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B","timestamp":"timeStampHere"}],"logs":[{"ops":[{"type":"init","model":"dataset","name":"first_ds","authorID":"ftl4xgy5pvhfehd4h5wo5wggbac3m5dfywvp2rcohb5ayzgg6gja","timestamp":"timeStampHere"}],"logs":[{"ops":[{"type":"init","model":"branch","name":"main","authorID":"ftl4xgy5pvhfehd4h5wo5wggbac3m5dfywvp2rcohb5ayzgg6gja","timestamp":"timeStampHere"},{"type":"init","model":"commit","ref":"QmHashOfVersion1","timestamp":"timeStampHere","note":"first commit"}]}]},{"ops":[{"type":"init","model":"dataset","name":"second_ds","authorID":"i2smhmm5qrkf242wycim34ffvw4tjoxopk5bwbhleecbn4ojh4aq","timestamp":"timeStampHere"}],"logs":[{"ops":[{"type":"init","model":"branch","name":"main","authorID":"i2smhmm5qrkf242wycim34ffvw4tjoxopk5bwbhleecbn4ojh4aq","timestamp":"timeStampHere"},{"type":"init","model":"commit","ref":"QmHashOfVersion2","timestamp":"timeStampHere","note":"second commit"}]}]}]}]`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
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

	author := tr.Owner
	rename := "changed_username"
	if err := tr.Book.WriteAuthorRename(tr.Ctx, author, rename); err != nil {
		t.Fatalf("error renaming author: %s", err)
	}

	if rename != tr.Book.Owner().Peername {
		t.Errorf("authorname mismatch. expected: %s, got: %s", rename, tr.Book.Owner().Peername)
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

	expect := []dsref.VersionInfo{
		{
			Username:    "test_author",
			Name:        "world_bank_population",
			Path:        "QmHashOfVersion5",
			CommitTime:  mustTime("2000-01-04T19:00:00-05:00"),
			CommitTitle: "v5",
		},
		{
			Username:    "test_author",
			Name:        "world_bank_population",
			Path:        "QmHashOfVersion4",
			CommitTime:  mustTime("2000-01-03T19:00:00-05:00"),
			CommitTitle: "v4",
		},
		{
			Username:    "test_author",
			Name:        "world_bank_population",
			Path:        "QmHashOfVersion3",
			CommitTime:  mustTime("2000-01-02T19:00:00-05:00"),
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

	expect = []dsref.VersionInfo{
		{
			Username:    "test_author",
			Name:        "world_bank_population",
			Path:        "QmHashOfVersion4",
			CommitTime:  mustTime("2000-01-03T19:00:00-05:00"),
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

	username := tr.Owner.Peername

	book := tr.Book
	name := "to_reconstruct"
	ref := dsref.Ref{Username: username, Name: name}
	history := []*dataset.Dataset{
		{
			Peername: username,
			Name:     name,
			Commit: &dataset.Commit{
				Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Title:     "initial commit",
			},
			Path: "HashOfVersion1",
		},
		{
			Peername: username,
			Name:     name,
			Commit: &dataset.Commit{
				Timestamp: time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
				Title:     "commit 2",
			},
			Path:         "HashOfVersion2",
			PreviousPath: "HashOfVersion1",
		},
		{
			Peername: username,
			Name:     name,
			Commit: &dataset.Commit{
				Timestamp: time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
				Title:     "commit 2",
			},
			Path:         "HashOfVersion3",
			PreviousPath: "HashOfVersion2",
		},
	}

	if err := book.ConstructDatasetLog(tr.Ctx, tr.Owner, ref, history); err != nil {
		t.Errorf("error constructing history: %s", err)
	}

	if err := book.ConstructDatasetLog(tr.Ctx, tr.Owner, ref, history); err == nil {
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

func mustProfileFromPrivKey(username string, pk crypto.PrivKey) *profile.Profile {
	p, err := profile.NewSparsePKProfile(username, pk)
	if err != nil {
		panic(err)
	}
	return p
}

type testRunner struct {
	Ctx   context.Context
	bus   event.Bus
	Owner *profile.Profile
	Book  *logbook.Book
	Fs    qfs.Filesystem
	Tick  int

	renameInitID    string
	worldBankInitID string
}

func newTestRunner(t *testing.T) (tr *testRunner, cleanup func()) {
	ctx := context.Background()
	fs := qfs.NewMemFS()
	prevTs := logbook.NewTimestamp
	tr = &testRunner{
		Ctx:   ctx,
		bus:   event.NewBus(ctx),
		Owner: testProfile(t),
	}
	logbook.NewTimestamp = tr.newTimestamp

	var err error
	tr.Book, err = logbook.NewJournal(*tr.Owner, tr.bus, fs, "/mem/logbook.qfb")
	if err != nil {
		t.Fatalf("creating book: %s", err.Error())
	}

	cleanup = func() {
		logbook.NewTimestamp = prevTs
	}

	return tr, cleanup
}

func (tr *testRunner) newTimestamp() int64 {
	defer func() { tr.Tick++ }()
	t := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	return t.Add(time.Minute * time.Duration(tr.Tick)).UnixNano()
}

func (tr *testRunner) WorldBankRef() dsref.Ref {
	return dsref.Ref{Username: tr.Owner.Peername, Name: "world_bank_population", InitID: tr.worldBankInitID}
}

func (tr *testRunner) WorldBankID() string {
	return "crwd4wku64be6uxu3wbfqj7z65vtps4jt5ayx5dpjq4e2k72ks7q"
}

func (tr *testRunner) WriteWorldBankExample(t *testing.T) string {
	book := tr.Book
	name := "world_bank_population"

	initID, err := book.WriteDatasetInit(tr.Ctx, tr.Owner, name)
	if err != nil {
		panic(err)
	}
	tr.worldBankInitID = initID

	// pretend we've just created a dataset, these are the only fields the log
	// will care about
	ds := &dataset.Dataset{
		ID:       initID,
		Peername: tr.Owner.Peername,
		Name:     name,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path:         "QmHashOfVersion1",
		PreviousPath: "",
	}

	if err := book.WriteVersionSave(tr.Ctx, tr.Owner, ds, nil); err != nil {
		panic(err)
	}

	// sometime later, we create another version
	ds.Commit = &dataset.Commit{
		Timestamp: time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
		Title:     "added body data",
	}
	ds.Path = "QmHashOfVersion2"
	ds.PreviousPath = "QmHashOfVersion1"

	if err := book.WriteVersionSave(tr.Ctx, tr.Owner, ds, nil); err != nil {
		t.Fatal(err)
	}

	if _, _, err := book.WriteRemotePush(tr.Ctx, tr.Owner, initID, 2, "registry.qri.cloud"); err != nil {
		t.Fatal(err)
	}

	if _, _, err := book.WriteRemoteDelete(tr.Ctx, tr.Owner, initID, 2, "registry.qri.cloud"); err != nil {
		t.Fatal(err)
	}

	book.WriteVersionDelete(tr.Ctx, tr.Owner, initID, 1)

	ds.Commit.Timestamp = time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)
	ds.Commit.Title = "added meta info"
	ds.Path = "QmHashOfVersion3"
	ds.PreviousPath = "QmHashOfVersion1"

	if err := book.WriteVersionAmend(tr.Ctx, tr.Owner, ds); err != nil {
		t.Fatal(err)
	}

	return initID
}

func (tr *testRunner) WriteMoreWorldBankCommits(t *testing.T, initID string) {
	book := tr.Book
	name := "world_bank_population"
	ds := &dataset.Dataset{
		ID:       initID,
		Peername: tr.Owner.Peername,
		Name:     name,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC),
			Title:     "v4",
		},
		Path:         "QmHashOfVersion4",
		PreviousPath: "QmHashOfVersion3",
	}

	if err := book.WriteVersionSave(tr.Ctx, tr.Owner, ds, nil); err != nil {
		panic(err)
	}

	ds = &dataset.Dataset{
		ID:       initID,
		Peername: tr.Owner.Peername,
		Name:     name,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 5, 0, 0, 0, 0, time.UTC),
			Title:     "v5",
		},
		Path:         "QmHashOfVersion5",
		PreviousPath: "QmHashOfVersion4",
	}

	if err := book.WriteVersionSave(tr.Ctx, tr.Owner, ds, nil); err != nil {
		panic(err)
	}
}

func (tr *testRunner) RenameInitialRef() dsref.Ref {
	return dsref.Ref{Username: tr.Book.Owner().Peername, Name: "dataset", InitID: tr.renameInitID}
}

func (tr *testRunner) RenameRef() dsref.Ref {
	return dsref.Ref{Username: tr.Book.Owner().Peername, Name: "renamed_dataset", InitID: tr.renameInitID}
}

func (tr *testRunner) WriteRenameExample(t *testing.T) {
	book := tr.Book
	name := "dataset"
	rename := "renamed_dataset"

	initID, err := book.WriteDatasetInit(tr.Ctx, tr.Owner, name)
	if err != nil {
		panic(err)
	}
	tr.renameInitID = initID

	// pretend we've just created a dataset, these are the only fields the log
	// will care about
	ds := &dataset.Dataset{
		ID:       initID,
		Peername: tr.Owner.Peername,
		Name:     name,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path:         "QmHashOfVersion1",
		PreviousPath: "",
	}

	if err := book.WriteVersionSave(tr.Ctx, tr.Owner, ds, nil); err != nil {
		panic(err)
	}

	ds.Commit.Timestamp = time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)
	ds.Commit.Title = "added meta info"
	ds.Path = "QmHashOfVersion2"
	ds.PreviousPath = "QmHashOfVersion1"

	if err := book.WriteVersionSave(tr.Ctx, tr.Owner, ds, nil); err != nil {
		t.Fatal(err)
	}

	if err := book.WriteDatasetRename(tr.Ctx, tr.Owner, initID, rename); err != nil {
		t.Fatal(err)
	}
}

func testPrivKey(t *testing.T) crypto.PrivKey {
	return testkeys.GetKeyData(10).PrivKey
}

func testProfile(t *testing.T) *profile.Profile {
	return mustProfileFromPrivKey("test_author", testPrivKey(t))
}

func testPrivKey2(t *testing.T) crypto.PrivKey {
	return testkeys.GetKeyData(9).PrivKey
}

// ForeignLogbook creates a logbook to use as an external source of oplog data
func (tr *testRunner) foreignLogbook(t *testing.T, username string) *logbook.Book {
	t.Helper()

	ms := qfs.NewMemFS()
	pk := testPrivKey2(t)
	pro := mustProfileFromPrivKey(username, pk)
	journal, err := logbook.NewJournal(*pro, event.NilBus, ms, "/mem/logbook.qfb")
	if err != nil {
		t.Fatal(err)
	}

	return journal
}

func (tr *testRunner) WorldBankPlainLog() logbook.PlainLog {
	return logbook.PlainLog{
		Ops: []logbook.PlainOp{
			{
				Type:      "init",
				Model:     "user",
				Name:      "test_author",
				AuthorID:  "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
				Timestamp: mustTime("1999-12-31T19:00:00-05:00"),
			},
		},
		Logs: []logbook.PlainLog{
			{
				Ops: []logbook.PlainOp{
					{
						Type:      "init",
						Model:     "dataset",
						Name:      "world_bank_population",
						AuthorID:  "tz7ffwfj6e6z2xvdqgh2pf6gjkza5nzlncbjrj54s5s5eh46ma3q",
						Timestamp: mustTime("1999-12-31T19:01:00-05:00"),
					},
				},
				Logs: []logbook.PlainLog{
					{
						Ops: []logbook.PlainOp{
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
								Model: "push",
								Relations: []string{
									"registry.qri.cloud",
								},
								Timestamp: mustTime("1999-12-31T19:03:00-05:00"),
								Size:      2,
							},
							{
								Type:      "remove",
								Model:     "push",
								Relations: []string{"registry.qri.cloud"},
								Timestamp: mustTime("1999-12-31T19:04:00-05:00"),
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
	}
}

// GenerateExampleOplog makes an example dataset history on a given journal,
// returning the initID and a signed log
func GenerateExampleOplog(ctx context.Context, t *testing.T, journal *logbook.Book, dsname, headPath string) (string, *oplog.Log) {
	author := journal.Owner()
	initID, err := journal.WriteDatasetInit(ctx, author, dsname)
	if err != nil {
		t.Fatal(err)
	}

	err = journal.WriteVersionSave(ctx, author, &dataset.Dataset{
		ID:       initID,
		Peername: author.Peername,
		Name:     dsname,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path:         headPath,
		PreviousPath: "",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// TODO (b5) - we need UserDatasetRef here b/c it returns the full hierarchy
	// of oplogs. This method should take an InitID
	lg, err := journal.UserDatasetBranchesLog(ctx, initID)
	if err != nil {
		t.Fatal(err)
	}

	if err := lg.Sign(author.PrivKey); err != nil {
		t.Fatal(err)
		return "", nil
	}

	return initID, lg
}

func makeLogbookOneCommit(ctx context.Context, t *testing.T, pro *profile.Profile, ref dsref.Ref, commitMessage, dsPath string) *logbook.Book {
	rootPath, err := ioutil.TempDir("", "create_logbook")
	if err != nil {
		t.Fatal(err)
	}
	fs := qfs.NewMemFS()

	builder := logbook.NewLogbookTempBuilder(t, pro, fs, rootPath)
	id := builder.DatasetInit(ctx, t, ref.Name)
	builder.Commit(ctx, t, id, commitMessage, dsPath)
	return builder.Logbook()
}
