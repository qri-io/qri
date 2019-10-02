// Package log is an operation-based conflict-free replicated data type (CRDT)
// of append-only logs. A log is a sequence of operations attributed to a single
// author, designated by a private key.
// Each operation is a record of some action the author took. Applications
// iterate the sequence of operations to produce the current state.
// Logs can be arranged into hierarchies to form logical groupings.
// A book contains an author's logs, both logs they've written as well as logs
// replicated from other authors. Books are encrypted at rest using the author
// private key.
package log

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	flatbuffers "github.com/google/flatbuffers/go"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/qri/logbook/log/logfb"
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
	authorname string
	id         string
	pk         crypto.PrivKey
	logs       map[uint32][]*Log
}

// NewBook initializes a Book
func NewBook(pk crypto.PrivKey, authorname, authorID string) (*Book, error) {
	return &Book{
		pk:         pk,
		authorname: authorname,
		logs:       map[uint32][]*Log{},
	}, nil
}

// AuthorName gives the human-readable name of the author
func (book Book) AuthorName() string {
	return book.authorname
}

// AuthorID returns the machine identifier for a name
func (book Book) AuthorID() string {
	return book.id
}

// AppendLog adds a log to a book
func (book *Book) AppendLog(l *Log) {
	book.logs[l.Model()] = append(book.logs[l.Model()], l)
}

// ModelLogs gives all sets whoe model type matches model
func (book *Book) ModelLogs(model uint32) []*Log {
	return book.logs[model]
}

// UnmarshalFlatbufferCipher decrypts and loads a flatbuffer ciphertext
func (book *Book) UnmarshalFlatbufferCipher(ctx context.Context, ciphertext []byte) error {
	plaintext, err := book.decrypt(ciphertext)
	if err != nil {
		return err
	}

	return book.unmarshalFlatbuffer(logfb.GetRootAsBook(plaintext, 0))
}

// FlatbufferCipher marshals book to a flatbuffer and encrypts the book using
// the book private key
func (book Book) FlatbufferCipher() ([]byte, error) {
	return book.encrypt(book.flatbufferBytes())
}

func (book Book) cipher() (cipher.AEAD, error) {
	pkBytes, err := book.pk.Raw()
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

func (book Book) encrypt(data []byte) ([]byte, error) {
	gcm, err := book.cipher()
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

func (book Book) decrypt(data []byte) ([]byte, error) {
	gcm, err := book.cipher()
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

func (book Book) marshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	authorname := builder.CreateString(book.authorname)
	id := builder.CreateString(book.id)

	setsl := book.logsSlice()
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
	logfb.BookAddName(builder, authorname)
	logfb.BookAddIdentifier(builder, id)
	logfb.BookAddLogs(builder, sets)
	return logfb.BookEnd(builder)
}

func (book *Book) unmarshalFlatbuffer(b *logfb.Book) error {
	newBook := Book{
		pk:         book.pk,
		authorname: string(b.Name()),
		id:         string(b.Identifier()),
		logs:       map[uint32][]*Log{},
	}

	count := b.LogsLength()
	lfb := &logfb.Log{}
	for i := 0; i < count; i++ {
		if b.Logs(lfb, i) {
			l := &Log{}
			if err := l.UnmarshalFlatbuffer(lfb); err != nil {
				return err
			}
			newBook.logs[l.Model()] = append(newBook.logs[l.Model()], l)
		}
	}

	*book = newBook
	return nil
}

func (book Book) logsSlice() (logs []*Log) {
	for _, logsl := range book.logs {
		logs = append(logs, logsl...)
	}
	return logs
}

// Log is a causally-ordered set of operations performed by a single author.
// log attribution is verified by an author's signature
type Log struct {
	signature []byte
	ops       []Op
	logs      []*Log
}

// InitLog creates a Log from an initialization operation
func InitLog(initop Op) *Log {
	return &Log{
		ops: []Op{initop},
	}
}

// FromFlatbufferBytes initializes a log from flatbuffer data
func FromFlatbufferBytes(data []byte) (*Log, error) {
	rootfb := logfb.GetRootAsLog(data, 0)
	lg := &Log{}
	return lg, lg.UnmarshalFlatbuffer(rootfb)
}

// Append adds an operation to the log
func (lg *Log) Append(op Op) {
	lg.ops = append(lg.ops, op)
}

// Len returns the number of of the latest entry in the log
func (lg Log) Len() int {
	return len(lg.ops)
}

// Model gives the operation type for a log, based on the first operation
// written to the log. Logs can contain multiple models of operations, but the
// first operation written to a log determines the kind of log for
// catagorization purposes
func (lg Log) Model() uint32 {
	return lg.ops[0].Model
}

// Author returns the name and identifier this log is attributed to
func (lg Log) Author() (name, identifier string) {
	return lg.ops[0].Name, lg.ops[0].AuthorID
}

// Name returns the human-readable name for this log, determined by the
// initialization event
// TODO (b5) - name must be made mutable by playing forward any name-changing
// operations and applying them to the log
func (lg Log) Name() string {
	return lg.ops[0].Name
}

// Child returns a child log for a given name, and nil if it doesn't exist
func (lg Log) Child(name string) *Log {
	for _, l := range lg.logs {
		if l.Name() == name {
			return l
		}
	}
	return nil
}

// AddChild appends a log as a direct descendant of this log
func (lg *Log) AddChild(l *Log) {
	lg.logs = append(lg.logs, l)
}

// Verify confirms that the signature for a log matches
func (lg Log) Verify(pub crypto.PubKey) error {
	ok, err := pub.Verify(lg.SigningBytes(), lg.signature)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

// Ops gives the set of operations in a log
func (lg Log) Ops() []Op {
	return lg.ops
}

// Sign assigns the log signature by signing the logging checksum with a given
// private key
// TODO (b5) - this is assuming the log is authored by this private key. as soon
// as we add collaborators, this won't be true
func (lg *Log) Sign(pk crypto.PrivKey) (err error) {
	lg.signature, err = pk.Sign(lg.SigningBytes())
	if err != nil {
		return err
	}

	return nil
}

// SigningBytes perpares a byte slice for signing from a log's operations
func (lg Log) SigningBytes() []byte {
	hasher := md5.New()
	for _, op := range lg.ops {
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
	logcount := len(lg.logs)
	logoffsets := make([]flatbuffers.UOffsetT, logcount)
	for i, o := range lg.logs {
		logoffsets[i] = o.MarshalFlatbuffer(builder)
	}

	logfb.LogStartLogsVector(builder, logcount)
	for i := logcount - 1; i >= 0; i-- {
		builder.PrependUOffsetT(logoffsets[i])
	}
	logs := builder.EndVector(logcount)

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
	logfb.LogAddLogs(builder, logs)
	return logfb.LogEnd(builder)
}

// UnmarshalFlatbuffer reads a Log from
func (lg *Log) UnmarshalFlatbuffer(lfb *logfb.Log) (err error) {
	newLg := Log{}

	if len(lfb.Signature()) != 0 {
		newLg.signature = lfb.Signature()
	}

	newLg.ops = make([]Op, lfb.OpsetLength())
	opfb := &logfb.Operation{}
	for i := 0; i < lfb.OpsetLength(); i++ {
		if lfb.Opset(opfb, i) {
			newLg.ops[i] = UnmarshalOpFlatbuffer(opfb)
		}
	}

	if lfb.LogsLength() > 0 {
		newLg.logs = make([]*Log, lfb.LogsLength())
		childfb := &logfb.Log{}
		for i := 0; i < lfb.LogsLength(); i++ {
			if lfb.Logs(childfb, i) {
				newLg.logs[i] = &Log{}
				newLg.logs[i].UnmarshalFlatbuffer(childfb)
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
	Size      uint64 // size of the referenced value in bytes
	Note      string // operation annotation for users. eg: commit title
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
