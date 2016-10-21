package cmd

type Commit struct {
	Author     *Person `json:"author"`
	Message    string  `json:""`
	Migrations []*Migration
	Changes    []*Change
}

func (c *Commit) LocalRead() error {
	return nil
}
