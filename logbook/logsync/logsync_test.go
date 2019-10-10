package logsync

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/log"
)

func ExampleNew() {
	// first some boilerplate setup
	ctx, done := context.WithCancel(context.Background())
	defer done()

	// our example has two authors. Johnathon and Basit are going to sync logbooks
	// let's start with two empty logbooks
	johnathonsLogbook := makeJohnathonLogbook()
	basitsLogbook := makeBasitLogbook()

	wait := make(chan struct{}, 1)

	// create a logsync from basit's logbook:
	basitLogsync := New(basitsLogbook, func(o *Options) {
		// we MUST override the PreCheck function. In this example we're only going
		// to allow pushes from johnathon
		o.ReceiveCheck = func(ctx context.Context, author log.Author, path string) error {
			if author.AuthorID() != johnathonsLogbook.Author().AuthorID() {
				return fmt.Errorf("rejected for secret reasons")
			}
			return nil
		}

		o.DidReceive = func(ctx context.Context, author log.Author, path string) error {
			wait <- struct{}{}
			return nil
		}
	})

	// for this example we're going to do sync over HTTP.
	// create an HTTP handler for the remote & wire it up to an example server
	handleFunc := HTTPHandler(basitLogsync)
	server := httptest.NewServer(handleFunc)
	defer server.Close()

	// johnathon creates a dataset with a bunch of history:
	worldBankDatasetRef := makeWorldBankLogs(ctx, johnathonsLogbook)

	ver, err := johnathonsLogbook.Versions(worldBankDatasetRef, 0, 100)
	if err != nil {
		panic(err)
	}
	fmt.Printf("johnathon has %d references for %s\n", len(ver), worldBankDatasetRef)

	// johnathon creates a new push
	johnathonLogsync := New(johnathonsLogbook)
	push, err := johnathonLogsync.NewPush(worldBankDatasetRef, server.URL)
	if err != nil {
		panic(err)
	}

	// execute the push, sending jonathon's world bank reference to basit
	if err = push.Do(ctx); err != nil {
		panic(err)
	}

	// wait for sync to complete
	<-wait
	if ver, err = basitsLogbook.Versions(worldBankDatasetRef, 0, 100); err != nil {
		panic(err)
	}
	fmt.Printf("basit has %d references for %s\n", len(ver), worldBankDatasetRef)

	// this time basit creates a history
	nasdaqDatasetRef := makeNasdaqLogs(ctx, basitsLogbook)

	if ver, err = basitsLogbook.Versions(nasdaqDatasetRef, 0, 100); err != nil {
		panic(err)
	}
	fmt.Printf("basit has %d references for %s\n", len(ver), nasdaqDatasetRef)

	// prepare to pull nasdaq refs from basit
	pull, err := johnathonLogsync.NewPull(nasdaqDatasetRef, server.URL)
	if err != nil {
		panic(err)
	}

	if err = pull.Do(ctx); err != nil {
		panic(err)
	}

	if ver, err = johnathonsLogbook.Versions(nasdaqDatasetRef, 0, 100); err != nil {
		panic(err)
	}
	fmt.Printf("johnathon has %d references for %s\n", len(ver), nasdaqDatasetRef)

	// Output: johnathon has 3 references for johnathon/world_bank_population
	// basit has 3 references for johnathon/world_bank_population
	// basit has 2 references for basit/nasdaq
	// johnathon has 2 references for basit/nasdaq
}

func makeJohnathonLogbook() *logbook.Book {
	book, err := newTestbook("johnathon", aPk)
	if err != nil {
		panic(err)
	}
	return book
}

func makeBasitLogbook() *logbook.Book {
	book, err := newTestbook("basit", bPk)
	if err != nil {
		panic(err)
	}
	return book
}

func makeWorldBankLogs(ctx context.Context, book *logbook.Book) dsref.Ref {
	ref, err := writeWorldBankLogs(ctx, book)
	if err != nil {
		panic(err)
	}
	return ref
}

func makeNasdaqLogs(ctx context.Context, book *logbook.Book) dsref.Ref {
	ref, err := writeNasdaqLogs(ctx, book)
	if err != nil {
		panic(err)
	}
	return ref
}

func TestSyncHTTP(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	a, b := tr.DefaultLogsyncs()

	server := httptest.NewServer(HTTPHandler(a))
	defer server.Close()

	ref, err := writeNasdaqLogs(tr.Ctx, tr.A)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := b.NewPull(ref, ""); err == nil {
		t.Errorf("expected invalid remote address to error")
	}

	pull, err := b.NewPull(ref, server.URL)
	if err != nil {
		t.Errorf("creating pull: %s", err)
	}

	if err := pull.Do(tr.Ctx); err != nil {
		t.Fatalf("pulling nasdaq logs %s", err.Error())
	}

	var expect, got []logbook.DatasetInfo
	if expect, err = tr.A.Versions(ref, 0, 100); err != nil {
		t.Error(err)
	}
	if got, err = tr.B.Versions(ref, 0, 100); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}

	worldBankRef, err := writeWorldBankLogs(tr.Ctx, tr.B)
	if err != nil {
		t.Fatal(err)
	}

	push, err := b.NewPush(worldBankRef, server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if err = push.Do(tr.Ctx); err != nil {
		t.Error(err)
	}

	if expect, err = tr.B.Versions(worldBankRef, 0, 100); err != nil {
		t.Error(err)
	}
	if got, err = tr.A.Versions(worldBankRef, 0, 100); err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}

	if err = b.DoRemove(tr.Ctx, ref, server.URL); err == nil {
		t.Errorf("expected an error removing a log user doesn't own")
	}

	if err = b.DoRemove(tr.Ctx, worldBankRef, server.URL); err != nil {
		t.Errorf("delete err: %s", err)
	}

	if got, err = tr.A.Versions(worldBankRef, 0, 100); err == nil {
		t.Logf("%v\n", got)
		t.Error("expected an err fetching removed reference")
	}
}

type testRunner struct {
	Ctx  context.Context
	A, B *logbook.Book
}

func (tr *testRunner) DefaultLogsyncs() (a, b *Logsync) {
	return New(tr.A), New(tr.B)
}

func newTestRunner(t *testing.T) (tr *testRunner, cleanup func()) {
	var err error
	tr = &testRunner{
		Ctx: context.Background(),
	}
	if tr.A, err = newTestbook("a", aPk); err != nil {
		t.Fatal(err)
	}
	if tr.B, err = newTestbook("b", bPk); err != nil {
		t.Fatal(err)
	}

	cleanup = func() {}
	return tr, cleanup
}

func newTestbook(username, b64pk string) (*logbook.Book, error) {
	// logbooks are encrypted at rest, we need a private key to interact with
	// them, including to create a new logbook. This is a dummy Private Key
	// you should never, ever use in real life. demo only folks.
	data, err := base64.StdEncoding.DecodeString(b64pk)
	if err != nil {
		return nil, err
	}
	pk, err := crypto.UnmarshalPrivateKey(data)
	if err != nil {
		return nil, err
	}

	// logbook relies on a qfs.Filesystem for read & write. create an in-memory
	// filesystem we can play with
	fs := qfs.NewMemFS()

	return logbook.NewBook(pk, username, fs, "/mem/logset")
}

func writeNasdaqLogs(ctx context.Context, book *logbook.Book) (ref dsref.Ref, err error) {
	name := "nasdaq"
	if err = book.WriteNameInit(ctx, name); err != nil {
		return ref, err
	}

	ds := &dataset.Dataset{
		Peername: book.AuthorName(),
		Name:     name,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
			Title:     "init dataset",
		},
		Path:         "v0",
		PreviousPath: "",
	}

	if err = book.WriteVersionSave(ctx, ds); err != nil {
		return ref, err
	}

	ds.Path = "v1"
	ds.PreviousPath = "v0"

	if err = book.WriteVersionSave(ctx, ds); err != nil {
		return ref, err
	}

	return dsref.Ref{
		Username: book.AuthorName(),
		Name:     name,
	}, nil
}

func writeWorldBankLogs(ctx context.Context, book *logbook.Book) (ref dsref.Ref, err error) {
	name := "world_bank_population"
	if err = book.WriteNameInit(ctx, name); err != nil {
		return ref, err
	}

	ds := &dataset.Dataset{
		Peername: book.AuthorName(),
		Name:     name,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
			Title:     "init dataset",
		},
		Path:         "v0",
		PreviousPath: "",
	}

	if err = book.WriteVersionSave(ctx, ds); err != nil {
		return ref, err
	}

	ds.Path = "v1"
	ds.PreviousPath = "v0"

	if err = book.WriteVersionSave(ctx, ds); err != nil {
		return ref, err
	}

	ds.Path = "v2"
	ds.PreviousPath = "v1"

	if err = book.WriteVersionSave(ctx, ds); err != nil {
		return ref, err
	}

	return dsref.Ref{
		Username: book.AuthorName(),
		Name:     name,
	}, nil
}

var aPk = `CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`

var bPk = "CAASpwkwggSjAgEAAoIBAQDACiqtbAeIR0gKZZfWuNgDssXnQnEQNrAlISlNMrtULuCtsLBk2tZ4C508T4/JQHfvbazZ/aPvkhr9KBaH8AzDU3FngHQnWblGtfm/0FAXbXPfn6DZ1rbA9rx9XpVZ+pUBDve0YxTSPOo5wOOR9u30JEvO47n1R/bF+wtMRHvDyRuoy4H86XxwMR76LYbgSlJm6SSKnrAVoWR9zqjXdaF1QljO77VbivnR5aS9vQ5Sd1mktwgb3SYUMlEGedtcMdLd3MPVCLFzq6cdjhSwVAxZ3RowR2m0hSEE/p6CKH9xz4wkMmjVrADfQTYU7spym1NBaNCrW1f+r4ScDEqI1yynAgMBAAECggEAWuJ04C5IQk654XHDMnO4h8eLsa7YI3w+UNQo38gqr+SfoJQGZzTKW3XjrC9bNTu1hzK4o1JOy4qyCy11vE/3Olm7SeiZECZ+cOCemhDUVsIOHL9HONFNHHWpLwwcUsEs05tpz400xWrezwZirSnX47tpxTgxQcwVFg2Bg07F5BntepqX+Ns7s2XTEc7YO8o77viYbpfPSjrsToahWP7ngIL4ymDjrZjgWTPZC7AzobDbhjTh5XuVKh60eUz0O7/Ezj2QK00NNkkD7nplU0tojZF10qXKCbECPn3pocVPAetTkwB1Zabq2tC2Y10dYlef0B2fkktJ4PAJyMszx4toQQKBgQD+69aoMf3Wcbw1Z1e9IcOutArrnSi9N0lVD7X2B6HHQGbHkuVyEXR10/8u4HVtbM850ZQjVnSTa4i9XJAy98FWwNS4zFh3OWVhgp/hXIetIlZF72GEi/yVFBhFMcKvXEpO/orEXMOJRdLb/7kNpMvl4MQ/fGWOmQ3InkKxLZFJ+wKBgQDA2jUTvSjjFVtOJBYVuTkfO1DKRGu7QQqNeF978ZEoU0b887kPu2yzx9pK0PzjPffpfUsa9myDSu7rncBX1FP0gNmSIAUja2pwMvJDU2VmE3Ua30Z1gVG1enCdl5ZWufum8Q+0AUqVkBdhPxw+XDJStA95FUArJzeZ2MTwbZH0RQKBgDG188og1Ys36qfPW0C6kNpEqcyAfS1I1rgLtEQiAN5GJMTOVIgF91vy11Rg2QVZrp9ryyOI/HqzAZtLraMCxWURfWn8D1RQkQCO5HaiAKM2ivRgVffvBHZd0M3NglWH/cWhxZW9MTRXtWLJX2DVvh0504s9yuAf4Jw6oG7EoAx5AoGBAJluAURO/jSMTTQB6cAmuJdsbX4+qSc1O9wJpI3LRp06hAPDM7ycdIMjwTw8wLVaG96bXCF7ZCGggCzcOKanupOP34kuCGiBkRDqt2tw8f8gA875S+k4lXU4kFgQvf8JwHi02LVxQZF0LeWkfCfw2eiKcLT4fzDV5ppzp1tREQmxAoGAGOXFomnMU9WPxJp6oaw1ZimtOXcAGHzKwiwQiw7AAWbQ+8RrL2AQHq0hD0eeacOAKsh89OXsdS9iW9GQ1mFR3FA7Kp5srjCMKNMgNSBNIb49iiG9O6P6UcO+RbYGg3CkSTG33W8l2pFIjBrtGktF5GoJudAPR4RXhVsRYZMiGag="
