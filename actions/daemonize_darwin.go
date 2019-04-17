package actions

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

// CopyFile copies a file from the source to the destination path
func CopyFile(src, dst string) error {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		return fmt.Errorf("CopyFile, error reading from %s: %s", src, err.Error())
	}

	err = ioutil.WriteFile(dst, content, os.ModePerm)
	if err != nil {
		return fmt.Errorf("CopyFile, error writing to %s: %s", dst, err.Error())
	}

	return nil
}

const daemonPlistName = "io.qri.daemon.plist"

const daemonPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>

    <key>Label</key>
    <string>io.qri.daemon.plist</string>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StartInterval</key>
    <integer>20</integer>

    <key>StandardErrorPath</key>
    <string>$QRIHOME/logs/stderr.log</string>

    <key>StandardOutPath</key>
    <string>$QRIHOME/logs/stdout.log</string>

    <key>EnvironmentVariables</key>
    <dict>
      <key>PATH</key>
      <string><![CDATA[/usr/local/bin:/usr/local/sbin:/usr/bin:/bin:/usr/sbin:/sbin]]></string>
    </dict>

    <key>WorkingDirectory</key>
    <string>$QRIHOME/root/</string>

    <key>ProgramArguments</key>
    <array>
      <string>$QRIBIN</string>
      <string>connect</string>
    </array>

  </dict>
</plist>
`

// DaemonInstall installs the daemon in a platform specific manner
func DaemonInstall() error {
	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	// Get the path to the qri base directory
	qriPath := os.Getenv("QRI_PATH")
	if qriPath == "" {
		qriPath = filepath.Join(home, ".qri")
	}
	// Get the path to a reliable location where the qri binary can be found
	qriBin := filepath.Join(qriPath, "bin", "qri")

	// Create directories in the qri base directory: bin/, root/, and logs/
	err = os.Mkdir(filepath.Join(qriPath, "bin"), os.ModePerm)
	if err != nil && !strings.Contains(err.Error(), "file exists") {
		return err
	}
	err = os.Mkdir(filepath.Join(qriPath, "root"), os.ModePerm)
	if err != nil && !strings.Contains(err.Error(), "file exists") {
		return err
	}
	err = os.Mkdir(filepath.Join(qriPath, "logs"), os.ModePerm)
	if err != nil && !strings.Contains(err.Error(), "file exists") {
		return err
	}

	// Get path to the currently running binary
	thisBin, err := os.Executable()
	if err != nil {
		return err
	}
	// Copy it to the reliable location in the qri base directory
	err = CopyFile(thisBin, qriBin)
	if err != nil {
		return err
	}

	// Replace variables in the template
	content := daemonPlistTemplate
	content = strings.Replace(content, "$QRIHOME", qriPath, -1)
	content = strings.Replace(content, "$QRIBIN", qriBin, -1)
	// Write it to the LaunchAgents directory
	plistFilename := filepath.Join(home, fmt.Sprintf("Library/LaunchAgents/%s", daemonPlistName))
	err = ioutil.WriteFile(plistFilename, []byte(content), os.ModePerm)
	if err != nil {
		return err
	}

	// Execute the launchctl command to load the plist
	cmdProgram := "launchctl"
	cmdArgs := []string{"load", plistFilename}

	cmd := exec.Command(cmdProgram, cmdArgs...)
	err = cmd.Run()
	if err != nil {
		return err
	}

	fmt.Printf("Launched!\n")
	return nil
}

// DaemonHelp installs the daemon in a platform specific manner
func DaemonHelp() error {
	fmt.Printf(`Usage: qri daemon [action]
action can be "install", "uninstall", "show"

qri daemon install   - Install the daemon so it is always running
qri daemon uninstall - Uninstall the daemon
qri daemon show      - Show details information about the daemon
`)
	return nil
}

// DaemonUninstall uninstalls the daemon in a platform specific manner
func DaemonUninstall() error {
	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	// Create path to installed launch agent
	plistFilename := filepath.Join(home, fmt.Sprintf("Library/LaunchAgents/%s", daemonPlistName))

	// Execute the launchctl command to load the plist
	cmdProgram := "launchctl"
	cmdArgs := []string{"unload", plistFilename}

	cmd := exec.Command(cmdProgram, cmdArgs...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()

	// Handle error message from launchctl
	if strings.Contains(stderr.String(), "Could not find specified service") {
		return fmt.Errorf("qri daemon not installed, cannot uninstall")
	}

	// Handle other errors from command execution
	if err != nil {
		return err
	}

	fmt.Printf("Uninstalled\n")
	return nil
}

// DaemonShow shows details about the daemon in a platform specific manner
func DaemonShow() error {
	// Execute the launchctl command to list details around the daemon
	cmdProgram := "launchctl"
	cmdArgs := []string{"list", daemonPlistName}

	cmd := exec.Command(cmdProgram, cmdArgs...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	// Handle error message from launchctl
	if strings.Contains(stderr.String(), "Could not find service") {
		return fmt.Errorf("qri daemon not running")
	}

	// Handle other errors from command execution
	if err != nil {
		return err
	}

	fmt.Printf(stdout.String())
	return nil
}
