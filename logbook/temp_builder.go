package logbook

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/profile"
)

// BookBuilder builds a logbook in a convenient way
type BookBuilder struct {
	Book   *Book
	Owner  *profile.Profile
	Dsrefs map[string][]string
}

// NewLogbookTempBuilder constructs a logbook tmp BookBuilder
func NewLogbookTempBuilder(t *testing.T, owner *profile.Profile, fs qfs.Filesystem, fsPath string) BookBuilder {
	book, err := NewJournal(*owner, event.NilBus, fs, fsPath)
	if err != nil {
		t.Fatal(err)
	}
	builder := BookBuilder{
		Book:   book,
		Owner:  owner,
		Dsrefs: make(map[string][]string),
	}
	return builder
}

// DatasetInit initializes a new dataset and return a reference to it
func (b *BookBuilder) DatasetInit(ctx context.Context, t *testing.T, dsname string) string {
	author := b.Owner
	initID, err := b.Book.WriteDatasetInit(ctx, author, dsname)
	if err != nil {
		t.Fatal(err)
	}
	b.Dsrefs[dsname] = make([]string, 0)
	return initID
}

// DatasetRename changes the name of a dataset
func (b *BookBuilder) DatasetRename(ctx context.Context, t *testing.T, initID, newName string) dsref.Ref {
	author := b.Owner
	if err := b.Book.WriteDatasetRename(ctx, author, initID, newName); err != nil {
		t.Fatal(err)
	}
	ref := dsref.Ref{}
	return dsref.Ref{Username: author.Peername, Name: newName, Path: ref.Path}
}

// DatasetDelete deletes a dataset
func (b *BookBuilder) DatasetDelete(ctx context.Context, t *testing.T, initID string) {
	author := b.Owner
	if err := b.Book.WriteDatasetDeleteAll(ctx, author, initID); err != nil {
		t.Fatal(err)
	}
	ref := dsref.Ref{}
	delete(b.Dsrefs, ref.Name)
}

// AddForeign merges a foreign log into this book, signs using owner
func (b *BookBuilder) AddForeign(ctx context.Context, t *testing.T, log *oplog.Log) {
	log.Sign(b.Owner.PrivKey)
	if err := b.Book.MergeLog(ctx, b.Owner.PubKey, log); err != nil {
		t.Fatal(err)
	}
}

// Commit adds a commit to a dataset
func (b *BookBuilder) Commit(ctx context.Context, t *testing.T, initID, title, ipfsHash string) dsref.Ref {
	author := b.Owner
	ref := dsref.Ref{}
	ds := &dataset.Dataset{
		ID:       initID,
		Peername: ref.Username,
		Name:     ref.Name,
		Commit: &dataset.Commit{
			Timestamp: time.Unix(0, NewTimestamp()),
			Title:     title,
		},
		Path:         ipfsHash,
		PreviousPath: ref.Path,
	}
	if err := b.Book.WriteVersionSave(ctx, author, ds, nil); err != nil {
		t.Fatal(err)
	}
	b.Dsrefs[ref.Name] = append(b.Dsrefs[ref.Name], ipfsHash)
	return dsref.Ref{Username: ref.Username, Name: ref.Name, Path: ipfsHash}
}

// Delete removes some number of commits from a dataset
func (b *BookBuilder) Delete(ctx context.Context, t *testing.T, initID string, num int) dsref.Ref {
	author := b.Owner
	ref := dsref.Ref{}
	if err := b.Book.WriteVersionDelete(ctx, author, initID, num); err != nil {
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
