// Package logbook records and syncs dataset histories. As users work on
// datasets, they build of a log of operations. Each operation is a record
// of an action taken, like creating a dataset, or unpublishing a version.
// Each of these operations is wrtten to a log attributed to the user that
// performed the action, and stored in the logbook under the namespace of that
// dataset. The current state of a user's log is derived from iterating over
// all operations to produce the current state.
package logbook

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook/log"
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
	pk       crypto.PrivKey
	location string
	fs       qfs.Filesystem
}

// NewBook initializes a logbook, reading any existing data at the given
// location, on the given filesystem. logbooks are encrypted at rest. The
// same key must be given to decrypt an existing logbook
func NewBook(pk crypto.PrivKey, username string, fs qfs.Filesystem, location string) (*Book, error) {
	ctx := context.Background()
	if pk == nil {
		return nil, fmt.Errorf("logbook: private key is required")
	}
	if fs == nil {
		return nil, fmt.Errorf("logbook: filsystem is required")
	}
	pid, err := calcProfileID(pk)
	if err != nil {
		return nil, err
	}

	// TODO (b5) - validate inputs

	bk, err := log.NewBook(pk, username, pid)
	if err != nil {
		return nil, err
	}

	book := &Book{
		bk:       bk,
		fs:       fs,
		pk:       pk,
		location: location,
	}

	if err = book.Load(ctx); err != nil {
		if err == ErrNotFound {
			err = book.initialize(ctx)
			return book, err
		}
		return nil, err
	}

	return book, nil
}

func (book *Book) initialize(ctx context.Context) error {
	// initialize author namespace
	l := log.InitLog(log.Op{
		Type:      log.OpTypeInit,
		Model:     nameModel,
		Name:      book.bk.AuthorName(),
		AuthorID:  book.bk.AuthorID(),
		Timestamp: time.Now().UnixNano(),
	})

	book.bk.AppendLog(l)

	_, err := book.Save(ctx)
	return err
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
			return ErrNotFound
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
	book.initName(ctx, name)
	_, err := book.Save(ctx)
	return err
}

func (book Book) initName(ctx context.Context, name string) *log.Log {
	lg := log.InitLog(log.Op{
		Type:      log.OpTypeInit,
		Model:     nameModel,
		AuthorID:  book.bk.AuthorID(),
		Name:      name,
		Timestamp: time.Now().UnixNano(),
	})

	ns := book.authorNamespace()
	ns.AddChild(lg)
	return lg
}

func (book Book) authorNamespace() *log.Log {
	for _, l := range book.bk.ModelLogs(nameModel) {
		if l.Name() == book.bk.AuthorName() {
			return l
		}
	}
	// this should never happen in practice
	return nil
}

// WriteVersionSave adds an operation to a log marking the creation of a
// dataset version. Book will copy details from the provided dataset pointer
func (book Book) WriteVersionSave(ctx context.Context, ref dsref.Ref, ds *dataset.Dataset) error {
	l, err := book.readRefLog(ref)
	if err != nil {
		if err == ErrNotFound {
			l = book.initName(ctx, ref.Name)
			err = nil
		} else {
			return err
		}
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
func (book Book) WriteVersionAmend(ctx context.Context, ref dsref.Ref, ds *dataset.Dataset) error {
	l, err := book.readRefLog(ref)
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
func (book Book) WriteVersionDelete(ctx context.Context, ref dsref.Ref, revisions int) error {
	l, err := book.readRefLog(ref)
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:  log.OpTypeRemove,
		Model: versionModel,
		Size:  uint64(revisions),
		// TODO (b5) - finish
	})

	_, err = book.Save(ctx)
	return err
}

// WritePublish adds an operation to a log marking the publication of a number
// of versions to one or more destinations
func (book Book) WritePublish(ctx context.Context, ref dsref.Ref, revisions int, destinations ...string) error {
	l, err := book.readRefLog(ref)
	if err != nil {
		return fmt.Errorf("%#v", book.bk.ModelLogs(nameModel)[0])
		// return err
	}

	l.Append(log.Op{
		Type:      log.OpTypeInit,
		Model:     publicationModel,
		Size:      uint64(revisions),
		Relations: destinations,
		// TODO (b5) - finish
	})

	_, err = book.Save(ctx)
	return err
}

// WriteUnpublish adds an operation to a log marking an unpublish request for a
// count of sequential versions from HEAD
func (book Book) WriteUnpublish(ctx context.Context, ref dsref.Ref, revisions int, destinations ...string) error {
	l, err := book.readRefLog(ref)
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:      log.OpTypeRemove,
		Model:     publicationModel,
		Size:      uint64(revisions),
		Relations: destinations,
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

// LogBytes gets signed bytes suitable for sending as a network request.
// keep in mind that logs should never be sent to someone who does not have
// proper permission to be disclosed log details
func (book Book) LogBytes(ref dsref.Ref) ([]byte, error) {
	lg, err := book.readRefLog(ref)
	if err != nil {
		return nil, err
	}

	return lg.SignedFlatbufferBytes(book.pk)
}

// Versions plays a set of operations for a given log, producing a State struct
// that describes the current state of a dataset
func (book Book) Versions(ref dsref.Ref, offset, limit int) ([]dsref.Info, error) {
	l, err := book.readRefLog(ref)
	if err != nil {
		return nil, err
	}

	refs := []dsref.Info{}
	for _, op := range l.Ops() {
		if op.Model == versionModel {
			switch op.Type {
			case log.OpTypeInit:
				refs = append(refs, book.infoFromOp(ref, op))
			case log.OpTypeAmend:
				refs[len(refs)-1] = book.infoFromOp(ref, op)
			case log.OpTypeRemove:
				refs = refs[:len(refs)-int(op.Size)]
			}
		}
	}

	return refs, nil
}

func (book Book) infoFromOp(ref dsref.Ref, op log.Op) dsref.Info {
	return dsref.Info{
		Ref: dsref.Ref{
			Username: ref.Username,
			Name:     ref.Name,
			Path:     op.Ref,
		},
		Timestamp:   time.Unix(op.Timestamp, op.Timestamp),
		CommitTitle: op.Note,
	}
}

// ACL represents an access control list. ACL is a work in progress, not fully
// implemented
// TODO (b5) - the real version of this struct will come from a different
// package
type ACL struct {
}

// ACL is a control list
func (book Book) ACL(alias string) (ACL, error) {
	return ACL{}, fmt.Errorf("not finished")
}

func (book Book) readRefLog(ref dsref.Ref) (*log.Log, error) {
	if ref.Username == "" {
		return nil, fmt.Errorf("ref.Peername is required")
	}
	if ref.Name == "" {
		return nil, fmt.Errorf("ref.Name is required")
	}

	for _, lg := range book.bk.ModelLogs(nameModel) {
		if lg.Name() == ref.Username {
			log := lg.Child(ref.Name)
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
