package log

import (
	"fmt"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/repo"
)

// Logbook is a journal of operations organized into a collection of
// author-namespaced, append-only logs.
// Logbooks are connected to a single author, and represent their view of
// the global dataset graph.
// Any write operation performed on the logbook are attributed to a single
// author, denoted by a private key. Logbooks can replicate logs from other
// authors, forming a conflict-free replicated data type (CRDT), and a basis
// for collaboration through knowledge of each other's operations
type Logbook struct {
	username string
	pk       crypto.PrivKey
	Logs     map[label]log
}

// NewLogbook initializes a logbook, reading any existing data at the given
// location, on the given filesystem. logbooks are encrypted at rest. The
// same key must be given to decrypt an existing logbook
func NewLogbook(pk crypto.PrivKey, username string, fs qfs.Filesystem, location string) (Logbook, error) {
	return Logbook{}, fmt.Errorf("not finished")
}

// NameInit initializes a new name within the author's namespace. Dataset
// histories start with a NameInit
func (book Logbook) NameInit(name string) error {
	return fmt.Errorf("not finished")
}

// VersionSave adds an operation to a log marking the creation of a dataset
// version. Logbook will copy details from the provided dataset pointer
func (book Logbook) VersionSave(alias string, ds *dataset.Dataset) error {
	return fmt.Errorf("not finished")
}

// VersionAmend adds an operation to a log amending a dataset version
func (book Logbook) VersionAmend(alias string, ds *dataset.Dataset) error {
	return fmt.Errorf("not finished")
}

// VersionDelete adds an operation to a log marking a number of sequential
// versions from HEAD as deleted. Because logs are append-only, deletes are
// recorded as "tombstone" operations that mark removal.
func (book Logbook) VersionDelete(alias string, revisions int) error {
	return fmt.Errorf("not finished")
}

// Publish adds an operation to a log marking the publication of a number of
// versions to one or more destinations. Versions count continously from head
// back
func (book Logbook) Publish(alias string, revisions int, destinations ...string) error {
	return fmt.Errorf("not finished")
}

// Unpublish adds an operation to a log marking an unpublish request for a count
// of sequential versions from HEAD
func (book Logbook) Unpublish(alias string, revisions int, destinations ...string) error {
	return fmt.Errorf("not finished")
}

// State plays a set of operations for a given log, producing a State struct
// that describes the current state of a dataset
func (book Logbook) State(alias string) State {
	return State{}
}

// State is the current state of a log, the result of executing all
// operations in the log
type State struct {
	Alias    string
	Versions []repo.DatasetRef
}

// opset is a causally-ordered set of operations performed by a single author
// attributing Opsets to authors
type opset struct {
	Author string
	Ops    []operation
}

// Len returns the number of of the latest entry in the log
func (set opset) Len() int {
	return len(set.Ops)
}

func (set opset) Type() string {
	return ""
}

func (set opset) Label() label {
	return label{}
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

// label combines two string components, one for the content-addressed
// identifier, and another for the human-readable name
type label [2]string

// Author returns the author portion of a label
func (lbl label) Author() string {
	return lbl[0]
}

// Name returns the name portion of a string
func (lbl label) Name() string {
	return lbl[1]
}

// log is a logical collection of unique opsets
type log map[label]opset

// // Put an operation into the log
// func (log log) Put(op operation) log {
// 	clk := log.Clock()

// 	if op.Tick == clk {
// 		return Log{
// 			Ops: append(log.Ops, op),
// 		}
// 	} else if op.Tick > clk {
// 		return Log{
// 			Ops: append(log.Ops, op),
// 		}
// 	}

// 	// TODO (b5) - iterate from the end of the slice instead
// 	for i, o := range log.Ops {
// 		if op.Tick == o.Tick {
// 			updated := log.Ops[i:]
// 			for j, _ := range updated {
// 				updated[j].Tick++
// 			}
// 			return Log{
// 				Ops: append(log.Ops[:i], append([]Op{op}, updated...)...),
// 			}
// 		} else if op.Tick < o.Tick {
// 			return Log{
// 				Ops:    append(log.Ops[:i], append([]Op{op}, log.Ops[i:]...)...),
// 				Secret: log.Secret,
// 			}
// 		}
// 	}

// 	panic("this isn't supposed to happen")
// }

// func (log Log) Copy() Log {
// 	cpy := make([]Op, len(log.Ops))
// 	copy(cpy, log.Ops)
// 	return Log{
// 		Ops:    cpy,
// 	}
// }

// State calculates the current log state by applying all operations
// func (log Log) State() LogState {
// 	s := LogState{}
// 	for _, op := range log.Ops {
// 		switch op.Type {
// 		case OpTypeMerge:
// 			// s.Merges++
// 		case OpTypeDatasetCommit:
// 			s.Commits = append(s.Commits, op)
// 		case OpTypeSuggestionUpdate:
// 			for i, sug := range s.Suggestions {
// 				if sug.Subject == op.Prev {
// 					s.Suggestions[i] = op
// 					continue
// 				}
// 			}
// 			s.Suggestions = append(s.Suggestions, op)
// 		case OpTypeSuggestionDelete:
// 			for i, sug := range s.Suggestions {
// 				if sug.Subject == op.Subject {
// 					if i == len(s.Suggestions)-1 {
// 						s.Suggestions = s.Suggestions[:i]
// 					} else {
// 						s.Suggestions = append(s.Suggestions[:i], s.Suggestions[i+1:]...)
// 					}
// 					break
// 				}
// 			}
// 		}
// 	}

// 	return s
// }
