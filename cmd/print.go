package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
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

func printNotYetFinished(cmd *cobra.Command) {
	color.Yellow("%s command is not yet implemented", cmd.Name())
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
	// check if more exist, use it
	// check if less exists, use it
	if envPager == "" {
		envPager = "more"
	}
	pager := exec.Command(envPager, "-R")
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

func printSearchResult(w io.Writer, i int, result lib.SearchResult) {
	white := color.New(color.FgWhite).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	ds := &dataset.Dataset{}
	if data, err := json.Marshal(result.Value); err == nil {
		if err = json.Unmarshal(data, ds); err == nil {
			fmt.Fprintf(w, "%s. %s\n", white(i+1), white(result.ID))
			if ds.Meta != nil && ds.Meta.Title != "" {
				fmt.Fprintf(w, "   %s\n", green(ds.Meta.Title))
			}
			if ds.Structure != nil {
				fmt.Fprintf(w, "   %s, %d entries, %d errors\n", printByteInfo(ds.Structure.Length), ds.Structure.Entries, ds.Structure.ErrCount)
			}
		}
	}
	fmt.Fprintln(w)
}

func printPeerInfo(w io.Writer, i int, p *config.ProfilePod) {
	white := color.New(color.FgWhite).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	if p.Online {
		fmt.Fprintf(w, "%s | %s\n", white(p.Peername), yellow("online"))
	} else {
		fmt.Fprintf(w, "%s\n", white(p.Peername))
	}
	fmt.Fprintf(w, "profile ID: %s\n", blue(p.ID))
	if len(p.NetworkAddrs) > 0 {
		fmt.Fprintf(w, "address:    %s\n", p.NetworkAddrs[0])
	}
	fmt.Fprintln(w, "")
}

func printPeerInfoNoColor(w io.Writer, i int, p *config.ProfilePod) {
	if p.Online {
		fmt.Fprintf(w, "%s | %s\n", p.Peername, "online")
	} else {
		fmt.Fprintf(w, "%s\n", p.Peername)
	}
	fmt.Fprintf(w, "profile ID: %s\n", p.ID)
	if len(p.NetworkAddrs) > 0 {
		fmt.Fprintf(w, "address:    %s\n", p.NetworkAddrs[0])
	}
	fmt.Fprintln(w, "")
}

func printResults(w io.Writer, r *dataset.Structure, data []byte, format dataset.DataFormat) {
	switch format {
	case dataset.JSONDataFormat:
		fmt.Fprintln(w, string(data))
	case dataset.CSVDataFormat:
		table := tablewriter.NewWriter(os.Stdout)
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		hr, err := terribleHackToGetHeaderRow(r)
		if err == nil {
			table.SetHeader(hr)
		}
		r := csv.NewReader(bytes.NewBuffer(data))
		for {
			rec, err := r.Read()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				fmt.Fprintln(w, err.Error())
				os.Exit(1)
			}

			table.Append(rec)
		}

		table.Render()
	}
}

// TODO - holy shit dis so bad. fix
func terribleHackToGetHeaderRow(st *dataset.Structure) ([]string, error) {
	sch := st.Schema
	if itemObj, ok := sch["items"].(map[string]interface{}); ok {
		if itemArr, ok := itemObj["items"].([]interface{}); ok {
			titles := make([]string, len(itemArr))
			for i, f := range itemArr {
				if field, ok := f.(map[string]interface{}); ok {
					if title, ok := field["title"].(string); ok {
						titles[i] = title
					}
				}
			}
			return titles, nil
		}
	}
	return nil, fmt.Errorf("nope")
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

func printDiffs(w io.Writer, diffText string) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	lines := strings.Split(diffText, "\n")
	for _, line := range lines {
		if len(line) >= 3 {
			if line[:2] == "+ " || line[:2] == "++" {
				fmt.Fprintf(w, "%s\n", green(line))
			} else if line[:2] == "- " || line[:2] == "--" {
				fmt.Fprintf(w, "%s\n", red(line))
			} else {
				fmt.Fprintf(w, "%s\n", line)
			}
		} else {
			fmt.Fprintf(w, "%s\n", line)
		}
	}
}

func usingRPCError(cmdName string) error {
	return fmt.Errorf(`sorry, we can't run the '%s' command while 'qri connect' is running
we know this is super irritating, and it'll be fixed in the future. 
In the meantime please close qri and re-run this command`, cmdName)
}
