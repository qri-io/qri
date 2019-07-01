package log

const (
	// LogPartitionSize is the number
	LogPartitionSize = 10000
	// TODO (b5) - consider adding a distance-from latest that's considered a
	// "dissociative state" where a point of divergence is too far back in history
	// to be reconciled without user intervention
	// DissociativeDistance = 250
)

type Log struct {
	Ops    []Op
	Secret []byte
}

// Put an operation into the log
func (log Log) Put(op Op) Log {
	clk := log.Clock()

	if op.Tick == clk {
		merge := Op{
			Type: OpTypeMerge,
			Tick: clk + 1,
		}
		return Log{
			Ops:    append(log.Ops, op, merge),
			Secret: log.Secret,
		}
	} else if op.Tick > clk {
		return Log{
			Ops:    append(log.Ops, op),
			Secret: log.Secret,
		}
	}

	// TODO (b5) - iterate from the end of the slice instead
	for i, o := range log.Ops {
		if op.Tick == o.Tick {
			merge := Op{
				Type: OpTypeMerge,
				Tick: op.Tick + 1,
			}
			updated := log.Ops[i:]
			for j, _ := range updated {
				updated[j].Tick++
			}
			return Log{
				Ops:    append(log.Ops[:i], append([]Op{op, merge}, updated...)...),
				Secret: log.Secret,
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

// Clock returns the index value of the latest entry in the log
func (log Log) Clock() uint64 {
	if len(log.Ops) == 0 {
		return 0
	}
	return log.Ops[len(log.Ops)-1].Tick
}

func (log Log) Copy() Log {
	cpy := make([]Op, len(log.Ops))
	copy(cpy, log.Ops)
	return Log{
		Ops:    cpy,
		Secret: log.Secret,
	}
}

// State calculates the current log state by applying all operations
func (log Log) State() LogState {
	s := LogState{}
	for _, op := range log.Ops {
		switch op.Type {
		case OpTypeMerge:
			s.Merges++
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

// LogState is the current state of a log, the result of executing all
// operations in the log
type LogState struct {
	Commits     []Op
	Suggestions []Op
	Merges      int
}

// Sync
func (log Log) Sync(b Log) (Log, error) {
	if len(log.Ops) == len(b.Ops) && log.Clock() == b.Clock() {
		return log, nil
	} else if len(log.Ops) > len(b.Ops) {
		return log, nil
	}

	return log, nil
}
