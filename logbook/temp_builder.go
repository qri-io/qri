package logbook

import (
	"context"
	"testing"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook/oplog"
)

// BookBuilder builds a logbook in a convenient way
type BookBuilder struct {
	Book     *Book
	Username string
	Dsrefs   map[string][]string
}

// NewLogbookTempBuilder constructs a logbook tmp BookBuilder
func NewLogbookTempBuilder(t *testing.T, privKey crypto.PrivKey, username string, fs qfs.Filesystem, rootPath string) BookBuilder {
	// TODO (b5) - accept an event bus
	bus := event.NewBus(context.Background())
	book, err := NewJournal(privKey, username, bus, fs, rootPath)
	if err != nil {
		t.Fatal(err)
	}
	builder := BookBuilder{
		Book:     book,
		Username: username,
		Dsrefs:   make(map[string][]string),
	}
	return builder
}

// DatasetInit initializes a new dataset and return a reference to it
func (b *BookBuilder) DatasetInit(ctx context.Context, t *testing.T, dsname string) string {
	initID, err := b.Book.WriteDatasetInit(ctx, dsname)
	if err != nil {
		t.Fatal(err)
	}
	b.Dsrefs[dsname] = make([]string, 0)
	return initID
}

// DatasetRename changes the name of a dataset
func (b *BookBuilder) DatasetRename(ctx context.Context, t *testing.T, initID, newName string) dsref.Ref {
	if err := b.Book.WriteDatasetRename(ctx, initID, newName); err != nil {
		t.Fatal(err)
	}
	ref := dsref.Ref{}
	return dsref.Ref{Username: b.Username, Name: newName, Path: ref.Path}
}

// DatasetDelete deletes a dataset
func (b *BookBuilder) DatasetDelete(ctx context.Context, t *testing.T, initID string) {
	if err := b.Book.WriteDatasetDelete(ctx, initID); err != nil {
		t.Fatal(err)
	}
	ref := dsref.Ref{}
	delete(b.Dsrefs, ref.Name)
}

// AddForeign merges a foreign log into this book
func (b *BookBuilder) AddForeign(ctx context.Context, t *testing.T, log *oplog.Log) {
	log.Sign(b.Book.pk)
	if err := b.Book.MergeLog(ctx, b.Book.Author(), log); err != nil {
		t.Fatal(err)
	}
}

// Commit adds a commit to a dataset
func (b *BookBuilder) Commit(ctx context.Context, t *testing.T, initID, title, ipfsHash string) dsref.Ref {
	ref := dsref.Ref{}
	ds := dataset.Dataset{
		Peername: ref.Username,
		Name:     ref.Name,
		Commit: &dataset.Commit{
			Timestamp: time.Unix(0, NewTimestamp()),
			Title:     title,
		},
		Path:         ipfsHash,
		PreviousPath: ref.Path,
	}
	if err := b.Book.WriteVersionSave(ctx, initID, &ds, nil); err != nil {
		t.Fatal(err)
	}
	b.Dsrefs[ref.Name] = append(b.Dsrefs[ref.Name], ipfsHash)
	return dsref.Ref{Username: ref.Username, Name: ref.Name, Path: ipfsHash}
}

// Delete removes some number of commits from a dataset
func (b *BookBuilder) Delete(ctx context.Context, t *testing.T, initID string, num int) dsref.Ref {
	ref := dsref.Ref{}
	if err := b.Book.WriteVersionDelete(ctx, initID, num); err != nil {
		t.Fatal(err)
	}
	prevRefs := b.Dsrefs[ref.Name]
	nextRefs := prevRefs[:len(prevRefs)-num]
	b.Dsrefs[ref.Name] = nextRefs
	lastRef := nextRefs[len(nextRefs)-1]
	return dsref.Ref{Username: ref.Username, Name: ref.Name, Path: lastRef}
}

// Logbook returns the built logbook
func (b *BookBuilder) Logbook() *Book {
	return b.Book
}
