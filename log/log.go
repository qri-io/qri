package log

import (
	"fmt"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/log/logfb"
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
	authors  []*logset
	datasets []*logset
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
	now := time.Now().UnixNano()
	o := nameInit{
		op: op{
			timestamp: now,
		},
	}
	l := &log{
		ops: []operation{o},
	}
	set := &logset{
		root: name,
		logs: map[string]*log{
			name: l,
		},
	}
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

func (book Book) readAlias(alias string) (*log, error) {
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

	newBook.authors = make([]*logset, b.AuthorsLength())
	var logsetfb logfb.Logset
	for i := 0; i < b.AuthorsLength(); i++ {
		if b.Authors(&logsetfb, i) {
			newBook.authors[i] = &logset{}
			if err := newBook.authors[i].UnmarshalFlatbuffer(&logsetfb); err != nil {
				return err
			}
		}
	}

	newBook.datasets = make([]*logset, b.DatasetsLength())
	for i := 0; i < b.DatasetsLength(); i++ {
		if ok := b.Datasets(&logsetfb, i); ok {
			newBook.datasets[i] = &logset{}
			if err := newBook.datasets[i].UnmarshalFlatbuffer(&logsetfb); err != nil {
				return err
			}
		}
	}

	*book = newBook
	return nil
}

// logset is a collection of unique logs
type logset struct {
	signer    string
	signature []byte
	root      string
	logs      map[string]*log
}

func (ls logset) Author() (string, string) {
	// TODO (b5) - fetch from master branch intiailization
	return "", ""
}

// MarshalFlatbuffer
func (ls logset) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	namestr, idstr := ls.Author()
	name := builder.CreateString(namestr)
	id := builder.CreateString(idstr)
	root := builder.CreateString(ls.root)

	count := len(ls.logs)
	offsets := make([]flatbuffers.UOffsetT, count)
	i := 0
	for _, log := range ls.logs {
		offsets[i] = log.MarshalFlatbuffer(builder)
		i++
	}

	logfb.LogsetStartLogsVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	logs := builder.EndVector(count)

	logfb.LogsetStart(builder)
	logfb.LogsetAddName(builder, name)
	logfb.LogsetAddRoot(builder, root)
	logfb.LogsetAddIdentifier(builder, id)
	logfb.LogsetAddLogs(builder, logs)
	return logfb.LogEnd(builder)
}

func (ls *logset) UnmarshalFlatbuffer(lsfb *logfb.Logset) (err error) {
	newLs := logset{
		root: string(lsfb.Root()),
		logs: map[string]*log{},
	}

	lgfb := &logfb.Log{}
	for i := 0; i < lsfb.LogsLength(); i++ {
		if lsfb.Logs(lgfb, i) {
			lg := &log{}
			if err = lg.UnmarshalFlatbuffer(lgfb); err != nil {
				return err
			}
			newLs.logs[lg.Name()] = lg
		}
	}

	*ls = newLs
	return nil
}

// log is a causally-ordered set of operations performed by a single author.
// log attribution is verified by an author's signature
type log struct {
	signature []byte
	ops       []operation
}

// Len returns the number of of the latest entry in the log
func (lg log) Len() int {
	return len(lg.ops)
}

func (lg log) Type() string {
	return ""
}

func (lg log) Author() (name, identifier string) {
	// TODO (b5) - name and identifier must come from init operation
	if len(lg.ops) > 0 {
		if initOp, ok := lg.ops[0].(initOperation); ok {
			return initOp.AuthorName(), initOp.AuthorID()
		}
	}
	return "", ""
}

func (lg log) Name() string {
	if len(lg.ops) > 0 {
		if initOp, ok := lg.ops[0].(initOperation); ok {
			return initOp.Name()
		}
	}
	return ""
}

func (lg log) Verify() error {
	return fmt.Errorf("not finished")
}

func (lg log) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	namestr, idstr := lg.Author()
	name := builder.CreateString(namestr)
	id := builder.CreateString(idstr)
	signature := builder.CreateByteString(lg.signature)

	count := len(lg.ops)
	offsets := make([]flatbuffers.UOffsetT, count)
	for i, o := range lg.ops {
		offsets[i] = o.MarshalFlatbuffer(builder)
	}

	logfb.LogStartOpsetVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	ops := builder.EndVector(count)

	logfb.LogStart(builder)
	logfb.LogAddName(builder, name)
	logfb.LogAddIdentifier(builder, id)
	logfb.LogAddSignature(builder, signature)
	logfb.LogAddOpset(builder, ops)
	return logfb.LogEnd(builder)
}

func (lg *log) UnmarshalFlatbuffer(lfb *logfb.Log) (err error) {
	newLg := log{
		signature: lfb.Signature(),
	}

	newLg.ops = make([]operation, lfb.OpsetLength())
	opfb := &logfb.Operation{}
	for i := 0; i < lfb.OpsetLength(); i++ {
		if lfb.Opset(opfb, i) {
			if newLg.ops[i], err = unmarshalOperationFlatbuffer(opfb); err != nil {
				return err
			}
		}
	}

	*lg = newLg
	return nil
}
