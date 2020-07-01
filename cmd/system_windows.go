package cmd

// ensureLargeNumOpenFiles doesn't need to do anything on Windows
func ensureLargeNumOpenFiles() {
	// Nothing to do.
}

// stdoutIsTerminal returns true on Windows, because we just assume that no Windows
// user is trying to use pipes
func stdoutIsTerminal() bool {
	return true
}