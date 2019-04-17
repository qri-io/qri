// +build !darwin

package actions

import (
	"fmt"
	"runtime"
)

// DaemonHelp is a stub for platforms that cannot yet use daemons
func DaemonHelp() error {
	return fmt.Errorf("cannot display help for daemon on platform: %s", runtime.GOOS)
}

// DaemonInstall is a stub for platforms that cannot yet install daemons
func DaemonInstall() error {
	return fmt.Errorf("cannot install daemon on platform: %s", runtime.GOOS)
}

// DaemonUninstall is a stub for platforms that cannot yet uninstall daemons
func DaemonUninstall() error {
	return fmt.Errorf("cannot uninstall daemon on platform: %s", runtime.GOOS)
}

// DaemonShow is a stub for platforms that cannot yet show details about daemons
func DaemonShow() error {
	return fmt.Errorf("cannot show daemon on platform: %s", runtime.GOOS)
}
