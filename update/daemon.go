// +build !darwin

package update

import (
	"fmt"
	"runtime"
)

// daemonHelp is a stub for platforms that cannot yet use daemons
func daemonHelp() error {
	return fmt.Errorf("cannot display help for daemon on platform: %s", runtime.GOOS)
}

// daemonInstall is a stub for platforms that cannot yet install daemons
func daemonInstall() error {
	return fmt.Errorf("cannot install daemon on platform: %s", runtime.GOOS)
}

// daemonUninstall is a stub for platforms that cannot yet uninstall daemons
func daemonUninstall() error {
	return fmt.Errorf("cannot uninstall daemon on platform: %s", runtime.GOOS)
}

// daemonShow is a stub for platforms that cannot yet show details about daemons
func daemonShow() (string, error) {
	return "", fmt.Errorf("cannot show daemon on platform: %s", runtime.GOOS)
}
