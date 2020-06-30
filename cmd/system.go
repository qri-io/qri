// +build !windows

package cmd

import (
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
	"golang.org/x/crypto/ssh/terminal"
)

// preferredNumOpenFiles is the perferred number of open files that the process can have.
// This value tends to be the recommended value for `ulimit -n`, as seen on github discussions
// around various projects such as hugo, mongo, redis.
const preferredNumOpenFiles = 10000

// ensureLargeNumOpenFiles ensures that user can have a large number of open files
func ensureLargeNumOpenFiles() {
	// Get the number of open files currently allowed.
	var rLimit unix.Rlimit
	err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		panic(err)
	}
	if rLimit.Cur >= preferredNumOpenFiles {
		return
	}

	// Set the number of open files that are allowed to be sufficiently large. This avoids
	// the error "too many open files" that often occurs when running IPFS or other
	// local database-like technologies.
	rLimit.Cur = preferredNumOpenFiles
	rLimit.Max = preferredNumOpenFiles

	err = unix.Setrlimit(unix.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			// If permission was denied, just ignore the error silently.
			return
		}
		fmt.Printf("error setting max open files limit: %s\n", err)
		return
	}
}

// stdoutIsTerminal returns whether stdout is writing to a terminal, as opposed to something
// like a pipe. For example, when running `qri get me/my_ds --format zip` this returns true,
// but when running `qri get me/my_ds --format zip | gzip -l` this returns false
func stdoutIsTerminal() bool {
	return terminal.IsTerminal(syscall.Stdout)
}