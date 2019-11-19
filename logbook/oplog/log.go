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

// Logstore persists a set of Logs
type Logstore interface {
	// Add a Log to the store. If the given log has children, the store must
	// persist those logs as well
	// if the given log specifies a parent, the log must be stored as a child of
	// that log, and error if no such log exists.
	// TODO (b5) - this store-by-parent rule is neither being used or enforced
	AppendLog(ctx context.Context, l *Log) error

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
	Log(ctx context.Context, id string) (*Log, error)

	// get the immediate descendants of a log, using the given log as an outparam.
	// Children must only mutate Logs field of the passed-in log pointer
	// added Children MAY include decendant logs
	Children(ctx context.Context, l *Log) error
	// get all generations of a log, using the given log as an outparam
	// Descendants MUST only mutate Logs field of the passed-in log pointer
	Descendants(ctx context.Context, l *Log) error
}

// AuthorLogstore adds encryption methods to the Logstore interface owned by a
// single author
type AuthorLogstore interface {
	// All AuthorLogstores are Logstores
	Logstore

	// get the id of the oplog that represnts this Logstore's author
	ID() string
	// attribute ownership of the logstore to an author
	// the given id MUST equal the id of a log already in the logstore
	SetID(id string) error

	// marshals all logs to a slice of bytes encrypted with the given private key
	FlatbufferCipher(pk crypto.PrivKey) ([]byte, error)
	// decrypt flatbuffer bytes, re-hydrating the store
	UnmarshalFlatbufferCipher(ctx context.Context, pk crypto.PrivKey, ciphertext []byte) error
}

// Book is a journal of operations organized into a collection of append-only
// logs. Each log is single-writer
// Books are connected to a single author, and represent their view of
// the global dataset graph.
// Any write operation performed on the logbook are attributed to a single
// author, denoted by a private key. Books can replicate logs from other
// authors, forming a conflict-free replicated data type (CRDT), and a basis
// for collaboration through knowledge of each other's operations
type Book struct {
	id   string
	logs []*Log
}

// assert at compile time that a Book pointer is an AuthorLogstore
var _ AuthorLogstore = (*Book)(nil)

// NewBook initializes a Book
func NewBook(authorID string) (*Book, error) {
	return &Book{
		id: authorID,
	}, nil
}

// ID gets the book identifier
func (book *Book) ID() string {
	return book.id
}

// SetID assigns the book identifier
func (book *Book) SetID(id string) {
	book.id = id
}

// AppendLog adds a log to a book
func (book *Book) AppendLog(_ context.Context, l *Log) error {
	book.logs = append(book.logs, l)
	return nil
}

// RemoveLog removes a log from the book
// TODO (b5) - this currently won't work when trying to remove the root log
func (book *Book) RemoveLog(ctx context.Context, names ...string) error {
	if len(names) == 0 {
		return fmt.Errorf("name is required")
	}

	remove := names[len(names)-1]
	parentPath := names[:len(names)-1]

	if len(parentPath) == 0 {
		for i, l := range book.logs {
			if l.Name() == remove {
				book.logs = append(book.logs[:i], book.logs[i+1:]...)
				return nil
			}
		}
		return ErrNotFound
	}

	parent, err := book.HeadRef(ctx, parentPath...)
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

// Log fetches a log for a given ID
func (book *Book) Log(_ context.Context, id string) (*Log, error) {
	for _, lg := range book.logs {
		if l, err := lg.Log(id); err == nil {
			return l, nil
		}
	}
	return nil, ErrNotFound
}

// HeadRef traverses the log graph & pulls out a log based on named head
// references
// HeadRef will not return logs that have been marked as removed. To fetch
// removed logs either traverse the entire book or reference a log by ID
func (book *Book) HeadRef(_ context.Context, names ...string) (*Log, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("name is required")
	}

	for _, log := range book.logs {
		if log.Name() == names[0] && !log.Removed() {
			return log.HeadRef(names[1:]...)
		}
	}
	return nil, ErrNotFound
}

// Logs returns the full map of logs keyed by model type
func (book *Book) Logs(ctx context.Context, offset, limit int) (topLevel []*Log, err error) {
	// fast-path for no pagination
	if offset == 0 && limit == -1 {
		return book.logs[:], nil
	}

	return nil, fmt.Errorf("log subsets not finished")
}

// UnmarshalFlatbufferCipher decrypts and loads a flatbuffer ciphertext
func (book *Book) UnmarshalFlatbufferCipher(ctx context.Context, pk crypto.PrivKey, ciphertext []byte) error {
	plaintext, err := book.decrypt(pk, ciphertext)
	if err != nil {
		return err
	}

	return book.unmarshalFlatbuffer(logfb.GetRootAsBook(plaintext, 0))
}

// Children gets all descentants of a log, because logbook stores all
// descendants in memory, children is a proxy for descenants
func (book *Book) Children(ctx context.Context, l *Log) error {
	return book.Descendants(ctx, l)
}

// Descendants gets all descentants of a log & assigns the results to the given
// Log parameter, setting only the Logs field
func (book *Book) Descendants(ctx context.Context, l *Log) error {
	got, err := book.Log(ctx, l.ID())
	if err != nil {
		return err
	}

	l.Logs = got.Logs
	return nil
}

// FlatbufferCipher marshals book to a flatbuffer and encrypts the book using
// a given private key. This same private key must be retained elsewhere to read
// the flatbuffer later on
func (book Book) FlatbufferCipher(pk crypto.PrivKey) ([]byte, error) {
	return book.encrypt(pk, book.flatbufferBytes())
}

func (book Book) cipher(pk crypto.PrivKey) (cipher.AEAD, error) {
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

func (book Book) encrypt(pk crypto.PrivKey, data []byte) ([]byte, error) {
	gcm, err := book.cipher(pk)
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

func (book Book) decrypt(pk crypto.PrivKey, data []byte) ([]byte, error) {
	gcm, err := book.cipher(pk)
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
func (book Book) flatbufferBytes() []byte {
	builder := flatbuffers.NewBuilder(0)
	off := book.marshalFlatbuffer(builder)
	builder.Finish(off)
	return builder.FinishedBytes()
}

// note: currently doesn't marshal book.author, we're considering deprecating
// the author field
func (book Book) marshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	id := builder.CreateString(book.id)

	setsl := book.logs
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
	logfb.BookAddIdentifier(builder, id)
	logfb.BookAddLogs(builder, sets)
	return logfb.BookEnd(builder)
}

func (book *Book) unmarshalFlatbuffer(b *logfb.Book) error {
	newBook := Book{
		id: string(b.Identifier()),
	}

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

	*book = newBook
	return nil
}

// Log is a causally-ordered set of operations performed by a single author.
// log attribution is verified by an author's signature
type Log struct {
	name     string // name value cache. not persisted
	authorID string // authorID value cache. not persisted
	parent   *Log   // parent link

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

// Author returns the name and identifier this log is attributed to
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

// AddChild appends a log as a direct descendant of this log
func (lg *Log) AddChild(l *Log) {
	l.parent = lg
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
		lg.Logs = append(lg.Logs, x)
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

// SignedFlatbufferBytes signs a log then marshals it to a flatbuffer
func (lg Log) SignedFlatbufferBytes(pk crypto.PrivKey) ([]byte, error) {
	if err := lg.Sign(pk); err != nil {
		return nil, err
	}

	builder := flatbuffers.NewBuilder(0)
	log := lg.MarshalFlatbuffer(builder)
	builder.Finish(log)
	return builder.FinishedBytes(), nil
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
