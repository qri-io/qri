package log

import (
	"fmt"

	flatbuffers "github.com/google/flatbuffers/go"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/logbook/log"
	"github.com/qri-io/qri/logbook/logfb"
	"github.com/qri-io/qri/repo"
)

// Book is a journal of operations organized into a collection of append-only
// logs. Each log is single-writer
// Books are connected to a single author, and represent their view of
// the global dataset graph.
// Any write operation performed on the logbook are attributed to a single
// author, denoted by a private key. Books can replicate logs from other
// authors, forming a conflict-free replicated data type (CRDT), and a basis
// for collaboration through knowledge of each other's operations
type Book struct {
	username string
	id       string
	pk       crypto.PrivKey
	authors  []*log.Set
	datasets []*log.Set
}

// NewBook initializes a logbook, reading any existing data at the given
// location, on the given filesystem. logbooks are encrypted at rest. The
// same key must be given to decrypt an existing logbook
func NewBook(pk crypto.PrivKey, username string, fs qfs.WritableFilesystem, location string) (*Book, error) {
	// validate inputs
	// check for an existing log
	return &Book{}, fmt.Errorf("not finished")
}

// RenameAuthor marks a change in author name
func (book Book) RenameAuthor() error {
	return fmt.Errorf("not finished")
}

// DeleteAuthor removes an author, we'll use this in key rotation
func (book Book) DeleteAuthor() error {
	return fmt.Errorf("not finished")
}

// NameInit initializes a new name within the author's namespace. Dataset
// histories start with a NameInit
func (book Book) NameInit(name string) error {
	op := log.NewNameInit(book.id, book.username, name)
	set := log.InitSet(name, op)
	book.datasets = append(book.datasets, set)
	return fmt.Errorf("not finished")
}

// VersionSave adds an operation to a log marking the creation of a dataset
// version. Book will copy details from the provided dataset pointer
func (book Book) VersionSave(alias string, ds *dataset.Dataset) error {
	return fmt.Errorf("not finished")
}

// VersionAmend adds an operation to a log amending a dataset version
func (book Book) VersionAmend(alias string, ds *dataset.Dataset) error {
	return fmt.Errorf("not finished")
}

// VersionDelete adds an operation to a log marking a number of sequential
// versions from HEAD as deleted. Because logs are append-only, deletes are
// recorded as "tombstone" operations that mark removal.
func (book Book) VersionDelete(alias string, revisions int) error {
	return fmt.Errorf("not finished")
}

// Publish adds an operation to a log marking the publication of a number of
// versions to one or more destinations. Versions count continously from head
// back
func (book Book) Publish(alias string, revisions int, destinations ...string) error {
	return fmt.Errorf("not finished")
}

// Unpublish adds an operation to a log marking an unpublish request for a count
// of sequential versions from HEAD
func (book Book) Unpublish(alias string, revisions int, destinations ...string) error {
	return fmt.Errorf("not finished")
}

// Author represents the author at a point in time
type Author struct {
	Username  string
	ID        string
	PublicKey string
}

// Author plays forward the current author's operation log to determine the
// latest author state
func (book Book) Author(username string) (Author, error) {
	a := Author{
		Username: "",
	}
	return a, nil
}

// Versions plays a set of operations for a given log, producing a State struct
// that describes the current state of a dataset
func (book Book) Versions(alias string, offset, limit int) ([]repo.DatasetRef, error) {
	return nil, fmt.Errorf("not finished")
}

// ACL represents an access control list
// TODO (b5) - the real version of this struct will come from a different
// package
type ACL struct {
}

// ACL is a control list
func (book Book) ACL(alias string) (ACL, error) {
	return ACL{}, fmt.Errorf("not finished")
}

func (book Book) readAlias(alias string) (*log.Log, error) {
	if alias == "" {
		return nil, fmt.Errorf("alias is required")
	}
	return nil, fmt.Errorf("not finished")
}

func (book Book) save() error {
	return fmt.Errorf("not finished")
}

// flatbufferBytes formats book as a flatbuffer byte slice
func (book Book) flatbufferBytes() []byte {
	builder := flatbuffers.NewBuilder(0)
	off := book.marshalFlatbuffer(builder)
	builder.Finish(off)
	return builder.FinishedBytes()
}

func (book Book) marshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	username := builder.CreateString(book.username)
	id := builder.CreateString(book.id)

	count := len(book.authors)
	offsets := make([]flatbuffers.UOffsetT, count)
	for i, lset := range book.authors {
		offsets[i] = lset.MarshalFlatbuffer(builder)
	}
	logfb.BookStartAuthorsVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	authors := builder.EndVector(count)

	count = len(book.datasets)
	offsets = make([]flatbuffers.UOffsetT, count)
	for i, lset := range book.datasets {
		offsets[i] = lset.MarshalFlatbuffer(builder)
	}
	logfb.BookStartAuthorsVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	datasets := builder.EndVector(count)

	logfb.BookStart(builder)
	logfb.BookAddName(builder, username)
	logfb.BookAddIdentifier(builder, id)
	logfb.BookAddAuthors(builder, authors)
	logfb.BookAddDatasets(builder, datasets)
	return logfb.BookEnd(builder)
}

func (book *Book) unmarshalFlatbuffer(b *logfb.Book) error {
	newBook := Book{}

	newBook.authors = make([]*log.Set, b.AuthorsLength())
	var logsetfb logfb.Logset
	for i := 0; i < b.AuthorsLength(); i++ {
		if b.Authors(&logsetfb, i) {
			newBook.authors[i] = &log.Set{}
			if err := newBook.authors[i].UnmarshalFlatbuffer(&logsetfb); err != nil {
				return err
			}
		}
	}

	newBook.datasets = make([]*log.Set, b.DatasetsLength())
	for i := 0; i < b.DatasetsLength(); i++ {
		if ok := b.Datasets(&logsetfb, i); ok {
			newBook.datasets[i] = &log.Set{}
			if err := newBook.datasets[i].UnmarshalFlatbuffer(&logsetfb); err != nil {
				return err
			}
		}
	}

	*book = newBook
	return nil
}
