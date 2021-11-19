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

// defaultFilePermMask is 0 because Windows does not use Unix-style file permissions
func defaultFilePermMask() int {
	return 0
}

// sizeOfTerminal returns an invalid size on Windows
func sizeOfTerminal() (int, int) {
	return -1, -1
}
