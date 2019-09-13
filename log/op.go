package log

// OpType is the set of kinds of operations
type OpType byte

const (
	OpTypeUserInit   OpType = 0x00
	OpTypeUserChange OpType = 0x01
	OpTypeUserDelete OpType = 0x02
	OpTypeUserRename OpType = 0x03

	OpTypeNameInit   OpType = 0x10
	OpTypeNameChange OpType = 0x11
	OpTypeNameDelete OpType = 0x12

	OpTypeVersionSave      OpType = 0x20
	OpTypeVersionDelete    OpType = 0x21
	OpTypeVersionPublish   OpType = 0x22
	OpTypeVersionUnpublish OpType = 0x23

	OpTypeACLInit   OpType = 0x30
	OpTypeACLUpdate OpType = 0x31
	OpTypeACLDelete OpType = 0x32
)

// Op is the common interface all Operations in a log must implement
type Op interface {
	// Tick is the logical clock tick for an event
	Tick() uint64
	// OpType is the type of operation
	OpType() OpType
	// Timestamp is a unix timestamp with nanosecond precision. Timestamps are
	// informative only, and never used for determining causal order
	Timestamp() uint64
	// Operations often refer to external content-addressed data structures
	Ref() string
}

type op struct {
	tick      uint64
	opType    OpType
	timestamp uint64
	ref       string
}

func (o op) Tick() uint64      { return o.tick }
func (o op) OpType() OpType    { return o.opType }
func (o op) Timestamp() uint64 { return o.timestamp }

// UserInit signifies the creation of a user
type UserInit struct {
	op
	Author   string
	Username string
}

// UserChange signifies a change in any user details that aren't
// a username
type UserChange struct {
	op
	Note string
}

// UserRename signifies a user has changed their username
type UserRename struct {
	op
	Username string
}

// UserDelete signifies user details have been deleted
type UserDelete struct {
	op
	Author string
}

// NameInit signifies dataset name creation in a user's namespace
type NameInit struct {
	op
	Author   string
	Username string
	Name     string
}

// NameChange signifies a dataset name change
type NameChange struct {
	op
	Name string
}

// NameDelete signifies a dataset name has been deleted
type NameDelete struct {
	op
	Name string
}

// VersionSave signifies creating one new dataset version
type VersionSave struct {
	op
	Prev string
	Size uint64
	Note string
}

// VersionDelete signifies deleting one or more versions of a dataset
type VersionDelete struct {
	op
	Revisions int
}

// VersionPublish signifies publishing one or more sequential versions of a
// dataset
type VersionPublish struct {
	op
	Revisions   uint32
	Destination string
}

// VersionUnpublish signifies unpublishing one or more sequential versions of a
// dataset
type VersionUnpublish struct {
	op
	Revisions   uint32
	Destination string
}

// ACLInit signifies intializing an access control list
type ACLInit struct {
	op
	Prev string
	Size uint64
	Note string
}

// ACLUpdate signifies a change to an access control list
type ACLUpdate struct {
	op
	Prev string
	Size uint64
	Note string
}

// ACLDelete signifies removing an access control list
type ACLDelete struct {
	op
	Prev string
}
