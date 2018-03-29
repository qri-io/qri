package config

// Repo configures a qri repo
type Repo struct {
	Middleware []string
	Type       string
}

// Default creates & returns a new defaul repo configuration
func (Repo) Default() *Repo {
	return &Repo{
		Type:       "fs",
		Middleware: []string{},
	}
}
