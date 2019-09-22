package log

import (
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/qri-io/qri/log/logfb"
)

// opType is the set of kinds of operations
// OpType splits the provided byte in half, using the higher 4 bits for the
// "category" of operation, and the lower 4 bits for the type of operation
// within the category
// the second byte is reserved for future use
type opType uint16

const (
	opTypeUserInit   opType = 0x0000
	opTypeUserChange opType = 0x0100
	opTypeUserDelete opType = 0x0200
	opTypeUserRename opType = 0x0300

	opTypeNameInit   opType = 0x1000
	opTypeNameChange opType = 0x1100
	opTypeNameDelete opType = 0x1200

	opTypeVersionSave      opType = 0x2000
	opTypeVersionDelete    opType = 0x2100
	opTypeVersionPublish   opType = 0x2200
	opTypeVersionUnpublish opType = 0x2300

	opTypeACLInit   opType = 0x3000
	opTypeACLUpdate opType = 0x3100
	opTypeACLDelete opType = 0x3200
)

type operation interface {
	OpType() opType
	Timestamp() int64
	Ref() string
	MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT
}

type initOperation interface {
	operation
	Name() string
	AuthorName() string
	AuthorID() string
}

type op struct {
	opType    opType
	timestamp int64
	ref       string
}

func (o op) OpType() opType   { return o.opType }
func (o op) Timestamp() int64 { return o.timestamp }
func (o op) Ref() string      { return o.ref }

// userInit signifies the creation of a user
type userInit struct {
	op
	Username string
}

func (o userInit) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	ref := builder.CreateString(o.ref)
	username := builder.CreateString(o.Username)

	logfb.OperationStart(builder)
	logfb.OperationAddType(builder, uint16(o.opType))
	logfb.OperationAddTimestamp(builder, o.timestamp)
	logfb.OperationAddRef(builder, ref)
	logfb.OperationAddName(builder, username)
	return logfb.OperationEnd(builder)
}

func (o userInit) Name() string {
	return o.Username
}

func (o userInit) AuthorName() string {
	return o.Username
}

func (o userInit) AuthorID() string {
	return o.ref
}

func newUserInitFlatbuffer(o *logfb.Operation) userInit {
	return userInit{
		op: op{
			opType:    opTypeUserInit,
			timestamp: o.Timestamp(),
			ref:       string(o.Ref()),
		},
		Username: string(o.Name()),
	}
}

// userChange signifies a change in any user details that aren't
// a username
type userChange struct {
	op
	Note string
}

func (o userChange) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// userRename signifies a user has changed their username
type userRename struct {
	op
	Username string
}

func (o userRename) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// userDelete signifies user details have been deleted
type userDelete struct {
	op
	Author string
}

func (o userDelete) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// nameInit signifies dataset name creation in a user's namespace
type nameInit struct {
	op
	Author   string
	Username string
	Name     string
}

func (o nameInit) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	author := builder.CreateString(o.Author)
	username := builder.CreateString(o.Username)

	logfb.OperationStart(builder)
	logfb.OperationAddType(builder, uint16(o.opType))
	// TODO (b5):
	logfb.OperationAddTimestamp(builder, 0)
	logfb.OperationAddRef(builder, author)
	logfb.OperationAddName(builder, username)
	return logfb.OperationEnd(builder)
}

// nameChange signifies a dataset name change
type nameChange struct {
	op
	Name string
}

func (o nameChange) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// nameDelete signifies a dataset name has been deleted
type nameDelete struct {
	op
	Name string
}

func (o nameDelete) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// versionSave signifies creating one new dataset version
type versionSave struct {
	op
	Prev string
	Size uint64
	Note string
}

func (o versionSave) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	prev := builder.CreateString(o.Prev)
	ref := builder.CreateString(o.ref)
	note := builder.CreateString(o.Note)

	logfb.OperationStart(builder)
	logfb.OperationAddType(builder, uint16(o.opType))
	// TODO (b5):
	logfb.OperationAddTimestamp(builder, 0)
	logfb.OperationAddRef(builder, ref)
	logfb.OperationAddPrev(builder, prev)
	logfb.OperationAddNote(builder, note)
	return logfb.OperationEnd(builder)
}

// versionDelete signifies deleting one or more versions of a dataset
type versionDelete struct {
	op
	Revisions int
}

func (o versionDelete) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// versionPublish signifies publishing one or more sequential versions of a
// dataset
type versionPublish struct {
	op
	Revisions   int
	Destination string
}

func (o versionPublish) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// versionUnpublish signifies unpublishing one or more sequential versions of a
// dataset
type versionUnpublish struct {
	op
	Revisions   int
	Destination string
}

func (o versionUnpublish) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// aclInit signifies intializing an access control list
type aclInit struct {
	op
	Prev string
	Size uint64
	Note string
}

func (o aclInit) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// aclUpdate signifies a change to an access control list
type aclUpdate struct {
	op
	Prev string
	Size uint64
	Note string
}

func (o aclUpdate) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// aclDelete signifies removing an access control list
type aclDelete struct {
	op
	Prev string
}

func (o aclDelete) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

func unmarshalOperationFlatbuffer(opfb *logfb.Operation) (opr operation, err error) {
	switch opType(opfb.Type()) {
	case opTypeUserInit:
		opr = newUserInitFlatbuffer(opfb)
	case opTypeUserChange:
		opr = userChange{
			op: op{
				opType:    opTypeUserChange,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeUserDelete:
		opr = userDelete{
			op: op{
				opType:    opTypeUserDelete,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeUserRename:
		opr = userRename{
			op: op{
				opType:    opTypeUserRename,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeNameInit:
		opr = nameInit{
			op: op{
				opType:    opTypeNameInit,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeNameChange:
		opr = nameChange{
			op: op{
				opType:    opTypeNameChange,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeNameDelete:
		opr = nameDelete{
			op: op{
				opType:    opTypeNameDelete,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeVersionSave:
		opr = versionSave{
			op: op{
				opType:    opTypeVersionSave,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeVersionDelete:
		opr = versionDelete{
			op: op{
				opType:    opTypeVersionDelete,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeVersionPublish:
		opr = versionPublish{
			op: op{
				opType:    opTypeVersionPublish,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeVersionUnpublish:
		opr = versionUnpublish{
			op: op{
				opType:    opTypeVersionUnpublish,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeACLInit:
		opr = aclInit{
			op: op{
				opType:    opTypeACLInit,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeACLUpdate:
		opr = aclUpdate{
			op: op{
				opType:    opTypeACLUpdate,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case opTypeACLDelete:
		opr = aclDelete{
			op: op{
				opType:    opTypeACLDelete,
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	}
	return opr, nil
}
