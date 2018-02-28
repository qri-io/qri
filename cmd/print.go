package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"os"
	"strings"
	"time"

	sp "github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/qri-io/dataset"
	"github.com/spf13/cobra"
)

var noColor bool
var printPrompt = color.New(color.FgWhite).PrintfFunc()
var spinner = sp.New(sp.CharSets[24], 100*time.Millisecond)

func setNoColor() {
	color.NoColor = noColor
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

// func PrintValidationErrors(errs map[string][]*history.ValidationError) {
// 	for key, es := range errs {
// 		color.Yellow("%s:", key)
// 		for _, e := range es {
// 			color.Yellow("\t%s", e.String())
// 		}
// 	}
// }

func printDatasetRefInfo(i int, ref repo.DatasetRef) {
	white := color.New(color.FgWhite).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	ds := ref.Dataset

	fmt.Printf("%s  %s\n", cyan(i), white(ref.Name))
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

		if ds.Structure != nil {
			fmt.Printf("    %d bytes, %d entries, %d errors", ds.Structure.Length, ds.Structure.Entries, ds.Structure.ErrCount)
		}
	}

	fmt.Println()

	// fmt.Println("\tfields:")
	// fmt.Printf("\t\t")
	// for _, f := range ds.Fields {
	// 	fmt.Printf("%s|%s\t", cyan(f.Name), blue(f.Type.String()))
	// }
	// fmt.Println()

	// table := tablewriter.NewWriter(os.Stdout)
	// table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	// table.SetCenterSeparator("")
	// table.Append(ds.FieldNames())
	// table.Append(ds.FieldTypeStrings())
	// table.Render()
	// fmt.Println()
}

func printPeerInfo(i int, p *profile.Profile) {
	white := color.New(color.FgWhite).SprintFunc()
	// cyan := color.New(color.FgCyan).SprintFunc()
	// blue := color.New(color.FgBlue).SprintFunc()
	fmt.Printf("peername: %s\n", white(p.Peername))
	fmt.Printf("peerID: %s\n", white(p.ID))
}

// func PrintDatasetDetailedInfo(ds *dataset.Dataset) {
// 	fmt.Println("")
// 	white := color.New(color.FgWhite).SprintFunc()
// 	cyan := color.New(color.FgCyan).SprintFunc()
// 	blue := color.New(color.FgBlue).SprintFunc()
// 	fmt.Printf("\taddress: %s\n", white(ds.Address))
// 	fmt.Printf("\tname: %s\n", white(ds.Name))
// 	if ds.Description != "" {
// 		fmt.Printf("\tdescription: %s\n", white(ds.Description))
// 	}

// 	fmt.Println("\tfields:")
// 	for _, f := range ds.Fields {
// 		fmt.Printf("\t%s", cyan(f.Name))
// 	}
// 	fmt.Printf("\n")
// 	for _, f := range ds.Fields {
// 		fmt.Printf("\t%s", blue(f.Type.String()))
// 	}
// 	fmt.Printf("\n")
// }

func printQuery(i int, r *repo.DatasetRef) {
	white := color.New(color.FgWhite).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	fmt.Printf("%s:\t%s\n\t%s\n", cyan(i), white(r.Dataset.Transform.Data), blue(r.Path))
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

// func PrintTree(ds *dataset.Dataset, indent int) {
// 	fmt.Println(strings.Repeat(" ", indent), ds.Address.String())
// 	for i, d := range ds.Datasets {
// 		if i < len(ds.Datasets)-1 {
// 			fmt.Println(strings.Repeat(" ", indent), "├──", d.Address.String())
// 		} else {
// 			fmt.Println(strings.Repeat(" ", indent), "└──", d.Address.String())

// 		}
// 	}
// }

func prompt(msg string) string {
	var input string
	printPrompt(msg)
	fmt.Scanln(&input)
	return strings.TrimSpace(input)
}

func inputText(message, defaultText string) string {
	if message == "" {
		message = "enter text:"
	}
	input := prompt(fmt.Sprintf("%s [%s]: ", message, defaultText))

	return input
}

/*
	white := color.New(color.FgWhite).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	fmt.Printf("%s:\t%s\n\t%s\n", cyan(i), white(r.Dataset.Transform.Data), blue(r.Path))
*/
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
	// output := ""
	// for _, line := range lines {
	// 	if len(line) >= 3 && (line[:2] == "+ " || line[:2] == "++") {
	// 		output += fmt.Sprintf("🎾%s\n", green(line))
	// 	} else if len(line) >= 3 && (line[:2] == "- " || line[:2] == "--") {
	// 		output += fmt.Sprintf("🏓%s\n", red(line))
	// 	} else {
	// 		output += fmt.Sprintf("%s\n", line)
	// 	}
	// }
	// fmt.Printf("%s", output)
}
