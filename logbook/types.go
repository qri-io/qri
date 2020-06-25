package logbook

import (
	"github.com/qri-io/qri/logbook/oplog"
)

// UserLog is the top-level log representing users that make datasets
type UserLog struct {
	l *oplog.Log
}

func newUserLog(log *oplog.Log) *UserLog {
	return &UserLog{l: log}
}

// TODO(dustmop): Consider changing the "Append" methods to type-safe methods that are specific
// to each log level, which accept individual parameters instead of type-unsafe Op values.

// Append adds an op to the UserLog
func (alog *UserLog) Append(op oplog.Op) {
	if op.Model != AuthorModel {
		log.Errorf("cannot Append, incorrect model %d for UserLog", op.Model)
		return
	}

	alog.l.Append(op)
}

// ProfileID returns the profileID for the user
func (alog *UserLog) ProfileID() string {
	return alog.l.Ops[0].AuthorID
}

// AddChild adds a child log
// TODO(dustmop): Change this parameter to be a DatasetLog
func (alog *UserLog) AddChild(l *oplog.Log) {
	alog.l.AddChild(l)
}

// DatasetLog is the mid-level log representing a single dataset
type DatasetLog struct {
	l *oplog.Log
}

func newDatasetLog(log *oplog.Log) *DatasetLog {
	return &DatasetLog{l: log}
}

// Append adds an op to the DatasetLog
func (dlog *DatasetLog) Append(op oplog.Op) {
	if op.Model != DatasetModel {
		log.Errorf("cannot Append, incorrect model %d for DatasetLog", op.Model)
		return
	}
	dlog.l.Append(op)
}

// InitID returns the initID for the dataset
func (dlog *DatasetLog) InitID() string {
	return dlog.l.ID()
}

// BranchLog is the bottom-level log representing a branch of a dataset history
type BranchLog struct {
	l *oplog.Log
}

func newBranchLog(l *oplog.Log) *BranchLog {
	blog := &BranchLog{l: l}
	// BranchLog should never have logs underneath it, display error if any are found
	if len(blog.l.Logs) > 0 {
		log.Errorf("invalid branchLog, has %d child Logs", len(blog.l.Logs))
	}
	return blog
}

// Append adds an op to the BranchLog
func (blog *BranchLog) Append(op oplog.Op) {
	if op.Model != BranchModel && op.Model != CommitModel && op.Model != PublicationModel {
		log.Errorf("cannot Append, incorrect model %d for BranchLog", op.Model)
		return
	}
	blog.l.Append(op)
}

// Size returns the size of the branch
func (blog *BranchLog) Size() int {
	return len(blog.l.Ops)
}

// Ops returns the raw Op list
func (blog *BranchLog) Ops() []oplog.Op {
	return blog.l.Ops
}
