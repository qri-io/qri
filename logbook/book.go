package log

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/logbook/log"
	"github.com/qri-io/qri/repo"
)

const (
	userModel        uint32 = 0x0001
	nameModel        uint32 = 0x0002
	versionModel     uint32 = 0x0003
	publicationModel uint32 = 0x0004
	aclModel         uint32 = 0x0005
)

// Book wraps a log.Book with a higher-order API specific to Qri
type Book struct {
	book     log.Book
	location string
	fs       qfs.Filesystem
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

// Save writes the
func (book Book) Save(ctx context.Context) error {
	ciphertext, err := book.book.(book.flatbufferBytes())
	if err != nil {
		return err
	}

	file := qfs.NewMemfileBytes(book.location, ciphertext)
	return book.fs.Put(ctx, file)
}

// Load
func (book Book) Load(ctx context.Context) error {
	f, err := book.fs.Get(ctx, book.location)
	if err != nil {
		if err == qfs.ErrNotFound {
			return nil
		}
		return err
	}

	ciphertext, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	plaintext, err := book.decrypt(ciphertext)
	if err != nil {
		return err
	}

	return
}

// NameInit initializes a new name within the author's namespace. Dataset
// histories start with a NameInit
func (book Book) NameInit(name string) error {
	// op := log.NewNameInit(book.id, book.username, name)
	op := log.Op{
		Type:      log.OpTypeInit,
		Model:     nameModel,
		AuthorID:  book.id,
		Name:      name,
		Timestamp: time.Now().UnixNano(),
	}

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
