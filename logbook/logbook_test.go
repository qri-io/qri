package logbook

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook/log"
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
	// b5/world_bank_population@QmHashOfVersion1
	// b5/world_bank_population@QmHashOfVersion3
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

	book, err := NewBook(pk, "b5", fs, "/mem/logset")
	if err != nil {
		t.Fatal(err)
	}

	if err := book.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBookLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)

	entries, err := tr.Book.Logs(tr.WorldBankRef(), 0, 30)
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

func TestBookRawLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)

	got := tr.Book.RawLogs()

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
								Timestamp: mustTime("1969-12-31T19:00:00-05:00"),
								Note:      "initial commit",
							},
							{
								Type:      "init",
								Model:     "version",
								Ref:       "QmHashOfVersion2",
								Prev:      "QmHashOfVersion1",
								Timestamp: mustTime("1969-12-31T19:00:00-05:00"),
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
								Timestamp: mustTime("1969-12-31T19:00:00-05:00"),
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

func TestLogTransfer(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteWorldBankExample(t)

	data, err := tr.Book.LogBytes(tr.WorldBankRef())
	if err != nil {
		t.Error(err)
	}

	got := &log.Log{}
	if err := got.UnmarshalFlatbufferBytes(data); err != nil {
		t.Error(err)
	}

	// TODO (b5) - create a second book & load it. Don't have an API for this yet
}

func TestRenameDataset(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	tr.WriteRenameExample(t)

	// TODO (b5) - finish:
	// if _, err := tr.Book.Logs(tr.RenameInitialRef(), 0, 30); err == nil {
	// 	t.Error("expected fetching renamed dataset to error")
	// }

	// entries, err := tr.Book.Logs(tr.RenameRef(), 0, 30)
	entries, err := tr.Book.Logs(tr.RenameInitialRef(), 0, 30)
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
	prevTs := newTimestamp
	tr = &testRunner{
		Ctx:      ctx,
		Username: authorName,
	}
	newTimestamp = tr.newTimestamp

	var err error
	tr.Book, err = NewBook(pk, authorName, fs, "/mem/logset")
	if err != nil {
		t.Fatalf("creating book: %s", err.Error())
	}

	cleanup = func() {
		newTimestamp = prevTs
	}

	return tr, cleanup
}

func (tr *testRunner) newTimestamp() time.Time {
	defer func() { tr.Tick++ }()
	return time.Unix(int64(94670280000+tr.Tick), 0)
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
