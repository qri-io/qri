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
	"strings"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook/log"
)

var (
	// ErrNoLogbook indicates a logbook doesn't exist
	ErrNoLogbook = fmt.Errorf("logbook: no logbook")
	// ErrNotFound is a sentinel error for data not found in a logbook
	ErrNotFound = fmt.Errorf("logbook: not found")
	// ErrLogTooShort indicates a log is missing elements. Because logs are
	// append-only, passing a shorter log than the one on file is grounds
	// for rejection
	ErrLogTooShort = fmt.Errorf("logbook: log is too short")

	newTimestamp = func() time.Time { return time.Now() }
)

const (
	userModel        uint32 = 0x0001
	nameModel        uint32 = 0x0002
	versionModel     uint32 = 0x0003
	publicationModel uint32 = 0x0004
	aclModel         uint32 = 0x0005
	cronJobModel     uint32 = 0x0006
)

func modelString(m uint32) string {
	switch m {
	case userModel:
		return "user"
	case nameModel:
		return "name"
	case versionModel:
		return "version"
	case publicationModel:
		return "publication"
	case aclModel:
		return "acl"
	case cronJobModel:
		return "cronJob"
	default:
		return ""
	}
}

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
		return nil, fmt.Errorf("logbook: filesystem is required")
	}
	if location == "" {
		return nil, fmt.Errorf("logbook: location is required")
	}
	pid, err := calcProfileID(pk)
	if err != nil {
		return nil, err
	}

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

	if err = book.load(ctx); err != nil {
		if err == ErrNotFound {
			err = book.initialize(ctx)
			return book, err
		}
		return nil, err
	}

	return book, nil
}

func (book *Book) initialize(ctx context.Context) error {
	// initialize author's log of user actions
	userActions := log.InitLog(log.Op{
		Type:      log.OpTypeInit,
		Model:     userModel,
		Name:      book.bk.AuthorName(),
		AuthorID:  book.bk.AuthorID(),
		Timestamp: newTimestamp().UnixNano(),
	})
	book.bk.AppendLog(userActions)

	// initialize author's namespace
	ns := log.InitLog(log.Op{
		Type:      log.OpTypeInit,
		Model:     nameModel,
		Name:      book.bk.AuthorName(),
		AuthorID:  book.bk.AuthorID(),
		Timestamp: newTimestamp().UnixNano(),
	})
	book.bk.AppendLog(ns)

	return book.save(ctx)
}

// Author returns this book's author
func (book *Book) Author() log.Author {
	return book.bk
}

// AuthorName returns the human-readable name of the author
func (book *Book) AuthorName() string {
	return book.bk.AuthorName()
}

// RenameAuthor marks a change in author name
func (book *Book) RenameAuthor() error {
	return fmt.Errorf("not finished")
}

// DeleteAuthor removes an author, used on teardown
func (book *Book) DeleteAuthor() error {
	return fmt.Errorf("not finished")
}

// save writes the book to book.location
func (book *Book) save(ctx context.Context) error {

	ciphertext, err := book.bk.FlatbufferCipher()
	if err != nil {
		return err
	}

	file := qfs.NewMemfileBytes(book.location, ciphertext)
	book.location, err = book.fs.Put(ctx, file)
	return err
}

// load reads the book dataset from book.location
func (book *Book) load(ctx context.Context) error {
	f, err := book.fs.Get(ctx, book.location)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
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
// TODO (b5) - this presently only works for datasets in an author's user
// namespace
func (book *Book) WriteNameInit(ctx context.Context, name string) error {
	if book == nil {
		return ErrNoLogbook
	}
	book.initName(ctx, name)
	return book.save(ctx)
}

func (book Book) initName(ctx context.Context, name string) *log.Log {
	lg := log.InitLog(log.Op{
		Type:      log.OpTypeInit,
		Model:     nameModel,
		AuthorID:  book.bk.AuthorID(),
		Name:      name,
		Timestamp: newTimestamp().UnixNano(),
	})

	ns := book.authorNamespace()
	ns.Logs = append(ns.Logs, lg)
	return lg
}

func (book Book) authorNamespace() *log.Log {
	for _, l := range book.bk.ModelLogs(nameModel) {
		if l.Name() == book.bk.AuthorName() {
			return l
		}
	}
	// this should never happen in practice
	// TODO (b5): create an author namespace on the spot if this happens
	return nil
}

// WriteNameAmend marks a rename event within a namespace
// TODO (b5) - finish
func (book *Book) WriteNameAmend(ctx context.Context, ref dsref.Ref, newName string) error {
	if book == nil {
		return ErrNoLogbook
	}

	l, err := book.readRefLog(ref)
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:      log.OpTypeAmend,
		Model:     nameModel,
		Name:      newName,
		Timestamp: newTimestamp().UnixNano(),
	})
	return nil
}

// WriteVersionSave adds an operation to a log marking the creation of a
// dataset version. Book will copy details from the provided dataset pointer
func (book *Book) WriteVersionSave(ctx context.Context, ds *dataset.Dataset) error {
	if book == nil {
		return ErrNoLogbook
	}

	ref := refFromDataset(ds)
	l, err := book.readRefLog(ref)
	if err != nil {
		if err == ErrNotFound {
			l = book.initName(ctx, ref.Name)
			err = nil
		} else {
			return err
		}
	}

	book.appendVersionSave(l, ds)
	return book.save(ctx)
}

func (book *Book) appendVersionSave(l *log.Log, ds *dataset.Dataset) {
	op := log.Op{
		Type:  log.OpTypeInit,
		Model: versionModel,
		Ref:   ds.Path,
		Prev:  ds.PreviousPath,

		Timestamp: ds.Commit.Timestamp.UnixNano(),
		Note:      ds.Commit.Title,
	}

	if ds.Structure != nil {
		op.Size = uint64(ds.Structure.Length)
	}

	l.Append(op)
}

// WriteVersionAmend adds an operation to a log amending a dataset version
func (book *Book) WriteVersionAmend(ctx context.Context, ds *dataset.Dataset) error {
	if book == nil {
		return ErrNoLogbook
	}

	l, err := book.readRefLog(refFromDataset(ds))
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:  log.OpTypeAmend,
		Model: versionModel,
		Ref:   ds.Path,
		Prev:  ds.PreviousPath,

		Timestamp: ds.Commit.Timestamp.UnixNano(),
		Note:      ds.Commit.Title,
	})

	return book.save(ctx)
}

// WriteVersionDelete adds an operation to a log marking a number of sequential
// versions from HEAD as deleted. Because logs are append-only, deletes are
// recorded as "tombstone" operations that mark removal.
func (book *Book) WriteVersionDelete(ctx context.Context, ref dsref.Ref, revisions int) error {
	if book == nil {
		return ErrNoLogbook
	}

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

	return book.save(ctx)
}

// WritePublish adds an operation to a log marking the publication of a number
// of versions to one or more destinations
func (book *Book) WritePublish(ctx context.Context, ref dsref.Ref, revisions int, destinations ...string) error {
	if book == nil {
		return ErrNoLogbook
	}

	l, err := book.readRefLog(ref)
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:      log.OpTypeInit,
		Model:     publicationModel,
		Size:      uint64(revisions),
		Relations: destinations,
		// TODO (b5) - finish
	})

	return book.save(ctx)
}

// WriteUnpublish adds an operation to a log marking an unpublish request for a
// count of sequential versions from HEAD
func (book *Book) WriteUnpublish(ctx context.Context, ref dsref.Ref, revisions int, destinations ...string) error {
	if book == nil {
		return ErrNoLogbook
	}

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

	return book.save(ctx)
}

// WriteCronJobRan adds an operation to a log marking the execution of a cronjob
func (book *Book) WriteCronJobRan(ctx context.Context, number int64, ref dsref.Ref) error {
	if book == nil {
		return ErrNoLogbook
	}

	l, err := book.readRefLog(ref)
	if err != nil {
		return err
	}

	l.Append(log.Op{
		Type:  log.OpTypeInit,
		Model: cronJobModel,
		Size:  uint64(number),
		// TODO (b5) - finish
	})

	return book.save(ctx)
}

// LogBytes gets signed bytes suitable for sending as a network request.
// keep in mind that logs should never be sent to someone who does not have
// proper permission to be disclosed log details
func (book Book) LogBytes(ref dsref.Ref) ([]byte, error) {
	if ref.Username == "" {
		return nil, fmt.Errorf("logbook: reference Username is required")
	}
	if ref.Name == "" {
		return nil, fmt.Errorf("logbook: reference Name is required")
	}

	for _, lg := range book.bk.ModelLogs(nameModel) {
		if lg.Name() == ref.Username {
			l := lg.Child(ref.Name)
			if l == nil {
				return nil, ErrNotFound
			}

			root := &log.Log{
				Ops:  lg.Ops,
				Logs: []*log.Log{l},
			}
			return root.SignedFlatbufferBytes(book.pk)
		}
	}

	return nil, ErrNotFound
}

// MergeLogBytes adds a log to the logbook, merging with any existing log data
func (book *Book) MergeLogBytes(ctx context.Context, sender log.Author, data []byte) error {
	if data == nil {
		return fmt.Errorf("no data provided to merge")
	}

	lg := &log.Log{}
	if err := lg.UnmarshalFlatbufferBytes(data); err != nil {
		return err
	}

	// eventually access control will dictate which logs can be written by whom.
	// For now we only allow users to merge logs they've written
	// book will need access to a store of public keys before we can verify
	// signatures non-same-senders
	if err := lg.Verify(sender.AuthorPubKey()); err != nil {
		return err
	}

	if lg.Author() != sender.AuthorID() {
		return fmt.Errorf("authors can only push logs they own")
	}

	merged := false
	for _, l := range book.bk.ModelLogs(nameModel) {
		// x.Model() == y.Model() && x.Ops[0].Name == y.Ops[0].Name && x.Ops[0].AuthorID == y.Ops[0].AuthorID
		if l.Model() == lg.Model() && l.Ops[0].Name == lg.Ops[0].Name && l.Ops[0].AuthorID == lg.Ops[0].AuthorID {
			merged = true
			l.Merge(lg)
			break
		}
	}

	if !merged {
		book.bk.AppendLog(lg)
	}

	return book.save(ctx)
}

// RemoveLog removes an entire log from a logbook
func (book *Book) RemoveLog(ctx context.Context, sender log.Author, ref dsref.Ref) error {
	l, err := book.readRefLog(ref)
	if err != nil {
		return err
	}

	// eventually access control will dictate which logs can be written by whom.
	// For now we only allow users to merge logs they've written
	// book will need access to a store of public keys before we can verify
	// signatures non-same-senders
	// if err := l.Verify(sender.AuthorPubKey()); err != nil {
	// 	return err
	// }

	if l.Author() != sender.AuthorID() {
		return fmt.Errorf("authors can only remove logs they own")
	}

	book.bk.RemoveLog(nameModel, dsRefToLogPath(ref)...)
	return book.save(ctx)
}

func dsRefToLogPath(ref dsref.Ref) (path []string) {
	for _, str := range []string{
		ref.Username,
		ref.Name,
	} {
		path = append(path, str)
	}
	return path
}

// ConstructDatasetLog creates a sparse log from a connected dataset history
// where no prior log exists
// the given history MUST be ordered from oldest to newest commits
// TODO (b5) - this presently only works for datasets in an author's user
// namespace
func (book *Book) ConstructDatasetLog(ctx context.Context, ref dsref.Ref, history []*dataset.Dataset) error {
	l, err := book.readRefLog(ref)
	if err == nil {
		// if the log already exists, it will either as-or-more rich than this log,
		// refuse to overwrite
		return ErrLogTooShort
	}

	l = book.initName(ctx, ref.Name)
	for _, ds := range history {
		book.appendVersionSave(l, ds)
	}

	return book.save(ctx)
}

// DatasetInfo describes info aboud a dataset version in a repository
type DatasetInfo struct {
	Ref         dsref.Ref // version Reference
	Published   bool      // indicates whether this reference is listed as an available dataset
	Timestamp   time.Time // creation timestamp
	CommitTitle string    // title from commit
}

func infoFromOp(ref dsref.Ref, op log.Op) DatasetInfo {
	return DatasetInfo{
		Ref: dsref.Ref{
			Username:  ref.Username,
			ProfileID: ref.ProfileID,
			Name:      ref.Name,
			Path:      op.Ref,
		},
		Timestamp:   time.Unix(0, op.Timestamp),
		CommitTitle: op.Note,
	}
}

// Versions plays a set of operations for a given log, producing a State struct
// that describes the current state of a dataset
func (book Book) Versions(ref dsref.Ref, offset, limit int) ([]DatasetInfo, error) {
	l, err := book.readRefLog(ref)
	if err != nil {
		return nil, err
	}

	refs := []DatasetInfo{}
	for _, op := range l.Ops {
		switch op.Model {
		case versionModel:
			switch op.Type {
			case log.OpTypeInit:
				refs = append(refs, infoFromOp(ref, op))
			case log.OpTypeAmend:
				refs[len(refs)-1] = infoFromOp(ref, op)
			case log.OpTypeRemove:
				refs = refs[:len(refs)-int(op.Size)]
			}
		case publicationModel:
			switch op.Type {
			case log.OpTypeInit:
				for i := 1; i <= int(op.Size); i++ {
					refs[len(refs)-i].Published = true
				}
			case log.OpTypeRemove:
				for i := 1; i <= int(op.Size); i++ {
					refs[len(refs)-i].Published = false
				}
			}
		}
	}

	if offset > len(refs) {
		offset = len(refs)
	}
	refs = refs[offset:]

	if limit < len(refs) {
		refs = refs[:limit]
	}

	return refs, nil
}

// LogEntry is a simplified representation of a log operation
type LogEntry struct {
	Timestamp time.Time
	Author    string
	Action    string
	Note      string
}

// String formats a LogEntry as a String
func (l LogEntry) String() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", l.Timestamp.Format(time.Kitchen), l.Author, l.Action, l.Note)
}

// LogEntries returns a summarized "line-by-line" representation of a log for a
// given dataset reference
func (book Book) LogEntries(ctx context.Context, ref dsref.Ref, offset, limit int) ([]LogEntry, error) {
	l, err := book.readRefLog(ref)
	if err != nil {
		return nil, err
	}

	res := []LogEntry{}
	for _, op := range l.Ops {
		if offset > 0 {
			offset--
			continue
		}
		res = append(res, logEntryFromOp(ref.Username, op))
		if len(res) == limit {
			break
		}
	}

	return res, nil
}

var actionStrings = map[uint32][3]string{
	userModel:        [3]string{"create profile", "update profile", "delete profile"},
	nameModel:        [3]string{"init", "rename", "delete"},
	versionModel:     [3]string{"save", "amend", "remove"},
	publicationModel: [3]string{"publish", "", "unpublish"},
	aclModel:         [3]string{"update access", "update access", ""},
	cronJobModel:     [3]string{"ran update", "", ""},
}

func logEntryFromOp(author string, op log.Op) LogEntry {
	return LogEntry{
		Timestamp: time.Unix(0, op.Timestamp),
		Author:    author,
		Action:    actionStrings[op.Model][int(op.Type)-1],
		Note:      op.Note,
	}
}

// RawLogs returns a serialized, complete set of logs keyed by model type logs
func (book Book) RawLogs(ctx context.Context) map[string][]Log {
	logs := map[string][]Log{}
	for m, lgs := range book.bk.Logs() {
		ls := make([]Log, len(lgs))
		for i, l := range lgs {
			ls[i] = newLog(l)
		}
		logs[modelString(m)] = ls
	}
	return logs
}

// Log is a human-oriented representation of log.Log intended for serialization
type Log struct {
	Ops  []Op  `json:"ops,omitempty"`
	Logs []Log `json:"logs,omitempty"`
}

func newLog(lg *log.Log) Log {
	ops := make([]Op, len(lg.Ops))
	for i, o := range lg.Ops {
		ops[i] = newOp(o)
	}

	var ls []Log
	if len(lg.Logs) > 0 {
		ls = make([]Log, len(lg.Logs))
		for i, l := range lg.Logs {
			ls[i] = newLog(l)
		}
	}

	return Log{
		Ops:  ops,
		Logs: ls,
	}
}

// Op is a human-oriented representation of log.Op intended for serialization
type Op struct {
	// type of operation
	Type string `json:"type,omitempty"`
	// data model to operate on
	Model string `json:"model,omitempty"`
	// identifier of data this operation is documenting
	Ref string `json:"ref,omitempty"`
	// previous reference in a causal history
	Prev string `json:"prev,omitempty"`
	// references this operation relates to. usage is operation type-dependant
	Relations []string `json:"relations,omitempty"`
	// human-readable name for the reference
	Name string `json:"name,omitempty"`
	// identifier for author
	AuthorID string `json:"authorID,omitempty"`
	// operation timestamp, for annotation purposes only
	Timestamp time.Time `json:"timestamp,omitempty"`
	// size of the referenced value in bytes
	Size uint64 `json:"size,omitempty"`
	// operation annotation for users. eg: commit title
	Note string `json:"note,omitempty"`
}

func newOp(op log.Op) Op {
	return Op{
		Type:      opTypeString(op.Type),
		Model:     modelString(op.Model),
		Ref:       op.Ref,
		Prev:      op.Prev,
		Relations: op.Relations,
		Name:      op.Name,
		AuthorID:  op.AuthorID,
		Timestamp: time.Unix(0, op.Timestamp),
		Size:      op.Size,
		Note:      op.Note,
	}
}

func opTypeString(op log.OpType) string {
	switch op {
	case log.OpTypeInit:
		return "init"
	case log.OpTypeAmend:
		return "amend"
	case log.OpTypeRemove:
		return "remove"
	default:
		return ""
	}
}

func refFromDataset(ds *dataset.Dataset) dsref.Ref {
	return dsref.Ref{
		Username:  ds.Peername,
		ProfileID: ds.ProfileID,
		Name:      ds.Name,
		Path:      ds.Path,
	}
}

func (book Book) readRefLog(ref dsref.Ref) (*log.Log, error) {
	if ref.Username == "" {
		return nil, fmt.Errorf("ref.Username is required")
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
		return "", fmt.Errorf("getting pubkey bytes: %s", err.Error())
	}

	mh, err := multihash.Sum(pubkeybytes, multihash.SHA2_256, 32)
	if err != nil {
		return "", fmt.Errorf("summing pubkey: %s", err.Error())
	}

	return mh.B58String(), nil
}
