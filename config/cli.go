package config

// CLI defines configuration details for the qri command line client (CLI)
type CLI struct {
	ColorizeOutput bool
}

// Default returns a new default CLI configuration
func (CLI) Default() *CLI {
	return &CLI{
		ColorizeOutput: true,
	}
}
