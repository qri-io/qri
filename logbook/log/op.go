package log

import (
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/qri-io/qri/logbook/logfb"
)

// OpType is the set of all kinds of operations, they are two bytes in length
// OpType splits the provided byte in half, using the higher 4 bits for the
// "category" of operation, and the lower 4 bits for the type of operation
// within the category
// the second byte is reserved for future use
type OpType uint16

const (
	OpTypeUserInit   OpType = 0x0000
	OpTypeUserChange OpType = 0x0100
	OpTypeUserDelete OpType = 0x0200
	OpTypeUserRename OpType = 0x0300

	OpTypeNameInit   OpType = 0x1000
	OpTypeNameChange OpType = 0x1100
	OpTypeNameDelete OpType = 0x1200

	OpTypeVersionSave      OpType = 0x2000
	OpTypeVersionDelete    OpType = 0x2100
	OpTypeVersionPublish   OpType = 0x2200
	OpTypeVersionUnpublish OpType = 0x2300

	OpTypeACLInit   OpType = 0x3000
	OpTypeACLUpdate OpType = 0x3100
	OpTypeACLDelete OpType = 0x3200
)

// Operation is the atomic unit of a log. Logs append operations to form a
// record of events
type Operation interface {
	// OpType designates the kind of operation
	// OpType designates the kind of operation
	OpType() OpType
	// Operations are timestamped with nanosecond-precision unix timestamps
	// for informative purposes only timestamps are not to be used in conflict
	// resolution
	Timestamp() int64
	// An Operation refers to an address for the payload of it's data. Reference
	// is
	Ref() string
	// All Operations can be written to a flatbuffer
	MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT
}

// InitOperation is an operation capable of initializing a new log. Init
// operations contain author attribution and human-readable naming information
// used to attribute the authorship of a log
type InitOperation interface {
	Operation
	Name() string
	AuthorName() string
	AuthorID() string
}

type op struct {
	timestamp int64
	ref       string
}

func (o op) Timestamp() int64 { return o.timestamp }
func (o op) Ref() string      { return o.ref }

// UserInit signifies the creation of a user
type UserInit struct {
	op
	Username string
}

// compile-time assertino that UserInit is an init operation
var _ InitOperation = (*UserInit)(nil)

func newUserInitFlatbuffer(o *logfb.Operation) UserInit {
	return UserInit{
		op: op{
			timestamp: o.Timestamp(),
			ref:       string(o.Ref()),
		},
		Username: string(o.Name()),
	}
}

// OpType designates the kind of operation
func (o UserInit) OpType() OpType { return OpTypeUserInit }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o UserInit) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	ref := builder.CreateString(o.ref)
	username := builder.CreateString(o.Username)

	logfb.OperationStart(builder)
	logfb.OperationAddType(builder, uint16(o.OpType()))
	logfb.OperationAddTimestamp(builder, o.timestamp)
	logfb.OperationAddRef(builder, ref)
	logfb.OperationAddName(builder, username)
	return logfb.OperationEnd(builder)
}

// Name returns the name of this branch, which for UserInit operations is the
// same as the author's username
func (o UserInit) Name() string {
	return o.Username
}

// AuthorName returns the human readable name of the author
func (o UserInit) AuthorName() string {
	return o.Username
}

// AuthorID returns the canonical identifier for the author. We rely on this
// being base58 hash of public key
func (o UserInit) AuthorID() string {
	return o.ref
}

// UserChange signifies a change in any user details that aren't
// a username
type UserChange struct {
	op
	Note string
}

// OpType designates the kind of operation
func (o UserChange) OpType() OpType { return OpTypeUserChange }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o UserChange) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// UserRename signifies a user has changed their username
type UserRename struct {
	op
	Username string
}

// OpType designates the kind of operation
func (o UserRename) OpType() OpType { return OpTypeUserRename }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o UserRename) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// UserDelete signifies user details have been deleted
type UserDelete struct {
	op
	Author string
}

// OpType designates the kind of operation
func (o UserDelete) OpType() OpType { return OpTypeUserDelete }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o UserDelete) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// NameInit signifies dataset name creation in a user's namespace
type NameInit struct {
	op
	author   string
	username string
	name     string
}

// NewNameInit creates a name init operation
func NewNameInit(author, username, name string) NameInit {
	now := time.Now().UnixNano()
	return NameInit{
		op: op{
			timestamp: now,
		},
		author:   author,
		username: username,
		name:     name,
	}
}

// OpType designates the kind of operation
func (o NameInit) OpType() OpType { return OpTypeNameInit }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o NameInit) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	author := builder.CreateString(o.author)
	username := builder.CreateString(o.username)
	name := builder.CreateString(o.name)

	logfb.OperationStart(builder)
	logfb.OperationAddType(builder, uint16(o.OpType()))
	logfb.OperationAddTimestamp(builder, o.timestamp)
	logfb.OperationAddRef(builder, author)
	logfb.OperationAddDestination(builder, username)
	logfb.OperationAddName(builder, name)
	return logfb.OperationEnd(builder)
}

// Name returns the name of this branch, which for UserInit operations is the
// same as the author's username
func (o NameInit) Name() string {
	return o.name
}

// AuthorName returns the human readable name of the author
func (o NameInit) AuthorName() string {
	return o.username
}

// AuthorID returns the canonical identifier for the author. We rely on this
// being base58 hash of public key
func (o NameInit) AuthorID() string {
	return o.ref
}

// NameChange signifies a dataset name change
type NameChange struct {
	op
	Name string
}

// OpType designates the kind of operation
func (o NameChange) OpType() OpType { return OpTypeNameChange }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o NameChange) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// NameDelete signifies a dataset name has been deleted
type NameDelete struct {
	op
	Name string
}

// OpType designates the kind of operation
func (o NameDelete) OpType() OpType { return OpTypeNameDelete }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o NameDelete) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// VersionSave signifies creating one new dataset version
type VersionSave struct {
	op
	Prev string
	Size uint64
	Note string
}

// OpType designates the kind of operation
func (o VersionSave) OpType() OpType { return OpTypeVersionSave }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o VersionSave) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	prev := builder.CreateString(o.Prev)
	ref := builder.CreateString(o.ref)
	note := builder.CreateString(o.Note)

	logfb.OperationStart(builder)
	logfb.OperationAddType(builder, uint16(o.OpType()))
	// TODO (b5):
	logfb.OperationAddTimestamp(builder, 0)
	logfb.OperationAddRef(builder, ref)
	logfb.OperationAddPrev(builder, prev)
	logfb.OperationAddNote(builder, note)
	return logfb.OperationEnd(builder)
}

// VersionDelete signifies deleting one or more versions of a dataset
type VersionDelete struct {
	op
	Revisions int
}

// OpType designates the kind of operation
func (o VersionDelete) OpType() OpType { return OpTypeVersionDelete }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o VersionDelete) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// VersionPublish signifies publishing one or more sequential versions of a
// dataset
type VersionPublish struct {
	op
	Revisions   int
	Destination string
}

// OpType designates the kind of operation
func (o VersionPublish) OpType() OpType { return OpTypeVersionPublish }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o VersionPublish) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// VersionUnpublish signifies unpublishing one or more sequential versions of a
// dataset
type VersionUnpublish struct {
	op
	Revisions   int
	Destination string
}

// OpType designates the kind of operation
func (o VersionUnpublish) OpType() OpType { return OpTypeVersionUnpublish }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o VersionUnpublish) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// ACLInit signifies intializing an access control list
type ACLInit struct {
	op
	Prev string
	Size uint64
	Note string
}

// OpType designates the kind of operation
func (o ACLInit) OpType() OpType { return OpTypeACLInit }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o ACLInit) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// ACLUpdate signifies a change to an access control list
type ACLUpdate struct {
	op
	Prev string
	Size uint64
	Note string
}

// OpType designates the kind of operation
func (o ACLUpdate) OpType() OpType { return OpTypeACLUpdate }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o ACLUpdate) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

// ACLDelete signifies removing an access control list
type ACLDelete struct {
	op
	Prev string
}

// OpType designates the kind of operation
func (o ACLDelete) OpType() OpType { return OpTypeACLDelete }

// MarshalFlatbuffer writes this operation to a flatbuffer, returning the
// ending byte offset
func (o ACLDelete) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// TODO (b5) - finish
	return 0
}

func unmarshalOperationFlatbuffer(opfb *logfb.Operation) (opr Operation, err error) {
	switch OpType(opfb.Type()) {
	case OpTypeUserInit:
		opr = newUserInitFlatbuffer(opfb)
	case OpTypeUserChange:
		opr = UserChange{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeUserDelete:
		opr = UserDelete{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeUserRename:
		opr = UserRename{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeNameInit:
		opr = NameInit{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeNameChange:
		opr = NameChange{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeNameDelete:
		opr = NameDelete{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeVersionSave:
		opr = VersionSave{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeVersionDelete:
		opr = VersionDelete{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeVersionPublish:
		opr = VersionPublish{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeVersionUnpublish:
		opr = VersionUnpublish{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeACLInit:
		opr = ACLInit{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeACLUpdate:
		opr = ACLUpdate{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	case OpTypeACLDelete:
		opr = ACLDelete{
			op: op{
				timestamp: opfb.Timestamp(),
				ref:       string(opfb.Ref()),
			},
		}
	}
	return opr, nil
}
