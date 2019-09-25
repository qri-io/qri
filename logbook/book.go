package log

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/logbook/log"
	"github.com/qri-io/qri/repo"
)

var (
	// ErrNotFound is a sentinel error for data not found in a logbook
	ErrNotFound = fmt.Errorf("logbook: not found")
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
	bk       *log.Book
	location string
	fs       qfs.WritableFilesystem
}

// NewBook initializes a logbook, reading any existing data at the given
// location, on the given filesystem. logbooks are encrypted at rest. The
// same key must be given to decrypt an existing logbook
func NewBook(pk crypto.PrivKey, username string, fs qfs.WritableFilesystem, location string) (*Book, error) {
	pid, err := calcProfileID(pk)
	if err != nil {
		return nil, err
	}

	// validate inputs
	// check for an existing log
	bk, err := log.NewBook(pk, username, pid)
	if err != nil {
		return nil, err
	}

	book := &Book{
		bk:       bk,
		fs:       fs,
		location: location,
	}

	return book, book.Load(context.Background())
}

// RenameAuthor marks a change in author name
func (book Book) RenameAuthor() error {
	return fmt.Errorf("not finished")
}

// DeleteAuthor removes an author, we'll use this in key rotation
func (book Book) DeleteAuthor() error {
	return fmt.Errorf("not finished")
}

// Save writes the book to book.location
func (book Book) Save(ctx context.Context) (string, error) {
	ciphertext, err := book.bk.FlatbufferCipher()
	if err != nil {
		return "", err
	}

	file := qfs.NewMemfileBytes(book.location, ciphertext)
	return book.fs.Put(ctx, file)
}

// Load reads the book dataset from book.location
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

	return book.bk.UnmarshalFlatbufferCipher(ctx, ciphertext)
}

// WriteNameInit initializes a new name within the author's namespace. Dataset
// histories start with a NameInit
func (book Book) WriteNameInit(ctx context.Context, name string) error {
	op := log.Op{
		Type:      log.OpTypeInit,
		Model:     nameModel,
		AuthorID:  book.bk.AuthorID(),
		Name:      name,
		Timestamp: time.Now().UnixNano(),
	}

	ns := book.authorNamespace()
	ns.AddChild(log.InitLog(op))

	_, err := book.Save(ctx)
	return err
}

func (book Book) authorNamespace() *log.Log {
	for _, l := range book.bk.ModelLogs(nameModel) {
		if l.Name() == book.bk.AuthorName() {
			return l
		}
	}

	l := log.InitLog(log.Op{
		Type:      log.OpTypeInit,
		Model:     nameModel,
		Name:      book.bk.AuthorName(),
		AuthorID:  book.bk.AuthorID(),
		Timestamp: time.Now().UnixNano(),
	})

	book.bk.AppendLog(l)
	return l
}

// WriteVersionSave adds an operation to a log marking the creation of a
// dataset version. Book will copy details from the provided dataset pointer
func (book Book) WriteVersionSave(ctx context.Context, alias string, ds *dataset.Dataset) error {
	l, err := book.readAliasLog(nameModel, alias, book.bk.AuthorName())
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:  log.OpTypeInit,
		Model: versionModel,
		Ref:   ds.Path,
		Prev:  ds.PreviousPath,
		Note:  ds.Commit.Title,
	})

	_, err = book.Save(ctx)
	return err
}

// WriteVersionAmend adds an operation to a log amending a dataset version
func (book Book) WriteVersionAmend(ctx context.Context, alias string, ds *dataset.Dataset) error {
	l, err := book.readAliasLog(nameModel, alias, book.bk.AuthorName())
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:  log.OpTypeAmend,
		Model: versionModel,
		Ref:   ds.Path,
		Prev:  ds.PreviousPath,
		Note:  ds.Commit.Title,
	})

	_, err = book.Save(ctx)
	return err
}

// WriteVersionDelete adds an operation to a log marking a number of sequential
// versions from HEAD as deleted. Because logs are append-only, deletes are
// recorded as "tombstone" operations that mark removal.
func (book Book) WriteVersionDelete(ctx context.Context, alias string, revisions int) error {
	l, err := book.readAliasLog(nameModel, alias, book.bk.AuthorName())
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:  log.OpTypeRemove,
		Model: versionModel,
		// TODO (b5) - finish
	})

	_, err = book.Save(ctx)
	return err
}

// WritePublish adds an operation to a log marking the publication of a number
// of versions to one or more destinations
func (book Book) WritePublish(ctx context.Context, alias string, revisions int, destinations ...string) error {
	l, err := book.readAliasLog(nameModel, alias, book.bk.AuthorName())
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:  log.OpTypeInit,
		Model: publicationModel,
		// TODO (b5) - finish
	})

	_, err = book.Save(ctx)
	return err
}

// WriteUnpublish adds an operation to a log marking an unpublish request for a
// count of sequential versions from HEAD
func (book Book) WriteUnpublish(ctx context.Context, alias string, revisions int, destinations ...string) error {
	l, err := book.readAliasLog(nameModel, alias, book.bk.AuthorName())
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:  log.OpTypeRemove,
		Model: publicationModel,
		// TODO (b5) - finish
	})

	_, err = book.Save(ctx)
	return err
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

// func (book Book) readAliasLogRoot(model uint32, alias string) (*log.Log, error) {
// 	if alias == "" {
// 		return nil, fmt.Errorf("alias is required")
// 	}

// 	for _, lg := range book.bk.ModelLogs(model) {
// 		if lg.Name() == alias {
// 			return lg.Child(lg.RootName()), nil
// 		}
// 	}

// 	return nil, ErrNotFound
// }

func (book Book) readAliasLog(model uint32, alias, branch string) (*log.Log, error) {
	if alias == "" {
		return nil, fmt.Errorf("alias is required")
	}

	for _, lg := range book.bk.ModelLogs(model) {
		if lg.Name() == alias {
			log := lg.Child(branch)
			if log == nil {
				return nil, ErrNotFound
			}
			return log, nil
		}
	}

	return nil, ErrNotFound
}

func calcProfileID(privKey crypto.PrivKey) (string, error) {
	pubkeybytes, err := privKey.GetPublic().Bytes()
	if err != nil {
		return "", fmt.Errorf("error getting pubkey bytes: %s", err.Error())
	}

	mh, err := multihash.Sum(pubkeybytes, multihash.SHA2_256, 32)
	if err != nil {
		return "", fmt.Errorf("error summing pubkey: %s", err.Error())
	}

	return mh.B58String(), nil
}
