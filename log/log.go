package log

import (
	"context"
)

// Opset is a causally-ordered set of operations performed by a single author
// attributing Opsets to authors
type Opset struct {
	Author string
	Ops    []Op
}

// Clock returns the index value of the latest entry in the log
func (ops Opset) Clock() uint64 {
	if len(ops) == 0 {
		return 0
	}
	return ops[len(ops)-1].Tick
}

func (opst opSet) Type() string {
	return ""
}

func (ops Opset) Label() Label {
	return Label{}
}

// Union combines two opsets that share either common or disjoint operations
// func Union(a, b Opset) (Opset, error) {
// 	if len(a) == len(b) && a.Clock() == b.Clock() {
// 		return log, nil
// 	} else if len(a) > len(b) {
// 		return log, nil
// 	}

// 	return log, nil
// }

func PutDatasetCommit(log Log, author, subject, prev, note string) (Log, error) {
	op := Op{
		Type:    OpTypeDatasetCommit,
		Tick:    log.Clock() + 1,
		Subject: subject,
		Prev:    prev,
		Note:    note,
	}
	return log.Put(op), nil
}

// Label combines two string components for
type Label [2]string

// Author returns the author portion of a label
func (lbl Label) Author() string {
	return lbl[0]
}

// Name returns the name portion of a string
func (lbl Label) Name() string {
	return lbl[1]
}

// Log is a logical collection of unique opsets
type Log map[Label]Opset

// Put adds a
func (log Log) Put(ctx context.Context, name string, log Log) error {
	return nil
}

// Put an operation into the log
func (log Log) Put(op Op) Log {
	clk := log.Clock()

	if op.Tick == clk {
		return Log{
			Ops: append(log.Ops, op),
		}
	} else if op.Tick > clk {
		return Log{
			Ops: append(log.Ops, op),
		}
	}

	// TODO (b5) - iterate from the end of the slice instead
	for i, o := range log.Ops {
		if op.Tick == o.Tick {
			updated := log.Ops[i:]
			for j, _ := range updated {
				updated[j].Tick++
			}
			return Log{
				Ops: append(log.Ops[:i], append([]Op{op}, updated...)...),
			}
		} else if op.Tick < o.Tick {
			return Log{
				Ops:    append(log.Ops[:i], append([]Op{op}, log.Ops[i:]...)...),
				Secret: log.Secret,
			}
		}
	}

	panic("this isn't supposed to happen")
}

// func (log Log) Copy() Log {
// 	cpy := make([]Op, len(log.Ops))
// 	copy(cpy, log.Ops)
// 	return Log{
// 		Ops:    cpy,
// 	}
// }

// State calculates the current log state by applying all operations
func (log Log) State() LogState {
	s := LogState{}
	for _, op := range log.Ops {
		switch op.Type {
		case OpTypeMerge:
			// s.Merges++
		case OpTypeDatasetCommit:
			s.Commits = append(s.Commits, op)
		case OpTypeSuggestionUpdate:
			for i, sug := range s.Suggestions {
				if sug.Subject == op.Prev {
					s.Suggestions[i] = op
					continue
				}
			}
			s.Suggestions = append(s.Suggestions, op)
		case OpTypeSuggestionDelete:
			for i, sug := range s.Suggestions {
				if sug.Subject == op.Subject {
					if i == len(s.Suggestions)-1 {
						s.Suggestions = s.Suggestions[:i]
					} else {
						s.Suggestions = append(s.Suggestions[:i], s.Suggestions[i+1:]...)
					}
					break
				}
			}
		}
	}

	return s
}

type Logset struct {
	Author string
	Logs   map[Label]Log
}

// LogStore
type LogStore interface {
	Put(ctx context.Context, name string, log Log) error
}

// State is the current state of a log, the result of executing all
// operations in the log
type State struct {
	Commits []Op
	Merges  int
}
