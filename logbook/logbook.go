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

	logger "github.com/ipfs/go-log"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event/hook"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook/oplog"
)

var (
	// ErrNoLogbook indicates a logbook doesn't exist
	ErrNoLogbook = fmt.Errorf("logbook: does not exist")
	// ErrNotFound is a sentinel error for data not found in a logbook
	ErrNotFound = fmt.Errorf("logbook: not found")
	// ErrLogTooShort indicates a log is missing elements. Because logs are
	// append-only, passing a shorter log than the one on file is grounds
	// for rejection
	ErrLogTooShort = fmt.Errorf("logbook: log is too short")

	// NewTimestamp generates the current unix nanosecond time.
	// This is mainly here for tests to override
	NewTimestamp = func() int64 { return time.Now().UnixNano() }

	// package logger
	log = logger.Logger("logbook")
)

const (
	// AuthorModel is the enum for an author model
	AuthorModel uint32 = iota
	// DatasetModel is the enum for a dataset model
	DatasetModel
	// BranchModel is the enum for a branch model
	BranchModel
	// CommitModel is the enum for a commit model
	CommitModel
	// PublicationModel is the enum for a publication model
	PublicationModel
	// ACLModel is the enum for a acl model
	ACLModel
)

// DefaultBranchName is the default name all branch-level logbook data is read
// from and written to. we currently don't present branches as a user-facing
// feature in qri, but logbook supports them
const DefaultBranchName = "main"

// ModelString gets a unique string descriptor for an integral model identifier
func ModelString(m uint32) string {
	switch m {
	case AuthorModel:
		return "user"
	case DatasetModel:
		return "dataset"
	case BranchModel:
		return "branch"
	case CommitModel:
		return "commit"
	case PublicationModel:
		return "publication"
	case ACLModel:
		return "acl"
	default:
		return ""
	}
}

// Book wraps a oplog.Book with a higher-order API specific to Qri
type Book struct {
	store oplog.Logstore

	pk         crypto.PrivKey
	authorID   string
	authorName string

	fsLocation string
	fs         qfs.Filesystem

	onChangeHook func(hook.DsChange)
}

// NewBook creates a book with a user-provided logstore
func NewBook(pk crypto.PrivKey, store oplog.Logstore) *Book {
	return &Book{pk: pk, store: store}
}

// NewJournal initializes a logbook owned by a single author, reading any
// existing data at the given filesystem location.
// logbooks are encrypted at rest with the given private key
func NewJournal(pk crypto.PrivKey, username string, fs qfs.Filesystem, location string) (*Book, error) {
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

	book := &Book{
		store:      &oplog.Journal{},
		fs:         fs,
		pk:         pk,
		authorName: username,
		fsLocation: location,
	}

	if err := book.load(ctx); err != nil {
		if err == ErrNotFound {
			err = book.initialize(ctx)
			return book, err
		}
		return nil, err
	}
	// else {
	// TODO (b5) verify username integrity on load
	// }

	return book, nil
}

func (book *Book) initialize(ctx context.Context) error {
	keyID, err := identity.KeyIDFromPriv(book.pk)
	if err != nil {
		return err
	}

	// initialize author's log of user actions
	userActions := oplog.InitLog(oplog.Op{
		Type:      oplog.OpTypeInit,
		Model:     AuthorModel,
		Name:      book.AuthorName(),
		AuthorID:  keyID,
		Timestamp: NewTimestamp(),
	})
	book.authorID = userActions.ID()

	if err = book.store.MergeLog(ctx, userActions); err != nil {
		return err
	}
	if al, ok := book.store.(oplog.AuthorLogstore); ok {
		al.SetID(ctx, book.authorID)
	}
	return book.save(ctx)
}

// ActivePeerID returns the in-use PeerID of the logbook author
// TODO (b5) - remove the need for this context by caching the active PeerID
// at key load / save / mutation points
func (book *Book) ActivePeerID(ctx context.Context) (id string, err error) {
	if book == nil {
		return "", ErrNoLogbook
	}

	lg, err := book.store.Get(ctx, book.authorID)
	if err != nil {
		panic(err)
	}
	return lg.Author(), nil
}

// Author returns this book's author
func (book *Book) Author() identity.Author {
	return book
}

// AuthorName returns the human-readable name of the author
func (book *Book) AuthorName() string {
	return book.authorName
}

// AuthorID returns the machine identifier for a name
func (book *Book) AuthorID() string {
	return book.authorID
}

// AuthorPubKey gives this book's author public key
func (book *Book) AuthorPubKey() crypto.PubKey {
	return book.pk.GetPublic()
}

// RenameAuthor marks a change in author name
func (book *Book) RenameAuthor() error {
	return fmt.Errorf("not finished")
}

// DeleteAuthor removes an author, used on teardown
func (book *Book) DeleteAuthor() error {
	return fmt.Errorf("not finished")
}

// save writes the book to book.fsLocation
func (book *Book) save(ctx context.Context) (err error) {
	if al, ok := book.store.(oplog.AuthorLogstore); ok {
		ciphertext, err := al.FlatbufferCipher(book.pk)
		if err != nil {
			return err
		}

		file := qfs.NewMemfileBytes(book.fsLocation, ciphertext)
		book.fsLocation, err = book.fs.Put(ctx, file)
	}
	return err
}

// load reads the book dataset from book.fsLocation
func (book *Book) load(ctx context.Context) error {
	if al, ok := book.store.(oplog.AuthorLogstore); ok {

		f, err := book.fs.Get(ctx, book.fsLocation)
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

		if err = al.UnmarshalFlatbufferCipher(ctx, book.pk, ciphertext); err != nil {
			return err
		}

		book.authorID = al.ID()
	}
	return nil
}

// WriteAuthorRename adds an operation updating the author's username
func (book *Book) WriteAuthorRename(ctx context.Context, newName string) error {
	if book == nil {
		return ErrNoLogbook
	}
	if !dsref.IsValidName(newName) {
		return fmt.Errorf("logbook: author name %q invalid", newName)
	}

	authorLog, err := book.authorLog(ctx)
	if err != nil {
		return err
	}
	authorLog.Append(oplog.Op{
		Type:      oplog.OpTypeAmend,
		Model:     AuthorModel,
		AuthorID:  book.AuthorID(),
		Name:      newName,
		Timestamp: NewTimestamp(),
	})

	if err := book.save(ctx); err != nil {
		return err
	}

	book.authorName = newName
	return nil
}

// WriteDatasetInit initializes a new dataset name within the author's namespace
func (book *Book) WriteDatasetInit(ctx context.Context, dsName string) (string, error) {
	if book == nil {
		return "", ErrNoLogbook
	}
	if dsName == "" {
		return "", fmt.Errorf("logbook: name is required to initialize a dataset")
	}
	if !dsref.IsValidName(dsName) {
		return "", fmt.Errorf("logbook: dataset name %q invalid", dsName)
	}
	if _, err := book.DatasetRef(ctx, dsref.Ref{Username: book.AuthorName(), Name: dsName}); err == nil {
		return "", fmt.Errorf("logbook: dataset named %q already exists", dsName)
	}

	authorLog, err := book.authorLog(ctx)
	if err != nil {
		return "", err
	}
	profileID := authorLog.ProfileID()

	log.Debugf("initializing name: '%s'", dsName)
	dsLog := oplog.InitLog(oplog.Op{
		Type:      oplog.OpTypeInit,
		Model:     DatasetModel,
		AuthorID:  book.AuthorID(),
		Name:      dsName,
		Timestamp: NewTimestamp(),
	})

	branch := oplog.InitLog(oplog.Op{
		Type:      oplog.OpTypeInit,
		Model:     BranchModel,
		AuthorID:  book.AuthorID(),
		Name:      DefaultBranchName,
		Timestamp: NewTimestamp(),
	})

	dsLog.AddChild(branch)

	authorLog.AddChild(dsLog)

	initID := dsLog.ID()

	if book.onChangeHook != nil {
		// TODO(dlong): Perhaps in the future, pass the authorID (hash of the author creation
		// block) to the dscache, use that instead-of or in-addition-to the profileID.
		book.onChangeHook(hook.DsChange{
			Type:       hook.DatasetNameInit,
			InitID:     initID,
			Username:   book.AuthorName(),
			ProfileID:  profileID,
			PrettyName: dsName,
		})
	}

	return initID, book.save(ctx)
}

// WriteDatasetRename marks renaming a dataset
func (book *Book) WriteDatasetRename(ctx context.Context, initID string, newName string) error {
	if book == nil {
		return ErrNoLogbook
	}
	if !dsref.IsValidName(newName) {
		return fmt.Errorf("logbook: new dataset name %q invalid", newName)
	}

	log.Debugf("WriteDatasetRename: '%s' -> '%s'", initID, newName)

	dsLog, err := book.datasetLog(ctx, initID)
	if err != nil {
		return err
	}

	dsLog.Append(oplog.Op{
		Type:      oplog.OpTypeAmend,
		Model:     DatasetModel,
		Name:      newName,
		Timestamp: NewTimestamp(),
	})
	if book.onChangeHook != nil {
		book.onChangeHook(hook.DsChange{
			Type:       hook.DatasetRename,
			InitID:     initID,
			PrettyName: newName,
		})
	}
	return book.save(ctx)
}

// RefToInitID converts a dsref to an initID by iterating the entire logbook looking for a match.
// This function is inefficient, iterating the entire set of operations in a log. Replacing this
// function call with mechanisms in dscache will fix this problem.
// TODO(dustmop): Don't depend on this function permanently, use a higher level resolver and
// convert all callers of this function to use that resolver's initID instead of converting a
// dsref yet again.
func (book *Book) RefToInitID(ref dsref.Ref) (string, error) {
	if book == nil {
		return "", ErrNoLogbook
	}

	// NOTE: Bad to retrieve the background context here, but HeadRef just ignores it anyway.
	ctx := context.Background()

	// HeadRef is inefficient, iterates the top two levels of the logbook.
	// Runs in O(M*N) where M = number of users, N = number of datasets per user.
	dsLog, err := book.store.HeadRef(ctx, ref.Username, ref.Name)
	if err != nil {
		if err == oplog.ErrNotFound {
			return "", ErrNotFound
		}
		return "", err
	}
	return dsLog.ID(), nil
}

// Return a strongly typed UserLog for the given profileID. Top level of the logbook.
func (book Book) userLog(ctx context.Context, profileID string) (*UserLog, error) {
	return nil, fmt.Errorf("TODO(dustmop): Not Implemented")
}

// Return a strongly typed UserLog for the author of the logbook.
func (book Book) authorLog(ctx context.Context) (*UserLog, error) {
	lg, err := book.store.Get(ctx, book.authorID)
	if err != nil {
		return nil, err
	}
	return &UserLog{l: lg}, nil
}

// Return a strongly typed DatasetLog. Uses DatasetModel model.
func (book *Book) datasetLog(ctx context.Context, initID string) (*DatasetLog, error) {
	lg, err := book.store.Get(ctx, initID)
	if err != nil {
		return nil, err
	}
	return &DatasetLog{l: lg}, nil
}

// Return a strongly typed BranchLog
func (book *Book) branchLog(ctx context.Context, initID string) (*BranchLog, error) {
	lg, err := book.store.Get(ctx, initID)
	if err != nil {
		return nil, err
	}
	if len(lg.Logs) != 1 {
		return nil, fmt.Errorf("expected dataset to have 1 branch, has %d", len(lg.Logs))
	}
	return &BranchLog{l: lg.Logs[0]}, nil
}

// WriteDatasetDelete closes a dataset, marking it as deleted
func (book *Book) WriteDatasetDelete(ctx context.Context, initID string) error {
	if book == nil {
		return ErrNoLogbook
	}
	log.Debugf("WriteDatasetDelete: '%s'", initID)

	dsLog, err := book.datasetLog(ctx, initID)
	if err != nil {
		return err
	}

	dsLog.Append(oplog.Op{
		Type:      oplog.OpTypeRemove,
		Model:     DatasetModel,
		Timestamp: NewTimestamp(),
	})
	if book.onChangeHook != nil {
		book.onChangeHook(hook.DsChange{
			Type:   hook.DatasetDeleteAll,
			InitID: initID,
		})
	}

	return book.save(ctx)
}

// WriteVersionSave adds an operation to a log marking the creation of a
// dataset version. Book will copy details from the provided dataset pointer
func (book *Book) WriteVersionSave(ctx context.Context, initID string, ds *dataset.Dataset) error {
	if book == nil {
		return ErrNoLogbook
	}

	log.Debugf("WriteVersionSave: %s", initID)
	branchLog, err := book.branchLog(ctx, initID)
	if err != nil {
		return err
	}

	topIndex := book.appendVersionSave(branchLog, ds)
	// TODO(dlong): Think about how to handle a failure exactly here, what needs to be rolled back?
	err = book.save(ctx)
	if err != nil {
		return err
	}

	info := dsref.ConvertDatasetToVersionInfo(ds)

	if book.onChangeHook != nil {
		book.onChangeHook(hook.DsChange{
			Type:     hook.DatasetCommitChange,
			InitID:   initID,
			TopIndex: topIndex,
			HeadRef:  info.Path,
			Info:     &info,
		})
	}
	return nil
}

func (book *Book) appendVersionSave(blog *BranchLog, ds *dataset.Dataset) int {
	op := oplog.Op{
		Type:  oplog.OpTypeInit,
		Model: CommitModel,
		Ref:   ds.Path,
		Prev:  ds.PreviousPath,

		Timestamp: ds.Commit.Timestamp.UnixNano(),
		Note:      ds.Commit.Title,
	}

	if ds.Structure != nil {
		op.Size = int64(ds.Structure.Length)
	}

	blog.Append(op)

	return blog.Size() - 1
}

// WriteVersionAmend adds an operation to a log when a dataset amends a commit
// TODO(dustmop): Currently unused by codebase, only called in tests.
func (book *Book) WriteVersionAmend(ctx context.Context, initID string, ds *dataset.Dataset) error {
	if book == nil {
		return ErrNoLogbook
	}
	log.Debugf("WriteVersionAmend: '%s'", initID)

	branchLog, err := book.branchLog(ctx, initID)
	if err != nil {
		return err
	}

	branchLog.Append(oplog.Op{
		Type:  oplog.OpTypeAmend,
		Model: CommitModel,
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
func (book *Book) WriteVersionDelete(ctx context.Context, initID string, revisions int) error {
	if book == nil {
		return ErrNoLogbook
	}
	log.Debugf("WriteVersionDelete: %s, revisions: %d", initID, revisions)

	branchLog, err := book.branchLog(ctx, initID)
	if err != nil {
		return err
	}

	branchLog.Append(oplog.Op{
		Type:  oplog.OpTypeRemove,
		Model: CommitModel,
		Size:  int64(revisions),
		// TODO (b5) - finish
	})

	// Calculate the commits after collapsing deletions found at the tail of history (most recent).
	items := branchToLogItems(branchLog, dsref.Ref{}, 0, -1, false)

	if len(items) > 0 {
		lastItem := items[len(items)-1]
		if book.onChangeHook != nil {
			book.onChangeHook(hook.DsChange{
				Type:     hook.DatasetCommitChange,
				InitID:   initID,
				TopIndex: len(items),
				HeadRef:  lastItem.Path,
				Info:     &lastItem.VersionInfo,
			})
		}
	}

	return book.save(ctx)
}

// WritePublish adds an operation to a log marking the publication of a number
// of versions to one or more destinations
func (book *Book) WritePublish(ctx context.Context, initID string, revisions int, destinations ...string) error {
	if book == nil {
		return ErrNoLogbook
	}
	log.Debugf("WritePublish: %s, revisions: %d, destinations: %v", initID, revisions, destinations)

	branchLog, err := book.branchLog(ctx, initID)
	if err != nil {
		return err
	}

	branchLog.Append(oplog.Op{
		Type:      oplog.OpTypeInit,
		Model:     PublicationModel,
		Size:      int64(revisions),
		Relations: destinations,
		// TODO (b5) - finish
	})

	return book.save(ctx)
}

// WriteUnpublish adds an operation to a log marking an unpublish request for a
// count of sequential versions from HEAD
// TODO(dustmop): Currently unused by codebase, only called in tests.
func (book *Book) WriteUnpublish(ctx context.Context, initID string, revisions int, destinations ...string) error {
	if book == nil {
		return ErrNoLogbook
	}
	log.Debugf("WriteUnpublish: %s, revisions: %d, destinations: %v", initID, revisions, destinations)

	branchLog, err := book.branchLog(ctx, initID)
	if err != nil {
		return err
	}

	branchLog.Append(oplog.Op{
		Type:      oplog.OpTypeRemove,
		Model:     PublicationModel,
		Size:      int64(revisions),
		Relations: destinations,
		// TODO (b5) - finish
	})

	return book.save(ctx)
}

// SetChangeHook assigns a hook that will be called when a dataset changes
func (book *Book) SetChangeHook(changeHook func(hook.DsChange)) {
	book.onChangeHook = changeHook
}

// ListAllLogs lists all of the logs in the logbook
func (book Book) ListAllLogs(ctx context.Context) ([]*oplog.Log, error) {
	return book.store.Logs(ctx, 0, -1)
}

// Log gets a log for a given ID
func (book Book) Log(ctx context.Context, id string) (*oplog.Log, error) {
	return book.store.Get(ctx, id)
}

// ResolveRef finds the identifier & head path for a dataset reference
// implements resolve.NameResolver interface
func (book *Book) ResolveRef(ctx context.Context, ref *dsref.Ref) (string, error) {
	if book == nil {
		return "", dsref.ErrNotFound
	}

	initID, err := book.RefToInitID(*ref)
	if err != nil {
		return "", dsref.ErrNotFound
	}
	ref.InitID = initID

	if ref.Path == "" {
		br, err := book.branchLog(ctx, initID)
		if err != nil {
			return "", err
		}
		ref.Path = book.latestSavePath(br.l)
	}

	return "", nil
}

func (book *Book) latestSavePath(branchLog *oplog.Log) string {
	removes := 0

	for i := len(branchLog.Ops) - 1; i >= 0; i-- {
		op := branchLog.Ops[i]
		if op.Model == CommitModel {
			switch op.Type {
			case oplog.OpTypeRemove:
				removes += int(op.Size)
			case oplog.OpTypeInit, oplog.OpTypeAmend:
				if removes > 0 {
					removes--
				}
				if removes == 0 {
					return op.Ref
				}
			}
		}
	}
	return ""
}

// UserDatasetRef gets a user's log and a dataset reference, the returned log
// will be a user log with a single dataset log containing all known branches:
//   user
//     dataset
//       branch
//       branch
//       ...
func (book Book) UserDatasetRef(ctx context.Context, ref dsref.Ref) (*oplog.Log, error) {
	if ref.Username == "" {
		return nil, fmt.Errorf("logbook: reference Username is required")
	}
	if ref.Name == "" {
		return nil, fmt.Errorf("logbook: reference Name is required")
	}

	// fetch user log
	author, err := book.store.HeadRef(ctx, ref.Username)
	if err != nil {
		return nil, err
	}

	// fetch dataset & all branches
	ds, err := book.store.HeadRef(ctx, ref.Username, ref.Name)
	if err != nil {
		return nil, err
	}

	br, err := book.BranchRef(ctx, ref)
	if err != nil {
		return nil, err
	}

	ds.AddChild(br)

	// construct a sparse oplog of just user, dataset, and branches
	sparseLog := &oplog.Log{Ops: author.Ops}
	sparseLog.AddChild(ds)
	return sparseLog, nil
}

// DatasetRef gets a dataset log and all branches. Dataset logs describe
// activity affecting an entire dataset. Things like dataset name changes and
// access control changes are kept in the dataset log
//
// currently all logs are hardcoded to only accept one branch name. This
// function will always return a single branch
//
// TODO(dustmop): Do not add new callers to this, transition away (preferring datasetLog instead),
// and delete it.
func (book Book) DatasetRef(ctx context.Context, ref dsref.Ref) (*oplog.Log, error) {
	if ref.Username == "" {
		return nil, fmt.Errorf("logbook: ref.Username is required")
	}
	if ref.Name == "" {
		return nil, fmt.Errorf("logbook: ref.Name is required")
	}

	return book.store.HeadRef(ctx, ref.Username, ref.Name)
}

// BranchRef gets a branch log for a dataset reference. Branch logs describe
// a line of commits
//
// currently all logs are hardcoded to only accept one branch name. This
// function always returns
//
// TODO(dustmop): Do not add new callers to this, transition away (preferring branchLog instead),
// and delete it.
func (book Book) BranchRef(ctx context.Context, ref dsref.Ref) (*oplog.Log, error) {
	if ref.Username == "" {
		return nil, fmt.Errorf("logbook: ref.Username is required")
	}
	if ref.Name == "" {
		return nil, fmt.Errorf("logbook: ref.Name is required")
	}

	return book.store.HeadRef(ctx, ref.Username, ref.Name, DefaultBranchName)
}

// SignLog populates the signature field of a log using the author's private key
func (book Book) SignLog(log *oplog.Log) error {
	return log.Sign(book.pk)
}

// LogBytes signs a log with this book's private key and writes to a flatbuffer
func (book Book) LogBytes(log *oplog.Log) ([]byte, error) {
	if err := book.SignLog(log); err != nil {
		return nil, err
	}
	return log.FlatbufferBytes(), nil
}

// DsrefAliasForLog parses log data into a dataset alias reference, populating
// only the username and name components of a dataset.
// the passed in oplog must refer unambiguously to a dataset or branch.
// book.Log() returns exact log references
func DsrefAliasForLog(log *oplog.Log) (dsref.Ref, error) {
	ref := dsref.Ref{}
	if log == nil {
		return ref, fmt.Errorf("logbook: log is required")
	}
	if log.Model() != AuthorModel {
		return ref, fmt.Errorf("logbook: log isn't rooted as an author")
	}
	if len(log.Logs) != 1 {
		return ref, fmt.Errorf("logbook: ambiguous dataset reference")
	}

	ref = dsref.Ref{
		Username: log.Name(),
		Name:     log.Logs[0].Name(),
	}

	return ref, nil
}

// MergeLog adds a log to the logbook, merging with any existing log data
func (book *Book) MergeLog(ctx context.Context, sender identity.Author, lg *oplog.Log) error {
	if book == nil {
		return ErrNoLogbook
	}
	// eventually access control will dictate which logs can be written by whom.
	// For now we only allow users to merge logs they've written
	// book will need access to a store of public keys before we can verify
	// signatures non-same-senders
	if err := lg.Verify(sender.AuthorPubKey()); err != nil {
		return err
	}

	// if lg.ID() != sender.AuthorID() {
	// 	return fmt.Errorf("authors can only push logs they own")
	// }

	if err := book.store.MergeLog(ctx, lg); err != nil {
		return err
	}

	return book.save(ctx)
}

// RemoveLog removes an entire log from a logbook
func (book *Book) RemoveLog(ctx context.Context, sender identity.Author, ref dsref.Ref) error {
	if book == nil {
		return ErrNoLogbook
	}

	l, err := book.BranchRef(ctx, ref)
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

	root := l
	for {
		p := root.Parent()
		if p == nil {
			break
		}
		root = p
	}

	if root.ID() != sender.AuthorID() {
		return fmt.Errorf("authors can only remove logs they own")
	}

	book.store.RemoveLog(ctx, dsRefToLogPath(ref)...)
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
	if book == nil {
		return ErrNoLogbook
	}

	if _, err := book.RefToInitID(ref); err == nil {
		// if the log already exists, it will either as-or-more rich than this log,
		// refuse to overwrite
		return ErrLogTooShort
	}

	initID, err := book.WriteDatasetInit(ctx, ref.Name)
	if err != nil {
		return err
	}
	branchLog, err := book.branchLog(ctx, initID)
	if err != nil {
		return err
	}
	for _, ds := range history {
		book.appendVersionSave(branchLog, ds)
	}
	return book.save(ctx)
}

func itemFromOp(ref dsref.Ref, op oplog.Op) DatasetLogItem {
	return DatasetLogItem{
		VersionInfo: dsref.VersionInfo{
			Username:   ref.Username,
			ProfileID:  ref.ProfileID,
			Name:       ref.Name,
			Path:       op.Ref,
			CommitTime: time.Unix(0, op.Timestamp),
			BodySize:   int(op.Size),
		},
		CommitTitle: op.Note,
	}
}

// Items collapses the history of a dataset branch into linear log items
func (book Book) Items(ctx context.Context, ref dsref.Ref, offset, limit int) ([]DatasetLogItem, error) {
	initID, err := book.RefToInitID(dsref.Ref{Username: ref.Username, Name: ref.Name})
	if err != nil {
		return nil, err
	}
	branchLog, err := book.branchLog(ctx, initID)
	if err != nil {
		return nil, err
	}

	return branchToLogItems(branchLog, ref, offset, limit, true), nil
}

// ConvertLogsToItems collapses the history of a dataset branch into linear log items
func ConvertLogsToItems(l *oplog.Log, ref dsref.Ref) []DatasetLogItem {
	return branchToLogItems(branchLogFromRawLog(l), ref, 0, -1, true)
}

// Items collapses the history of a dataset branch into linear log items
// If collapseAllDeletes is true, all delete operations will remove the refs before them. Otherwise,
// only refs at the end of history will be removed in this manner.
func branchToLogItems(blog *BranchLog, ref dsref.Ref, offset, limit int, collapseAllDeletes bool) []DatasetLogItem {
	refs := []DatasetLogItem{}
	deleteAtEnd := 0
	for _, op := range blog.Ops() {
		switch op.Model {
		case CommitModel:
			switch op.Type {
			case oplog.OpTypeInit:
				refs = append(refs, itemFromOp(ref, op))
			case oplog.OpTypeAmend:
				deleteAtEnd = 0
				refs[len(refs)-1] = itemFromOp(ref, op)
			case oplog.OpTypeRemove:
				if collapseAllDeletes {
					refs = refs[:len(refs)-int(op.Size)]
				} else {
					deleteAtEnd += int(op.Size)
				}
			}
		case PublicationModel:
			switch op.Type {
			case oplog.OpTypeInit:
				for i := 1; i <= int(op.Size); i++ {
					refs[len(refs)-i].Published = true
				}
			case oplog.OpTypeRemove:
				for i := 1; i <= int(op.Size); i++ {
					refs[len(refs)-i].Published = false
				}
			}
		}
	}

	if deleteAtEnd > 0 {
		if deleteAtEnd < len(refs) {
			refs = refs[:len(refs)-deleteAtEnd]
		} else {
			refs = []DatasetLogItem{}
		}
	}

	// reverse the slice, placing newest first
	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for i := len(refs)/2 - 1; i >= 0; i-- {
		opp := len(refs) - 1 - i
		refs[i], refs[opp] = refs[opp], refs[i]
	}

	if offset > len(refs) {
		offset = len(refs)
	}
	refs = refs[offset:]

	if limit < len(refs) && limit != -1 {
		refs = refs[:limit]
	}

	return refs
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
	l, err := book.BranchRef(ctx, ref)
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
	AuthorModel:      [3]string{"create profile", "update profile", "delete profile"},
	DatasetModel:     [3]string{"init dataset", "rename dataset", "delete dataset"},
	BranchModel:      [3]string{"init branch", "rename branch", "delete branch"},
	CommitModel:      [3]string{"save commit", "amend commit", "remove commit"},
	PublicationModel: [3]string{"publish", "", "unpublish"},
	ACLModel:         [3]string{"update access", "update access", "remove all access"},
}

func logEntryFromOp(author string, op oplog.Op) LogEntry {
	note := op.Note
	if note == "" && op.Name != "" {
		note = op.Name
	}
	return LogEntry{
		Timestamp: time.Unix(0, op.Timestamp),
		Author:    author,
		Action:    actionStrings[op.Model][int(op.Type)-1],
		Note:      note,
	}
}

// PlainLogs returns plain-old-data representations of the logs, intended for serialization
func (book Book) PlainLogs(ctx context.Context) ([]PlainLog, error) {
	raw, err := book.store.Logs(ctx, 0, -1)
	if err != nil {
		return nil, err
	}

	logs := make([]PlainLog, len(raw))
	for i, l := range raw {
		logs[i] = NewPlainLog(l)
	}
	return logs, nil
}

// PlainLog is a human-oriented representation of oplog.Log intended for serialization
type PlainLog struct {
	Ops  []PlainOp  `json:"ops,omitempty"`
	Logs []PlainLog `json:"logs,omitempty"`
}

// NewPlainLog converts an oplog to a plain log
func NewPlainLog(lg *oplog.Log) PlainLog {
	if lg == nil {
		return PlainLog{}
	}

	ops := make([]PlainOp, len(lg.Ops))
	for i, o := range lg.Ops {
		ops[i] = newPlainOp(o)
	}

	var ls []PlainLog
	if len(lg.Logs) > 0 {
		ls = make([]PlainLog, len(lg.Logs))
		for i, l := range lg.Logs {
			ls[i] = NewPlainLog(l)
		}
	}

	return PlainLog{
		Ops:  ops,
		Logs: ls,
	}
}

// DatasetLogItem is a line item in a dataset response
type DatasetLogItem struct {
	// Decription of a dataset reference
	dsref.VersionInfo
	// Title field from the commit
	CommitTitle string `json:"commitTitle,omitempty"`
	// Message field from the commit
	CommitMessage string `json:"commitMessage,omitempty"`
}

// PlainOp is a human-oriented representation of oplog.Op intended for serialization
type PlainOp struct {
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
	Size int64 `json:"size,omitempty"`
	// operation annotation for users. eg: commit title
	Note string `json:"note,omitempty"`
}

func newPlainOp(op oplog.Op) PlainOp {
	return PlainOp{
		Type:      opTypeString(op.Type),
		Model:     ModelString(op.Model),
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

func opTypeString(op oplog.OpType) string {
	switch op {
	case oplog.OpTypeInit:
		return "init"
	case oplog.OpTypeAmend:
		return "amend"
	case oplog.OpTypeRemove:
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
