// Package oplog is an operation-based replicated data type of append-only logs
// oplog has three main structures: logbook, log, and op
// A log is a sequence of operations attributed to a single author, designated
// by a private key.
// an operation is a record of some action an author took. Applications iterate
// the sequence of operations to produce the current state.
// Logs can be arranged into hierarchies to form logical groupings.
// A book contains an author's logs, both logs they've written as well as logs
// replicated from other authors. Books are encrypted at rest using the author
// private key.
package oplog

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	flatbuffers "github.com/google/flatbuffers/go"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/logbook/oplog/logfb"
	"golang.org/x/crypto/blake2b"
)

var (
	// ErrNotFound is a sentinel error for data not found in a logbook
	ErrNotFound = fmt.Errorf("log: not found")
)

// Logstore persists a set of operations organized into hierarchical append-only
// logs
type Logstore interface {
	// MergeLog adds a Log to the store, controlling for conflicts
	// * logs that are already known to the store are merged with a
	//   longest-log-wins strategy, adding all descendants
	// * new top level logs are appended to the store, including all descendants
	// * attempting to add a log with a parent not already in the store MUST fail
	//
	// TODO (b5) - currently a Log pointer doesn't provide a clear method for
	// getting the ID of it's parent, which negates the potential for attempting
	// to merge child log, so we don't need to control for the third point quite
	// yet
	MergeLog(ctx context.Context, l *Log) error

	// Remove a log from the store, all descendant logs must be removed as well
	RemoveLog(ctx context.Context, names ...string) error

	// Logs lists top level logs in the store, that is, the set logs that have no
	// parent. passing -1 as a limit returns all top level logs after the offset
	//
	// The order of logs returned is up to the store, but the stored order must
	// be deterministic
	Logs(ctx context.Context, offset, limit int) (topLevel []*Log, err error)

	// get a log according to a hierarchy of log.Name() references
	// for example, fetching HeadRef(ctx, "foo", "bar", "baz") is a request
	// for the log at the hierarchy foo/bar/baz:
	//   foo
	//     bar
	//       baz
	//
	// HeadRef must return ErrNotFound if any name in the heirarchy is  missing
	// from the store
	// Head references are mutated by adding operations to a log that modifies
	// the name of the initialization model, which means names are not a
	// persistent identifier
	//
	// HeadRef MAY return children of a log. If the returned log.Log value is
	// populated, it MUST contain all children of the log.
	// use Logstore.Children or Logstore.Descendants to populate missing children
	HeadRef(ctx context.Context, names ...string) (*Log, error)

	// get a log according to it's ID string
	// Log must return ErrNotFound if the ID does not exist
	//
	// Log MAY return children of a log. If the returned log.Log value is
	// populated, it MUST contain all children of the log.
	// use Logstore.Children or Logstore.Descendants to populate missing children
	Get(ctx context.Context, id string) (*Log, error)

	// GetAuthorID fetches the first log that matches the given model and
	// authorID. Note that AutorID is the initial operation AuthorID field,
	// not the init identifier
	GetAuthorID(ctx context.Context, model uint32, authorID string) (*Log, error)

	// get the immediate descendants of a log, using the given log as an outparam.
	// Children must only mutate Logs field of the passed-in log pointer
	// added Children MAY include decendant logs
	Children(ctx context.Context, l *Log) error
	// get all generations of a log, using the given log as an outparam
	// Descendants MUST only mutate Logs field of the passed-in log pointer
	Descendants(ctx context.Context, l *Log) error

	// ReplaceAll replaces the contents of the entire log
	ReplaceAll(ctx context.Context, l *Log) error
}

// SparseAncestorsAllDescendantsLogstore is a an extension interface to
// Logstore with an optimized method for getting a log with sparse parents and
// all descendants
type SparseAncestorsAllDescendantsLogstore interface {
	Logstore
	// GetSparseAncestorsAllDescendants is a fast-path method for getting a
	// log that includes sparse parents & complete descendants. "sparse parents"
	// have the only children given in parent specified
	// AllDescendants include
	// the returned log will match the ID of the request, with parents
	GetSparseAncestorsAllDescendants(ctx context.Context, id string) (*Log, error)
}

// GetWithSparseAncestorsAllDescendants is a fast-path method for getting a
// log that includes sparse parents & complete descendants. "sparse parents"
// means the only children given in parent
// the returned log will match the ID of the request, with parents
func GetWithSparseAncestorsAllDescendants(ctx context.Context, store Logstore, id string) (*Log, error) {
	if sparseLS, ok := store.(SparseAncestorsAllDescendantsLogstore); ok {
		return sparseLS.GetSparseAncestorsAllDescendants(ctx, id)
	}

	l, err := store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// TODO (b5) - this is error prone. logs may exist but not be fetched. This
	// check is here now because not all implementations support the Descendants
	// method properly.
	// in the future remove this check once all implementations we know of have
	// a working `Descendants` implementation
	if len(l.Logs) == 0 {
		if err = store.Descendants(ctx, l); err != nil {
			return nil, err
		}
	}

	cursor := l
	for cursor.ParentID != "" {
		parent := cursor.parent
		if parent == nil {
			if parent, err = store.Get(ctx, cursor.ParentID); err != nil {
				return nil, err
			}
		}

		// TODO (b5) - hack to carry signatures possibly stored on the child
		// this is incorrect long term, but has the effect of producing correct
		// signed logs. Need to switch to log-specific signatures ASAP
		if parent.Signature == nil {
			parent.Signature = cursor.Signature
		}

		cursor.parent = &Log{
			ParentID:  parent.ParentID,
			Ops:       parent.Ops,
			Logs:      []*Log{cursor},
			Signature: parent.Signature,
		}
		cursor = cursor.parent
	}

	return l, nil
}

// AuthorLogstore describes a store owned by a single author, it adds encryption
// methods for safe local persistence as well as owner ID accessors
type AuthorLogstore interface {
	// All AuthorLogstores are Logstores
	Logstore
	// marshals all logs to a slice of bytes encrypted with the given private key
	FlatbufferCipher(pk crypto.PrivKey) ([]byte, error)
	// decrypt flatbuffer bytes, re-hydrating the store
	UnmarshalFlatbufferCipher(ctx context.Context, pk crypto.PrivKey, ciphertext []byte) error
}

// Journal is a store of logs known to a single author, representing their
// view of an abstract dataset graph. journals live in memory by default, and
// can be encrypted for storage
type Journal struct {
	logs []*Log
}

// assert at compile time that a Journal pointer is an AuthorLogstore
var _ AuthorLogstore = (*Journal)(nil)

// MergeLog adds a log to the journal
func (j *Journal) MergeLog(ctx context.Context, incoming *Log) error {
	if incoming.ID() == "" {
		return fmt.Errorf("oplog: log ID cannot be empty")
	}

	// Find a log with a matching id
	found, err := j.Get(ctx, incoming.ID())
	if err == nil {
		// If found, merge it
		found.Merge(incoming)
		return nil
	} else if !errors.Is(err, ErrNotFound) {
		// Okay if log is not found by id, but any other error should be returned
		return err
	}

	// Find a user log with a matching profileID
	for _, lg := range j.logs {
		if lg.FirstOpAuthorID() == incoming.FirstOpAuthorID() {
			found = lg
			found.Merge(incoming)
			return nil
		}
	}

	// Append to the end
	j.logs = append(j.logs, incoming)
	return nil
}

// RemoveLog removes a log from the journal
// TODO (b5) - this currently won't work when trying to remove the root log
func (j *Journal) RemoveLog(ctx context.Context, names ...string) error {
	if len(names) == 0 {
		return fmt.Errorf("name is required")
	}

	remove := names[len(names)-1]
	parentPath := names[:len(names)-1]

	if len(parentPath) == 0 {
		for i, l := range j.logs {
			if l.Name() == remove {
				j.logs = append(j.logs[:i], j.logs[i+1:]...)
				return nil
			}
		}
		return ErrNotFound
	}

	parent, err := j.HeadRef(ctx, parentPath...)
	if err != nil {
		return err
	}

	// iterate list looking for log to remove
	for i, l := range parent.Logs {
		if l.Name() == remove {
			parent.Logs = append(parent.Logs[:i], parent.Logs[i+1:]...)
			return nil
		}
	}

	return ErrNotFound
}

// Get fetches a log for a given ID
func (j *Journal) Get(_ context.Context, id string) (*Log, error) {
	for _, lg := range j.logs {
		if l, err := lg.Log(id); err == nil {
			return l, nil
		}
	}
	return nil, ErrNotFound
}

// GetAuthorID fetches the first log that matches the given model and authorID
func (j *Journal) GetAuthorID(_ context.Context, model uint32, authorID string) (*Log, error) {
	for _, lg := range j.logs {
		// NOTE: old logbook entries erroneously used logbook identifiers in the AuthorID
		// space when they should have been using external author Identifiers. In the short
		// term we're relying on the fact that the 0th operation always uses an external
		// identifier
		if lg.Model() == model && lg.FirstOpAuthorID() == authorID {
			return lg, nil
		}
	}
	return nil, fmt.Errorf("getting log by author ID %q %w", authorID, ErrNotFound)
}

// HeadRef traverses the log graph & pulls out a log based on named head
// references
// HeadRef will not return logs that have been marked as removed. To fetch
// removed logs either traverse the entire journal or reference a log by ID
func (j *Journal) HeadRef(_ context.Context, names ...string) (*Log, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("name is required")
	}

	for _, log := range j.logs {
		if log.Name() == names[0] && !log.Removed() {
			return log.HeadRef(names[1:]...)
		}
	}
	return nil, ErrNotFound
}

// Logs returns the full map of logs keyed by model type
func (j *Journal) Logs(ctx context.Context, offset, limit int) (topLevel []*Log, err error) {
	// fast-path for no pagination
	if offset == 0 && limit == -1 {
		return j.logs[:], nil
	}

	return nil, fmt.Errorf("log subsets not finished")
}

// UnmarshalFlatbufferCipher decrypts and loads a flatbuffer ciphertext
func (j *Journal) UnmarshalFlatbufferCipher(ctx context.Context, pk crypto.PrivKey, ciphertext []byte) error {
	plaintext, err := j.decrypt(pk, ciphertext)
	if err != nil {
		return err
	}

	return j.unmarshalFlatbuffer(logfb.GetRootAsBook(plaintext, 0))
}

// Children gets all descentants of a log, because logbook stores all
// descendants in memory, children is a proxy for descenants
func (j *Journal) Children(ctx context.Context, l *Log) error {
	return j.Descendants(ctx, l)
}

// Descendants gets all descentants of a log & assigns the results to the given
// Log parameter, setting only the Logs field
func (j *Journal) Descendants(ctx context.Context, l *Log) error {
	got, err := j.Get(ctx, l.ID())
	if err != nil {
		return err
	}

	l.Logs = got.Logs
	return nil
}

// ReplaceAll replaces the entirety of the logs
func (j *Journal) ReplaceAll(ctx context.Context, l *Log) error {
	j.logs = []*Log{l}
	return nil
}

// FlatbufferCipher marshals journal to a flatbuffer and encrypts the book using
// a given private key. This same private key must be retained elsewhere to read
// the flatbuffer later on
func (j Journal) FlatbufferCipher(pk crypto.PrivKey) ([]byte, error) {
	return j.encrypt(pk, j.flatbufferBytes())
}

func (j Journal) cipher(pk crypto.PrivKey) (cipher.AEAD, error) {
	pkBytes, err := pk.Raw()
	if err != nil {
		return nil, err
	}
	hasher := md5.New()
	hasher.Write(pkBytes)
	hash := hex.EncodeToString(hasher.Sum(nil))

	block, err := aes.NewCipher([]byte(hash))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (j Journal) encrypt(pk crypto.PrivKey, data []byte) ([]byte, error) {
	gcm, err := j.cipher(pk)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func (j Journal) decrypt(pk crypto.PrivKey, data []byte) ([]byte, error) {
	gcm, err := j.cipher(pk)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// flatbufferBytes formats book as a flatbuffer byte slice
func (j Journal) flatbufferBytes() []byte {
	builder := flatbuffers.NewBuilder(0)
	off := j.marshalFlatbuffer(builder)
	builder.Finish(off)
	return builder.FinishedBytes()
}

// note: currently doesn't marshal book.author, we're considering deprecating
// the author field
func (j Journal) marshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {

	setsl := j.logs
	count := len(setsl)
	offsets := make([]flatbuffers.UOffsetT, count)
	for i, lset := range setsl {
		offsets[i] = lset.MarshalFlatbuffer(builder)
	}
	logfb.BookStartLogsVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	sets := builder.EndVector(count)

	logfb.BookStart(builder)
	logfb.BookAddLogs(builder, sets)
	return logfb.BookEnd(builder)
}

func (j *Journal) unmarshalFlatbuffer(b *logfb.Book) error {
	newBook := Journal{}

	count := b.LogsLength()
	lfb := &logfb.Log{}
	for i := 0; i < count; i++ {
		if b.Logs(lfb, i) {
			l := &Log{}
			if err := l.UnmarshalFlatbuffer(lfb, nil); err != nil {
				return err
			}
			newBook.logs = append(newBook.logs, l)
		}
	}

	*j = newBook
	return nil
}

// Log is a causally-ordered set of operations performed by a single author.
// log attribution is verified by an author's signature
type Log struct {
	name     string // name value cache. not persisted
	authorID string // authorID value cache. not persisted
	parent   *Log   // parent link

	ParentID  string // init id of parent Log
	Signature []byte
	Ops       []Op
	Logs      []*Log
}

// InitLog creates a Log from an initialization operation
func InitLog(initop Op) *Log {
	return &Log{
		Ops: []Op{initop},
	}
}

// FromFlatbufferBytes initializes a log from flatbuffer data
func FromFlatbufferBytes(data []byte) (*Log, error) {
	rootfb := logfb.GetRootAsLog(data, 0)
	lg := &Log{}
	return lg, lg.UnmarshalFlatbuffer(rootfb, nil)
}

// Append adds an operation to the log
func (lg *Log) Append(op Op) {
	if op.Model == lg.Model() {
		if op.Name != "" {
			lg.name = op.Name
		}
		if op.AuthorID != "" {
			lg.authorID = op.AuthorID
		}
	}
	lg.Ops = append(lg.Ops, op)
}

// ID returns the hash of the initialization operation
// if the log is empty, returns the empty string
func (lg Log) ID() string {
	if len(lg.Ops) == 0 {
		return ""
	}
	return lg.Ops[0].Hash()
}

// Head gets the latest operation in the log
func (lg Log) Head() Op {
	if len(lg.Ops) == 0 {
		return Op{}
	}
	return lg.Ops[len(lg.Ops)-1]
}

// Model gives the operation type for a log, based on the first operation
// written to the log. Logs can contain multiple models of operations, but the
// first operation written to a log determines the kind of log for
// catagorization purposes
func (lg Log) Model() uint32 {
	return lg.Ops[0].Model
}

// Author returns one of two different things: either the user's ProfileID,
// or the has of the first Op for the UserLog, depending on if they have
// ever changed their username.
func (lg Log) Author() (identifier string) {
	if lg.authorID == "" {
		m := lg.Model()
		for _, o := range lg.Ops {
			if o.Model == m && o.AuthorID != "" {
				lg.authorID = o.AuthorID
			}
		}
	}
	return lg.authorID
}

// FirstOpAuthorID returns the authorID of the first Op. For UserLog, this is ProfileID
func (lg Log) FirstOpAuthorID() string {
	m := lg.Model()
	for _, o := range lg.Ops {
		if o.Model == m && o.AuthorID != "" {
			return o.AuthorID
		}
	}
	return ""
}

// Parent returns this log's parent if one exists
func (lg *Log) Parent() *Log {
	return lg.parent
}

// Name returns the human-readable name for this log, determined by the
// initialization event
func (lg Log) Name() string {
	if lg.name == "" {
		m := lg.Model()
		for _, o := range lg.Ops {
			if o.Model == m && o.Name != "" {
				lg.name = o.Name
			}
		}
	}
	return lg.name
}

// Removed returns true if the log contains a remove operation for the log model
func (lg Log) Removed() bool {
	m := lg.Model()
	for _, op := range lg.Ops {
		if op.Model == m && op.Type == OpTypeRemove {
			return true
		}
	}
	return false
}

// DeepCopy produces a fresh duplicate of this log
func (lg *Log) DeepCopy() *Log {
	lg.FlatbufferBytes()
	cp := &Log{}
	if err := cp.UnmarshalFlatbufferBytes(lg.FlatbufferBytes()); err != nil {
		panic(err)
	}
	cp.ParentID = lg.ParentID
	return cp
}

// Log fetches a log by ID, checking the current log and all descendants for an
// exact match
func (lg *Log) Log(id string) (*Log, error) {
	if lg.ID() == id {
		return lg, nil
	}
	if len(lg.Logs) > 0 {
		for _, l := range lg.Logs {
			if got, err := l.Log(id); err == nil {
				return got, nil
			}
		}
	}
	return nil, ErrNotFound
}

// HeadRef returns a descendant log, traversing the log tree by name
// HeadRef will not return logs that have been marked as removed. To fetch
// removed logs either traverse the entire book or reference a log by ID
func (lg *Log) HeadRef(names ...string) (*Log, error) {
	if len(names) == 0 {
		return lg, nil
	}

	for _, log := range lg.Logs {
		if log.Name() == names[0] && !log.Removed() {
			return log.HeadRef(names[1:]...)
		}
	}
	return nil, ErrNotFound
}

// AddChild appends a log as a direct descendant of this log, controlling
// for duplicates
func (lg *Log) AddChild(l *Log) {
	l.ParentID = lg.ID()
	l.parent = lg
	for i, ch := range lg.Logs {
		if ch.ID() == l.ID() {
			if len(l.Ops) > len(ch.Ops) {
				lg.Logs[i] = l
			}
			return
		}
	}
	lg.Logs = append(lg.Logs, l)
}

// Merge combines two logs that are assumed to be a shared root, combining
// children from both branches, matching branches prefer longer Opsets
// Merging relies on comparison of initialization operations, which
// must be present to constitute a match
func (lg *Log) Merge(l *Log) {
	// if the incoming log has more operations, use it & clear the cache
	if len(l.Ops) > len(lg.Ops) {
		lg.Ops = l.Ops
		lg.name = ""
		lg.authorID = ""
		lg.Signature = nil
	}

LOOP:
	for _, x := range l.Logs {
		for j, y := range lg.Logs {
			// if logs match. merge 'em
			if x.Ops[0].Equal(y.Ops[0]) {
				lg.Logs[j].Merge(x)
				continue LOOP
			}
		}
		// no match, append!
		lg.AddChild(x)
	}
}

// Verify confirms that the signature for a log matches
func (lg Log) Verify(pub crypto.PubKey) error {
	ok, err := pub.Verify(lg.SigningBytes(), lg.Signature)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

// Sign assigns the log signature by signing the logging checksum with a given
// private key
// TODO (b5) - this is assuming the log is authored by this private key. as soon
// as we add collaborators, this won't be true
func (lg *Log) Sign(pk crypto.PrivKey) (err error) {
	lg.Signature, err = pk.Sign(lg.SigningBytes())
	if err != nil {
		return err
	}

	return nil
}

// SigningBytes perpares a byte slice for signing from a log's operations
func (lg Log) SigningBytes() []byte {
	hasher := md5.New()
	for _, op := range lg.Ops {
		hasher.Write([]byte(op.Ref))
	}
	return hasher.Sum(nil)
}

// FlatbufferBytes marshals a log to flabuffer-formatted bytes
func (lg Log) FlatbufferBytes() []byte {
	builder := flatbuffers.NewBuilder(0)
	log := lg.MarshalFlatbuffer(builder)
	builder.Finish(log)
	return builder.FinishedBytes()
}

// MarshalFlatbuffer writes log to a flatbuffer, returning the ending byte
// offset
func (lg Log) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// build logs bottom up, collecting offsets
	logcount := len(lg.Logs)
	logoffsets := make([]flatbuffers.UOffsetT, logcount)
	for i, o := range lg.Logs {
		logoffsets[i] = o.MarshalFlatbuffer(builder)
	}

	logfb.LogStartLogsVector(builder, logcount)
	for i := logcount - 1; i >= 0; i-- {
		builder.PrependUOffsetT(logoffsets[i])
	}
	logs := builder.EndVector(logcount)

	name := builder.CreateString(lg.Name())
	id := builder.CreateString(lg.Author())
	signature := builder.CreateByteString(lg.Signature)

	count := len(lg.Ops)
	offsets := make([]flatbuffers.UOffsetT, count)
	for i, o := range lg.Ops {
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
	logfb.LogAddLogs(builder, logs)
	return logfb.LogEnd(builder)
}

// UnmarshalFlatbufferBytes is a convenince wrapper to deserialze a flatbuffer
// slice into a log
func (lg *Log) UnmarshalFlatbufferBytes(data []byte) error {
	return lg.UnmarshalFlatbuffer(logfb.GetRootAsLog(data, 0), nil)
}

// UnmarshalFlatbuffer populates a logfb.Log from a Log pointer
func (lg *Log) UnmarshalFlatbuffer(lfb *logfb.Log, parent *Log) (err error) {
	newLg := Log{parent: parent}
	if parent != nil {
		newLg.ParentID = parent.ID()
	}

	if len(lfb.Signature()) != 0 {
		newLg.Signature = lfb.Signature()
	}

	newLg.Ops = make([]Op, lfb.OpsetLength())
	opfb := &logfb.Operation{}
	for i := 0; i < lfb.OpsetLength(); i++ {
		if lfb.Opset(opfb, i) {
			newLg.Ops[i] = UnmarshalOpFlatbuffer(opfb)
		}
	}

	if lfb.LogsLength() > 0 {
		newLg.Logs = make([]*Log, lfb.LogsLength())
		childfb := &logfb.Log{}
		for i := 0; i < lfb.LogsLength(); i++ {
			if lfb.Logs(childfb, i) {
				newLg.Logs[i] = &Log{}
				newLg.Logs[i].UnmarshalFlatbuffer(childfb, lg)
				newLg.Logs[i].ParentID = newLg.ID()
			}
		}
	}

	*lg = newLg
	return nil
}

// OpType is the set of all kinds of operations, they are two bytes in length
// OpType splits the provided byte in half, using the higher 4 bits for the
// "category" of operation, and the lower 4 bits for the type of operation
// within the category
// the second byte is reserved for future use
type OpType byte

const (
	// OpTypeInit is the creation of a model
	OpTypeInit OpType = 0x01
	// OpTypeAmend represents amending a model
	OpTypeAmend OpType = 0x02
	// OpTypeRemove represents deleting a model
	OpTypeRemove OpType = 0x03
)

// Op is an operation, a single atomic unit in a log that describes a state
// change
type Op struct {
	Type      OpType   // type of operation
	Model     uint32   // data model to operate on
	Ref       string   // identifier of data this operation is documenting
	Prev      string   // previous reference in a causal history
	Relations []string // references this operation relates to. usage is operation type-dependant
	Name      string   // human-readable name for the reference
	AuthorID  string   // identifier for author

	Timestamp int64  // operation timestamp, for annotation purposes only
	Size      int64  // size of the referenced value in bytes
	Note      string // operation annotation for users. eg: commit title
}

// Equal tests equality between two operations
func (o Op) Equal(b Op) bool {
	return o.Type == b.Type &&
		o.Model == b.Model &&
		o.Ref == b.Ref &&
		o.Prev == b.Prev &&
		len(o.Relations) == len(b.Relations) &&
		o.Name == b.Name &&
		o.AuthorID == b.AuthorID &&
		o.Timestamp == b.Timestamp &&
		o.Size == b.Size &&
		o.Note == b.Note
}

// Hash uses lower-case base32 encoding for id bytes for a few reasons:
// * base64 uses the "/" character, which messes with paths
// * can be used as URLS
// * doesn't rely on case, which means it works in case-insensitive contexts
// * lowercase is easier on the eyes
//
// we're intentionally *not* using multiformat CIDs here. ID's are not
// identifiers of content stored wholly in an immutable filesystem, they're a
// reference to the intialization operation in a history
var base32Enc = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

// Hash returns the base32-lowercase-encoded blake2b-256 hash of the Op flatbuffer
func (o Op) Hash() string {
	builder := flatbuffers.NewBuilder(0)
	end := o.MarshalFlatbuffer(builder)
	builder.Finish(end)
	data := builder.FinishedBytes()
	sum := blake2b.Sum256(data)
	return base32Enc.EncodeToString(sum[:])
}

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o Op) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	ref := builder.CreateString(o.Ref)
	prev := builder.CreateString(o.Prev)
	name := builder.CreateString(o.Name)
	authorID := builder.CreateString(o.AuthorID)
	note := builder.CreateString(o.Note)

	count := len(o.Relations)
	offsets := make([]flatbuffers.UOffsetT, count)
	for i, r := range o.Relations {
		offsets[i] = builder.CreateString(r)
	}

	logfb.OperationStartRelationsVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	rels := builder.EndVector(count)

	logfb.OperationStart(builder)
	logfb.OperationAddType(builder, logfb.OpType(o.Type))
	logfb.OperationAddModel(builder, o.Model)
	logfb.OperationAddRef(builder, ref)
	logfb.OperationAddRelations(builder, rels)
	logfb.OperationAddPrev(builder, prev)
	logfb.OperationAddName(builder, name)
	logfb.OperationAddAuthorID(builder, authorID)
	logfb.OperationAddTimestamp(builder, o.Timestamp)
	logfb.OperationAddSize(builder, o.Size)
	logfb.OperationAddNote(builder, note)
	return logfb.OperationEnd(builder)
}

// UnmarshalOpFlatbuffer creates an op from a flatbuffer operation pointer
func UnmarshalOpFlatbuffer(o *logfb.Operation) Op {
	op := Op{
		Type:      OpType(byte(o.Type())),
		Model:     o.Model(),
		Timestamp: o.Timestamp(),
		Ref:       string(o.Ref()),
		Prev:      string(o.Prev()),
		Name:      string(o.Name()),
		AuthorID:  string(o.AuthorID()),
		Size:      o.Size(),
		Note:      string(o.Note()),
	}

	if o.RelationsLength() > 0 {
		op.Relations = make([]string, o.RelationsLength())
		for i := 0; i < o.RelationsLength(); i++ {
			op.Relations[i] = string(o.Relations(i))
		}
	}

	return op
}
