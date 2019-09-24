package log

import (
	"fmt"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/qri-io/qri/logbook/logfb"
)

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

// Author gives authorship information about who created this logset
func (ls Set) Author() (string, string) {
	// TODO (b5) - fetch from master branch intiailization
	return "", ""
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

// MarshalFlatbuffer writes log to a flatbuffer, returning the
// ending byte offset
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
