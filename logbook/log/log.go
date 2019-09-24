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
	sets       map[uint32][]*Set
}

// NewBook initializes a Book
func NewBook(pk crypto.PrivKey, authorname, authorID string) (*Book, error) {
	return &Book{
		pk:         pk,
		authorname: authorname,
		sets:       map[uint32][]*Set{},
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

// AppendSet adds a set to a book
func (book *Book) AppendSet(set *Set) {
	book.sets[set.Model()] = append(book.sets[set.Model()], set)
}

// ModelSets gives all sets whoe model type matches model
func (book *Book) ModelSets(model uint32) []*Set {
	return book.sets[model]
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

	setsl := book.setsSlice()
	count := len(setsl)
	offsets := make([]flatbuffers.UOffsetT, count)
	for i, lset := range setsl {
		offsets[i] = lset.MarshalFlatbuffer(builder)
	}
	logfb.BookStartSetsVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	sets := builder.EndVector(count)

	logfb.BookStart(builder)
	logfb.BookAddName(builder, authorname)
	logfb.BookAddIdentifier(builder, id)
	logfb.BookAddSets(builder, sets)
	return logfb.BookEnd(builder)
}

func (book *Book) unmarshalFlatbuffer(b *logfb.Book) error {
	newBook := Book{
		id:   string(b.Identifier()),
		sets: map[uint32][]*Set{},
	}

	count := b.SetsLength()
	logsetfb := &logfb.Logset{}
	for i := 0; i < count; i++ {
		if b.Sets(logsetfb, i) {
			set := &Set{}
			if err := set.UnmarshalFlatbuffer(logsetfb); err != nil {
				return err
			}
			newBook.sets[set.Model()] = append(newBook.sets[set.Model()], set)
		}
	}

	*book = newBook
	return nil
}

func (book Book) setsSlice() (sets []*Set) {
	for _, setsl := range book.sets {
		sets = append(sets, setsl...)
	}
	return sets
}

// Set is a collection of logs
type Set struct {
	signer    string
	signature []byte
	root      string
	logs      map[string]*Log
}

// InitSet creates a Log from an initialization operation
func InitSet(name string, initop Op) *Set {
	lg := InitLog(initop)
	return &Set{
		root: name,
		logs: map[string]*Log{
			name: lg,
		},
	}
}

// NewSet creates a set from a given log, rooted at the set name
func NewSet(lg *Log) *Set {
	name := lg.Name()
	return &Set{
		root: name,
		logs: map[string]*Log{
			name: lg,
		},
	}
}

// Author gives authorship information about who created this logset
func (ls Set) Author() (string, string) {
	// TODO (b5) - fetch from master branch intiailization
	return "", ""
}

// Model returns the model of the root log
func (ls Set) Model() uint32 {
	return ls.logs[ls.root].ops[0].Model
}

// RootName gives the name of the root branch
func (ls Set) RootName() string {
	return ls.root
}

// Log returns a log from the set for a given name
func (ls Set) Log(name string) *Log {
	return ls.logs[name]
}

// MarshalFlatbuffer writes the set to a flatbuffer builder
func (ls Set) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
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

// UnmarshalFlatbuffer creates a set from a logset pointer
func (ls *Set) UnmarshalFlatbuffer(lsfb *logfb.Logset) (err error) {
	newLs := Set{
		root: string(lsfb.Root()),
		logs: map[string]*Log{},
	}

	lgfb := &logfb.Log{}
	for i := 0; i < lsfb.LogsLength(); i++ {
		if lsfb.Logs(lgfb, i) {
			lg := &Log{}
			if err = lg.UnmarshalFlatbuffer(lgfb); err != nil {
				return err
			}
			newLs.logs[lg.Name()] = lg
		}
	}

	*ls = newLs
	return nil
}

// Log is a causally-ordered set of operations performed by a single author.
// log attribution is verified by an author's signature
type Log struct {
	signature []byte
	ops       []Op
}

// InitLog creates a Log from an initialization operation
func InitLog(initop Op) *Log {
	return &Log{
		ops: []Op{initop},
	}
}

// Append adds an operation to the log
func (lg *Log) Append(op Op) {
	lg.ops = append(lg.ops, op)
}

// Len returns the number of of the latest entry in the log
func (lg Log) Len() int {
	return len(lg.ops)
}

// Type gives the operation type for a log, based on the first operation written
// to the log. Logs can contain multiple types of operations, but the first
// operation written to a log determines the kind of log for catagorization
// purposes
func (lg Log) Type() string {
	return ""
}

// Author returns the name and identifier this log is attributed to
func (lg Log) Author() (name, identifier string) {
	// if len(lg.ops) > 0 {
	// 	if initOp, ok := lg.ops[0].(InitOperation); ok {
	// 		return initOp.AuthorName(), initOp.AuthorID()
	// 	}
	// }
	return lg.ops[0].Name, lg.ops[0].AuthorID
}

// Name returns the human-readable name for this log, determined by the
// initialization event
// TODO (b5) - name must be made mutable by playing forward any name-changing
// operations and applying them to the log
func (lg Log) Name() string {
	// if len(lg.ops) > 0 {
	// 	if initOp, ok := lg.ops[0].(InitOperation); ok {
	// 		return initOp.Name()
	// 	}
	// }
	return lg.ops[0].Name
}

// Verify confirms that the signature for a log matches
func (lg Log) Verify() error {
	return fmt.Errorf("not finished")
}

// MarshalFlatbuffer writes log to a flatbuffer, returning the ending byte
// offset
func (lg Log) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
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
