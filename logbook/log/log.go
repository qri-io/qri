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
func InitSet(name string, initop InitOperation) *Set {
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
	ops       []Operation
}

// InitLog creates a Log from an initialization operation
func InitLog(initop InitOperation) *Log {
	return &Log{
		ops: []Operation{initop},
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
	// TODO (b5) - name and identifier must come from init operation
	if len(lg.ops) > 0 {
		if initOp, ok := lg.ops[0].(InitOperation); ok {
			return initOp.AuthorName(), initOp.AuthorID()
		}
	}
	return "", ""
}

// Name returns the human-readable name for this log, determined by the
// initialization event
// TODO (b5) - name must be made mutable by playing forward any name-changing
// operations and applying them to the log
func (lg Log) Name() string {
	if len(lg.ops) > 0 {
		if initOp, ok := lg.ops[0].(InitOperation); ok {
			return initOp.Name()
		}
	}
	return ""
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
	newLg := Log{
		signature: lfb.Signature(),
	}

	newLg.ops = make([]Operation, lfb.OpsetLength())
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
