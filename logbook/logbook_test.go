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

	// Create a new LogBook, passing in:
	//  * the author private key to encrypt & decrypt the logbook
	//  * author's current username
	//  * a qfs.Filesystem for reading & writing the logbook
	//  * a base path on the filesystem to read & write the logbook to
	// Initializing a logbook ensures the author has an user opset that matches
	// their current state. It will error if a stored book can't be decrypted
	book, err := NewBook(pk, "b5", fs, "/mem/logset")
	if err != nil {
		panic(err) // real programs don't panic
	}

	// create a name to store dataset versions in. NameInit will create a new
	// log under the logbook author's namespace with the given name, and an opset
	// that tracks operations by this author within that new namespace.
	// The entire logbook is persisted to the filestore after each operation
	if err := book.WriteNameInit(ctx, "world_bank_population"); err != nil {
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
	if err := book.WriteVersionSave(ctx, ds); err != nil {
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
	if err := book.WriteVersionSave(ctx, ds2); err != nil {
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
	if err := book.WritePublish(ctx, ref, 2, "registry.qri.cloud"); err != nil {
		panic(err)
	}

	// pretend the user just deleted a dataset version, well, we need to log it!
	// VersionDelete accepts an argument of number of versions back from HEAD
	// more complex deletes that remove pieces of history may require either
	// composing multiple log operations
	book.WriteVersionDelete(ctx, ref, 1)

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
	if err := book.WriteVersionSave(ctx, ds3); err != nil {
		panic(err)
	}

	// now for the fun bit. When we ask for the state of the log, it will
	// play our opsets forward and get us the current state of tne log
	// we can also get the state of a log from the book:
	log, err := book.Versions(ref, 0, 100)
	if err != nil {
		panic(err)
	}

	for _, info := range log {
		fmt.Println(info.Ref.String())
	}

	// Output:
	// b5/world_bank_population@QmHashOfVersion3
	// b5/world_bank_population@QmHashOfVersion1
}

func TestNewBook(t *testing.T) {
	pk := testPrivKey(t)
	fs := qfs.NewMemFS()

	if _, err := NewBook(nil, "b5", nil, "/mem/logset"); err == nil {
		t.Errorf("expected missing private key arg to error")
	}
	if _, err := NewBook(pk, "", nil, "/mem/logset"); err == nil {
		t.Errorf("expected missing author arg to error")
	}
	if _, err := NewBook(pk, "b5", nil, "/mem/logset"); err == nil {
		t.Errorf("expected missing filesystem arg to error")
	}
	if _, err := NewBook(pk, "b5", fs, ""); err == nil {
		t.Errorf("expected missing location arg to error")
	}

	_, err := NewBook(pk, "b5", fs, "/mem/logset")
	if err != nil {
		t.Fatal(err)
	}
}

func TestErrNoLogbook(t *testing.T) {
	var (
		book *Book
		ctx  = context.Background()
		err  error
	)

	if err = book.WriteCronJobRan(ctx, 0, dsref.Ref{}); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteNameAmend(ctx, dsref.Ref{}, ""); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteNameInit(ctx, ""); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WritePublish(ctx, dsref.Ref{}, 0); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteUnpublish(ctx, dsref.Ref{}, 0); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteVersionAmend(ctx, nil); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteVersionDelete(ctx, dsref.Ref{}, 0); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
	}
	if err = book.WriteVersionSave(ctx, nil); err != ErrNoLogbook {
		t.Errorf("expected '%s', got: %v", ErrNoLogbook, err)
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
		"10:07PM\ttest_author\tinit\t",
		"12:00AM\ttest_author\tsave\tinitial commit",
		"12:00AM\ttest_author\tsave\tadded body data",
		"12:00AM\ttest_author\tpublish\t",
		"12:00AM\ttest_author\tunpublish\t",
		"12:00AM\ttest_author\tremove\t",
		"12:00AM\ttest_author\tamend\tadded meta info",
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteRenameExample(t)

	if _, err := tr.Book.Log(dsref.Ref{}); err == nil {
		t.Error("expected LogBytes with empty ref to fail")
	}
	if _, err := tr.Book.Log(dsref.Ref{Username: tr.Username}); err == nil {
		t.Error("expected LogBytes with empty name ref to fail")
	}
	if _, err := tr.Book.Log(tr.RenameRef()); err != nil {
		t.Errorf("expected LogBytes with proper ref to not produce an error. got: %s", err)
	}
}

func TestLogBytes(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteRenameExample(t)
	log, err := tr.Book.Log(tr.RenameRef())
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
	log, err := tr.Book.Log(egRef)
	if err != nil {
		t.Error(err)
	}

	if _, err := DsrefAliasForLog(nil); err == nil {
		t.Error("expected nil ref to error")
	}

	wrongModelLog, err := tr.Book.bk.Log(userModel, tr.Username)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DsrefAliasForLog(wrongModelLog); err == nil {
		t.Error("expected converting log of wrong model to error")
	}

	ambiguousLog, err := tr.Book.bk.Log(nameModel, tr.Username)
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

func TestBookRawLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)

	got := tr.Book.RawLogs(tr.Ctx)

	// data, err := json.MarshalIndent(got, "", "  ")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// t.Logf("%s", string(data))

	expect := map[string][]Log{
		"name": []Log{
			{
				Ops: []Op{
					{
						Type:      "init",
						Model:     "name",
						Name:      "test_author",
						AuthorID:  "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
						Timestamp: mustTime("2047-03-18T17:07:12.45224192-05:00"),
					},
				},
				Logs: []Log{
					{
						Ops: []Op{
							{
								Type:      "init",
								Model:     "name",
								Name:      "world_bank_population",
								AuthorID:  "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
								Timestamp: mustTime("2047-03-18T17:07:13.45224192-05:00"),
							},
							{
								Type:      "init",
								Model:     "version",
								Ref:       "QmHashOfVersion1",
								Timestamp: mustTime("1999-12-31T19:00:00-05:00"),
								Note:      "initial commit",
							},
							{
								Type:      "init",
								Model:     "version",
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
								Model:     "version",
								Timestamp: mustTime("1969-12-31T19:00:00-05:00"),
								Size:      1,
							},
							{
								Type:      "amend",
								Model:     "version",
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
		"user": []Log{
			{
				Ops: []Op{
					{
						Type:      "init",
						Model:     "user",
						Name:      "test_author",
						AuthorID:  "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
						Timestamp: mustTime("2047-03-18T17:07:11.45224192-05:00"),
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

// TODO (b5) - this test should also check that only the requested log is being
// transferred, not any extras
func TestLogTransfer(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)

	log, err := tr.Book.Log(tr.WorldBankRef())
	if err != nil {
		t.Error(err)
	}

	pk2 := testPrivKey2(t)
	fs2 := qfs.NewMemFS()
	book2, err := NewBook(pk2, "user2", fs2, "/mem/fs2_location")
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

	revs, err := book2.Versions(tr.WorldBankRef(), 0, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) == 0 {
		t.Errorf("expected book 2 to now have versions for world bank ref")
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
		"10:07PM\ttest_author\tinit\t",
		"12:00AM\ttest_author\tsave\tinitial commit",
		"12:00AM\ttest_author\tran update\t",
		"12:00AM\ttest_author\tsave\tadded meta info",
		"10:07PM\ttest_author\trename\t",
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestVersions(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)
	tr.WriteMoreWorldBankCommits(t)
	book := tr.Book

	versions, err := book.Versions(tr.WorldBankRef(), 0, 10)
	if err != nil {
		t.Error(err)
	}

	expect := []DatasetInfo{
		{
			Ref: dsref.Ref{
				Username: "test_author",
				Name:     "world_bank_population",
				Path:     "QmHashOfVersion5",
			},
			Timestamp:   mustTime("2000-01-04T19:00:00-05:00"),
			CommitTitle: "v5",
		},
		{
			Ref: dsref.Ref{
				Username: "test_author",
				Name:     "world_bank_population",
				Path:     "QmHashOfVersion4",
			},
			Timestamp:   mustTime("2000-01-03T19:00:00-05:00"),
			CommitTitle: "v4",
		},
		{
			Ref: dsref.Ref{
				Username: "test_author",
				Name:     "world_bank_population",
				Path:     "QmHashOfVersion3",
			},
			Timestamp:   mustTime("2000-01-02T19:00:00-05:00"),
			CommitTitle: "added meta info",
		},
	}

	if diff := cmp.Diff(expect, versions); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}

	versions, err = book.Versions(tr.WorldBankRef(), 1, 1)
	if err != nil {
		t.Error(err)
	}

	expect = []DatasetInfo{
		{
			Ref: dsref.Ref{
				Username: "test_author",
				Name:     "world_bank_population",
				Path:     "QmHashOfVersion4",
			},
			Timestamp:   mustTime("2000-01-03T19:00:00-05:00"),
			CommitTitle: "v4",
		},
	}
	if diff := cmp.Diff(expect, versions); diff != "" {
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
	versions, err := book.Versions(ref, 0, 100)
	if err != nil {
		t.Errorf("getting versions: %s", err)
	}

	if len(versions) != 3 {
		t.Errorf("expected 3 versions to return from history")
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
	tr.Book, err = NewBook(pk, authorName, fs, "/mem/logset")
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
	return time.Unix(int64(94670280000+tr.Tick), 0).UnixNano()
}

func (tr *testRunner) WorldBankRef() dsref.Ref {
	return dsref.Ref{Username: tr.Username, Name: "world_bank_population"}
}

func (tr *testRunner) WriteWorldBankExample(t *testing.T) {
	book := tr.Book
	name := "world_bank_population"

	if err := book.WriteNameInit(tr.Ctx, name); err != nil {
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

	if err := book.WriteVersionSave(tr.Ctx, ds); err != nil {
		panic(err)
	}

	// sometime later, we create another version
	ds.Commit = &dataset.Commit{
		Timestamp: time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
		Title:     "added body data",
	}
	ds.Path = "QmHashOfVersion2"
	ds.PreviousPath = "QmHashOfVersion1"

	if err := book.WriteVersionSave(tr.Ctx, ds); err != nil {
		t.Fatal(err)
	}

	ref := dsref.Ref{Username: tr.Username, Name: name}
	if err := book.WritePublish(tr.Ctx, ref, 2, "registry.qri.cloud"); err != nil {
		t.Fatal(err)
	}

	if err := book.WriteUnpublish(tr.Ctx, ref, 2, "registry.qri.cloud"); err != nil {
		t.Fatal(err)
	}

	book.WriteVersionDelete(tr.Ctx, ref, 1)

	ds.Commit.Timestamp = time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)
	ds.Commit.Title = "added meta info"
	ds.Path = "QmHashOfVersion3"
	ds.PreviousPath = "QmHashOfVersion1"

	if err := book.WriteVersionAmend(tr.Ctx, ds); err != nil {
		t.Fatal(err)
	}
}

func (tr *testRunner) WriteMoreWorldBankCommits(t *testing.T) {
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

	if err := book.WriteVersionSave(tr.Ctx, ds); err != nil {
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

	if err := book.WriteVersionSave(tr.Ctx, ds); err != nil {
		panic(err)
	}
}

func (tr *testRunner) RenameInitialRef() dsref.Ref {
	return dsref.Ref{Username: tr.Book.bk.AuthorName(), Name: "dataset"}
}

func (tr *testRunner) RenameRef() dsref.Ref {
	return dsref.Ref{Username: tr.Book.bk.AuthorName(), Name: "renamed_dataset"}
}

func (tr *testRunner) WriteRenameExample(t *testing.T) {
	book := tr.Book
	name := "dataset"
	rename := "renamed_dataset"

	if err := book.WriteNameInit(tr.Ctx, name); err != nil {
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

	if err := book.WriteVersionSave(tr.Ctx, ds); err != nil {
		panic(err)
	}

	ds.Commit.Timestamp = time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)
	ds.Commit.Title = "added meta info"
	ds.Path = "QmHashOfVersion2"
	ds.PreviousPath = "QmHashOfVersion1"

	// pretend we ran a cron job that created this version
	ref := dsref.Ref{Username: book.bk.AuthorName(), Name: name}
	if err := book.WriteCronJobRan(tr.Ctx, 1, ref); err != nil {
		t.Fatal(err)
	}

	if err := book.WriteVersionSave(tr.Ctx, ds); err != nil {
		t.Fatal(err)
	}

	if err := book.WriteNameAmend(tr.Ctx, ref, rename); err != nil {
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
