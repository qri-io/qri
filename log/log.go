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
	pk       crypto.PrivKey
	authors  []logset
	logs     []logset
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
	l := log{
		ops: []operation{o},
	}
	set := logset{
		root: name,
		logs: map[string]log{
			name: l,
		},
	}
	book.logs = append(book.logs, set)
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
	// TODO (b5) - finish
	return 0
}

func (book *Book) unmarshalFlatbuffer(b *logfb.Book) error {
	// TODO (b5) - finish
	return fmt.Errorf("not finished")
}

// log is a causally-ordered set of operations performed by a single author.
// log attribution is verified by an author's signature
type log struct {
	signature []byte
	ops       []operation
}

// Len returns the number of of the latest entry in the log
func (set log) Len() int {
	return len(set.ops)
}

func (set log) Type() string {
	return ""
}

func (set log) Author() (name, identifier string) {
	// TODO (b5) - name and identifier must come from init operation
	return "", ""
}

func (set log) Verify() error {
	return fmt.Errorf("not finished")
}

func (set log) MarshalFlatbuffer(builder *flatbuffers.Builder, name, identifier string) flatbuffers.UOffsetT {
	// count := len(set)
	// offsets := make([]flatbuffers.UOffsetT, count)
	// for i, o := range set {
	// 	offsets[i] = o.MarshalFlatbuffer(builder)
	// }

	// logfb.OpStartListVector(builder, count)
	// for i := count - 1; i >= 0; i-- {
	// 	builder.PrependUOffsetT(offsets[i])
	// }
	// return builder.EndVector(count)
	// TODO (b5)
	return 0
}

// logset is a collection of unique logs
type logset struct {
	signer    string
	signature []byte
	root      string
	logs      map[string]log
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

	count := len(ls.logs)
	offsets := make([]flatbuffers.UOffsetT, count)
	i := 0
	for _, log := range ls.logs {
		name, id := log.Author()
		offsets[i] = log.MarshalFlatbuffer(builder, name, id)
		i++
	}

	logfb.LogStartOpsetVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	ops := builder.EndVector(count)

	logfb.LogStart(builder)
	logfb.LogAddName(builder, name)
	logfb.LogAddIdentifier(builder, id)
	logfb.LogAddOpset(builder, ops)
	return logfb.LogEnd(builder)
}

func (ls logset) UnmarshalFlatbuffer(flatlog *logfb.Log) error {
	return nil
}

type logmap map[string]logset

func (l logmap) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	count := len(l)
	offsets := make([]flatbuffers.UOffsetT, count)
	i := 0
	for _, lg := range l {
		offsets[i] = lg.MarshalFlatbuffer(builder)
		i++
	}

	logfb.LogsetStartLogsVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	logs := builder.EndVector(count)

	logfb.LogsetStart(builder)
	logfb.LogsetAddLogs(builder, logs)
	return logfb.LogsetEnd(builder)
	return 0
}

func (l logmap) UnmarshalFlatbuffer(logset *logfb.Logset) error {
	return fmt.Errorf("not finished")
}
