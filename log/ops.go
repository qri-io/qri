package log

// OpType defines different types of operations
type OpType uint8

const (
	OpTypeUnknown OpType = iota
	OpTypeMerge
	OpTypeAclUpdate
	OpTypePeerUpdate
	OpTypePeerDelete
	OpTypeDatasetCommit
	OpTypeDatasetDelete
	OpTypeSuggestionUpdate
	OpTypeSuggestionDelete
)

// Op is a single operation in the log
type Op struct {
	Tick    uint64
	Type    OpType
	Author  string
	Subject string
	Prev    string
	Note    string
}

func PutDatasetCommit(log Log, author, subject, prev, note string) (Log, error) {
	op := Op{
		Type:    OpTypeDatasetCommit,
		Tick:    log.Clock() + 1,
		Author:  author,
		Subject: subject,
		Prev:    prev,
		Note:    note,
	}
	return log.Put(op), nil
}

func PutSuggestionUpdate(log Log, author, subject, prev, note string) (Log, error) {
	op := Op{
		Type:    OpTypeSuggestionUpdate,
		Tick:    log.Clock() + 1,
		Author:  author,
		Subject: subject,
		Prev:    prev,
		Note:    note,
	}
	return log.Put(op), nil
}

func PutSuggestionDelete(log Log, author, subject string) (Log, error) {
	op := Op{
		Type:    OpTypeSuggestionDelete,
		Tick:    log.Clock() + 1,
		Author:  author,
		Subject: subject,
	}
	return log.Put(op), nil
}
