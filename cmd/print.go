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
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

const (
	bite = 1 << (10 * iota)
	kilobyte
	megabyte
	gigabyte
	terabyte
	petabyte
	exabyte
	zettabyte
	yottabyte
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
	fmt.Fprintln(w, color.New(color.FgWhite).Sprintf(msg, params...))
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

func printItems(w io.Writer, items []fmt.Stringer) (err error) {
	buf := &bytes.Buffer{}
	prefix := []byte("    ")
	for i, item := range items {
		buf.WriteString(fmtItem(i+1, item.String(), prefix))
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

func printByteInfo(n int) string {
	// Use 64-bit ints to support platforms on which int is not large enough to represent
	// the constants below (exabyte, petabyte, etc). For example: Raspberry Pi running arm6.
	l := int64(n)
	length := struct {
		name  string
		value int64
	}{"", 0}

	switch {
	// yottabyte and zettabyte overflow int
	// case l > yottabyte:
	//  length.name = "YB"
	//  length.value = l / yottabyte
	// case l > zettabyte:
	//  length.name = "ZB"
	//  length.value = l / zettabyte
	case l >= exabyte:
		length.name = "EB"
		length.value = l / exabyte
	case l >= petabyte:
		length.name = "PB"
		length.value = l / petabyte
	case l >= terabyte:
		length.name = "TB"
		length.value = l / terabyte
	case l >= gigabyte:
		length.name = "GB"
		length.value = l / gigabyte
	case l >= megabyte:
		length.name = "MB"
		length.value = l / megabyte
	case l >= kilobyte:
		length.name = "KB"
		length.value = l / kilobyte
	default:
		length.name = "byte"
		length.value = l
	}
	if length.value != 1 {
		length.name += "s"
	}
	return fmt.Sprintf("%v %s", length.value, length.name)
}

func printDatasetRefInfo(w io.Writer, i int, ref repo.DatasetRef) {
	white := color.New(color.FgWhite).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	ds := ref.Dataset

	fmt.Fprintf(w, "%s  %s\n", cyan(i), white(ref.AliasString()))
	if ds != nil && ds.Meta != nil && ds.Meta.Title != "" {
		fmt.Fprintf(w, "    %s\n", blue(ds.Meta.Title))
	}
	if ref.Path != "" {
		fmt.Fprintf(w, "    %s\n", ref.Path)
	}
	if ds != nil && ds.Structure != nil {
		fmt.Fprintf(w, "    %s", printByteInfo(ds.Structure.Length))
		if ds.Structure.Entries == 1 {
			fmt.Fprintf(w, ", %d entry", ds.Structure.Entries)
		} else {
			fmt.Fprintf(w, ", %d entries", ds.Structure.Entries)
		}
		if ds.Structure.ErrCount == 1 {
			fmt.Fprintf(w, ", %d error", ds.Structure.ErrCount)
		} else {
			fmt.Fprintf(w, ", %d errors", ds.Structure.ErrCount)
		}
		if ds.NumVersions == 0 {
			// nothing
		} else if ds.NumVersions == 1 {
			fmt.Fprintf(w, ", %d version", ds.NumVersions)
		} else {
			fmt.Fprintf(w, ", %d versions", ds.NumVersions)
		}
	}

	fmt.Fprintf(w, "\n")
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
