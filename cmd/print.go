package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	sp "github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
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

var printPrompt = color.New(color.FgWhite).PrintfFunc()
var spinner = sp.New(sp.CharSets[24], 100*time.Millisecond)

func setNoColor() {
	color.NoColor = core.Config.CLI.ColorizeOutput
}

func printSuccess(msg string, params ...interface{}) {
	color.Green(msg, params...)
}

func printInfo(msg string, params ...interface{}) {
	color.White(msg, params...)
}

func printWarning(msg string, params ...interface{}) {
	color.Yellow(msg, params...)
}

func printErr(err error, params ...interface{}) {
	color.Red(err.Error(), params...)
}

func printNotYetFinished(cmd *cobra.Command) {
	color.Yellow("%s command is not yet implemented", cmd.Name())
}

func printByteInfo(l int) string {
	length := struct {
		name  string
		value int
	}{"", 0}

	switch {
	// yottabyte and zettabyte overflow int
	// case l > yottabyte:
	// 	length.name = "YB"
	// 	length.value = l / yottabyte
	// case l > zettabyte:
	// 	length.name = "ZB"
	// 	length.value = l / zettabyte
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

func printDatasetRefInfo(i int, ref repo.DatasetRef) {
	white := color.New(color.FgWhite).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	ds := ref.Dataset

	fmt.Printf("%s  %s\n", cyan(i), white(ref.AliasString()))
	fmt.Printf("    %s\n", blue(ref.Path))
	if ds != nil && ds.Meta != nil {
		if ds.Meta.Title != "" {
			fmt.Printf("    %s\n", white(ds.Meta.Title))
		}

		if ds.Meta.Description != "" {
			if len(ds.Meta.Description) > 77 {
				fmt.Printf("    %s...\n", white(ds.Meta.Description[:77]))
			} else {
				fmt.Printf("    %s\n", white(ds.Meta.Description))
			}
		}
	}
	if ds != nil && ds.Structure != nil {
		fmt.Printf("    %s, %d entries, %d errors", printByteInfo(ds.Structure.Length), ds.Structure.Entries, ds.Structure.ErrCount)
	}

	fmt.Println()
}

func printPeerInfo(i int, p *config.ProfilePod) {
	white := color.New(color.FgWhite).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	if p.Online {
		fmt.Printf("%s | %s\n", white(p.Peername), yellow("online"))
	} else {
		fmt.Printf("%s\n", white(p.Peername))
	}
	fmt.Printf("%s\n", blue(p.ID))
	fmt.Printf("%s\n", p.Twitter)
	fmt.Printf("%s\n", p.Description)
	fmt.Println("")
}

func printResults(r *dataset.Structure, data []byte, format dataset.DataFormat) {
	switch format {
	case dataset.JSONDataFormat:
		fmt.Println(string(data))
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
				fmt.Println(err.Error())
				os.Exit(1)
			}

			table.Append(rec)
		}

		table.Render()
	}
}

// TODO - holy shit dis so bad. fix
func terribleHackToGetHeaderRow(st *dataset.Structure) ([]string, error) {
	data, err := st.Schema.MarshalJSON()
	if err != nil {
		return nil, err
	}
	sch := map[string]interface{}{}
	if err := json.Unmarshal(data, &sch); err != nil {
		return nil, err
	}
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

func prompt(msg string) string {
	var input string
	printPrompt(msg)
	fmt.Scanln(&input)
	return strings.TrimSpace(input)
}

func inputText(message, defaultText string) string {
	input := prompt(fmt.Sprintf("%s [%s]: ", message, defaultText))
	if input == "" {
		input = defaultText
	}

	return input
}

func confirm(message string, def bool) bool {
	if noPrompt {
		return def
	}

	yellow := color.New(color.FgYellow).SprintFunc()
	defaultText := "y/N"
	if def {
		defaultText = "Y/n"
	}
	input := prompt(fmt.Sprintf("%s [%s]: ", yellow(message), defaultText))
	if input == "" {
		return def
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return (input == "y" || input == "yes") == def
}

func printDiffs(diffText string) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	lines := strings.Split(diffText, "\n")
	for _, line := range lines {
		if len(line) >= 3 {
			if line[:2] == "+ " || line[:2] == "++" {
				fmt.Printf("%s\n", green(line))
			} else if line[:2] == "- " || line[:2] == "--" {
				fmt.Printf("%s\n", red(line))
			} else {
				fmt.Printf("%s\n", line)
			}
		} else {
			fmt.Printf("%s\n", line)
		}
	}
}
