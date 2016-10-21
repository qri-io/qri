package cmd

type Status struct {
	Updated int64
}

// Status returns a string representing the current status of the repo
func (r *Status) String() string {
	return "this is the repo status"
}
