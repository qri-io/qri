package log

// opType is the set of kinds of operations
// OpType splits the provided byte in half, using the higher 4 bits for the
// "category" of operation, and the lower 4 bits for the type of operation
// within the category
type opType byte

const (
	opTypeUserInit   opType = 0x00
	opTypeUserChange opType = 0x01
	opTypeUserDelete opType = 0x02
	opTypeUserRename opType = 0x03

	opTypeNameInit   opType = 0x10
	opTypeNameChange opType = 0x11
	opTypeNameDelete opType = 0x12

	opTypeVersionSave      opType = 0x20
	opTypeVersionDelete    opType = 0x21
	opTypeVersionPublish   opType = 0x22
	opTypeVersionUnpublish opType = 0x23

	opTypeACLInit   opType = 0x30
	opTypeACLUpdate opType = 0x31
	opTypeACLDelete opType = 0x32
)

type operation interface {
	OpType() opType
	Timestamp() uint64
	Ref() string
}

type op struct {
	opType    opType
	timestamp uint64
	ref       string
}

func (o op) OpType() opType    { return o.opType }
func (o op) Timestamp() uint64 { return o.timestamp }
func (o op) Ref() string       { return o.ref }

// userInit signifies the creation of a user
type userInit struct {
	op
	Author   string
	Username string
}

// userChange signifies a change in any user details that aren't
// a username
type userChange struct {
	op
	Note string
}

// userRename signifies a user has changed their username
type userRename struct {
	op
	Username string
}

// userDelete signifies user details have been deleted
type userDelete struct {
	op
	Author string
}

// nameInit signifies dataset name creation in a user's namespace
type nameInit struct {
	op
	Author   string
	Username string
	Name     string
}

// nameChange signifies a dataset name change
type nameChange struct {
	op
	Name string
}

// nameDelete signifies a dataset name has been deleted
type nameDelete struct {
	op
	Name string
}

// versionSave signifies creating one new dataset version
type versionSave struct {
	op
	Prev string
	Size uint64
	Note string
}

// versionDelete signifies deleting one or more versions of a dataset
type versionDelete struct {
	op
	Revisions int
}

// versionPublish signifies publishing one or more sequential versions of a
// dataset
type versionPublish struct {
	op
	Revisions   uint32
	Destination string
}

// versionUnpublish signifies unpublishing one or more sequential versions of a
// dataset
type versionUnpublish struct {
	op
	Revisions   uint32
	Destination string
}

// aclInit signifies intializing an access control list
type aclInit struct {
	op
	Prev string
	Size uint64
	Note string
}

// aclUpdate signifies a change to an access control list
type aclUpdate struct {
	op
	Prev string
	Size uint64
	Note string
}

// aclDelete signifies removing an access control list
type aclDelete struct {
	op
	Prev string
}
