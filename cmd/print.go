package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/qri-io/deepdiff"
	"github.com/qri-io/qri/lib"
)

var noPrompt = false

func setNoColor(noColor bool) {
	color.NoColor = noColor
}

func setNoPrompt(np bool) {
	noPrompt = np
}

func printSuccess(w io.Writer, msg string, params ...interface{}) {
	fmt.Fprintln(w, color.New(color.FgGreen).Sprintf(msg, params...))
}

func printInfo(w io.Writer, msg string, params ...interface{}) {
	fmt.Fprintln(w, fmt.Sprintf(msg, params...))
}

func printWarning(w io.Writer, msg string, params ...interface{}) {
	fmt.Fprintln(w, color.New(color.FgYellow).Sprintf(msg, params...))
}

func printErr(w io.Writer, err error, params ...interface{}) {
	if e, ok := err.(lib.Error); ok && e.Message() != "" {
		fmt.Fprintln(w, color.New(color.FgRed).Sprintf(e.Message(), params...))
		return
	}
	fmt.Fprintln(w, color.New(color.FgRed).Sprintf(err.Error(), params...))
}

// print a slice of stringer items to io.Writer as an indented & numbered list
// offset specifies the number of items that have been skipped, index is 1-based
func printItems(w io.Writer, items []fmt.Stringer, offset int) (err error) {
	buf := &bytes.Buffer{}
	prefix := []byte("    ")
	for i, item := range items {
		buf.WriteString(fmtItem(i+1+offset, item.String(), prefix))
	}
	return printToPager(w, buf)
}

func printToPager(w io.Writer, buf *bytes.Buffer) (err error) {
	// TODO (ramfox): This is POSIX specific, need to expand!
	envPager := os.Getenv("PAGER")
	if ok := doesCommandExist(envPager); !ok {
		// if PAGER does not exist, check to see if 'more' is available on this machine
		envPager = "more"
		if ok := doesCommandExist(envPager); !ok {
			// if 'more' does not exist, check to see if 'less' is available on this machine
			envPager = "less"
			if ok := doesCommandExist(envPager); !ok {
				// sensible default: if none of these commands exist
				// just print the results to the given io.Writer
				fmt.Fprintln(w, buf.String())
				return nil
			}
		}
	}
	pager := &exec.Cmd{}
	os := runtime.GOOS
	if os == "linux" {
		pager = exec.Command("/bin/sh", "-c", envPager, "-R")
	} else {
		pager = exec.Command("/bin/sh", "-c", envPager+" -R")
	}

	pager.Stdin = buf
	pager.Stdout = w
	err = pager.Run()
	if err != nil {
		// sensible default: if something goes wrong printing to the
		// pager, just print the results to the given io.Writer
		fmt.Fprintln(w, buf.String())
	}
	return
}

func fmtItem(i int, item string, prefix []byte) string {
	var res []byte
	bol := true
	b := []byte(item)
	d := []byte(fmt.Sprintf("%d", i))
	prefix1 := append(d, prefix[len(d):]...)
	for i, c := range b {
		if bol && c != '\n' {
			if i == 0 {
				res = append(res, prefix1...)
			} else {
				res = append(res, prefix...)
			}
		}
		res = append(res, c)
		bol = c == '\n'
	}
	return string(res)
}

func prompt(w io.Writer, r io.Reader, msg string) string {
	var input string
	printInfo(w, msg)
	fmt.Fscanln(r, &input)
	return strings.TrimSpace(input)
}

func inputText(w io.Writer, r io.Reader, message, defaultText string) string {
	input := prompt(w, r, fmt.Sprintf("%s [%s]: ", message, defaultText))
	if input == "" {
		input = defaultText
	}

	return input
}

func confirm(w io.Writer, r io.Reader, message string, def bool) bool {
	if noPrompt {
		return def
	}

	yellow := color.New(color.FgYellow).SprintFunc()
	defaultText := "y/N"
	if def {
		defaultText = "Y/n"
	}
	input := prompt(w, r, fmt.Sprintf("%s [%s]: ", yellow(message), defaultText))
	if input == "" {
		return def
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return (input == "y" || input == "yes") == def
}

func usingRPCError(cmdName string) error {
	return fmt.Errorf(`sorry, we can't run the '%s' command while 'qri connect' is running
we know this is super irritating, and it'll be fixed in the future. 
In the meantime please close qri and re-run this command`, cmdName)
}

func doesCommandExist(cmdName string) bool {
	if cmdName == "" {
		return false
	}
	cmd := exec.Command("/bin/sh", "-c", "command -v "+cmdName)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func printDiff(w io.Writer, res *lib.DiffResponse, summaryOnly bool) (err error) {
	var stats, text string
	// TODO (b5): this reading from a package variable is pretty hacky :/
	if color.NoColor {
		stats = deepdiff.FormatPrettyStats(res.Stat)
		if !summaryOnly {
			text, err = deepdiff.FormatPretty(res.Diff)
			if err != nil {
				return err
			}
		}
	} else {
		stats = deepdiff.FormatPrettyStatsColor(res.Stat)
		if !summaryOnly {
			text, err = deepdiff.FormatPrettyColor(res.Diff)
			if err != nil {
				return err
			}
		}
	}
	buf := bytes.NewBuffer([]byte(stats + "\n" + text))
	printToPager(w, buf)
	return nil
}
