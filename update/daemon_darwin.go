package update

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

// copyFile copies a file from the source to the destination path
func copyFile(src, dst string) error {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading from %s: %s", src, err.Error())
	}

	err = ioutil.WriteFile(dst, content, os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing to %s: %s", dst, err.Error())
	}

	return nil
}

const updateDaemonPlistName = "io.qri.update.plist"

const updateDaemonPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Label</key>
    <string>io.qri.update.plist</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StartInterval</key>
    <integer>20</integer>
    <key>StandardErrorPath</key>
    <string>$UPDATEHOME/service.log</string>
    <key>StandardOutPath</key>
    <string>$UPDATEHOME/service.log</string>
    <key>EnvironmentVariables</key>
    <dict>
      <key>PATH</key>
      <string><![CDATA[/usr/local/bin:/usr/local/sbin:/usr/bin:/bin:/usr/sbin:/sbin:$QRIHOME/bin]]></string>
    </dict>
    <key>WorkingDirectory</key>
    <string>$UPDATEHOME/root/</string>
    <key>ProgramArguments</key>
    <array>
      <string>$QRIBIN</string>
			<string>update</string>
			<string>service</string>
			<string>start</string>
    </array>
  </dict>
</plist>
`

// daemonInstall installs the daemon in a platform specific manner
func daemonInstall(repoPath string) error {
	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	updatePath, err := Path(repoPath)
	if err != nil {
		return err
	}

	// Get the path to a reliable location where the qri binary can be found
	qriBin := filepath.Join(repoPath, "bin", "qri")

	// Create bin in the qri repo: bin/
	err = os.Mkdir(filepath.Join(repoPath, "bin"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// ensure .qri/update/root/ as the update service root execution directory
	// note this is for the service process, not the scripts a service runs
	err = os.Mkdir(filepath.Join(updatePath, "root"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// create .qri/update/logs/
	err = os.Mkdir(filepath.Join(updatePath, "logs"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Get path to the currently running binary
	thisBin, err := os.Executable()
	if err != nil {
		return err
	}
	// Copy it to the reliable location in the qri base directory
	err = copyFile(thisBin, qriBin)
	if err != nil {
		return err
	}

	// Replace variables in the template
	content := updateDaemonPlistTemplate
	content = strings.Replace(content, "$QRIHOME", repoPath, -1)
	content = strings.Replace(content, "$UPDATEHOME", updatePath, -1)
	content = strings.Replace(content, "$QRIBIN", qriBin, -1)
	// Write it to the LaunchAgents directory
	plistFilename := filepath.Join(home, fmt.Sprintf("Library/LaunchAgents/%s", updateDaemonPlistName))
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

// daemonUninstall uninstalls the daemon in a platform specific manner
func daemonUninstall(repoPath string) error {
	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	// Create path to installed launch agent
	plistFilename := filepath.Join(home, fmt.Sprintf("Library/LaunchAgents/%s", updateDaemonPlistName))

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
		return fmt.Errorf("update daemon not installed, cannot uninstall")
	}

	// Handle other errors from command execution
	if err != nil {
		return err
	}

	fmt.Printf("Uninstalled\n")
	return nil
}

// daemonShow shows details about the daemon in a platform specific manner
func daemonShow() (string, error) {
	// Execute the launchctl command to list details around the daemon
	cmdProgram := "launchctl"
	cmdArgs := []string{"list", updateDaemonPlistName}

	cmd := exec.Command(cmdProgram, cmdArgs...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	// Handle error message from launchctl
	if strings.Contains(stderr.String(), "Could not find service") {
		return "", fmt.Errorf("update daemon not running")
	}

	// Handle other errors from command execution
	if err != nil {
		return "", err
	}

	return stdout.String(), nil
}
