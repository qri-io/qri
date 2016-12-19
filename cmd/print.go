// print gathers all tools for formatting output
package cmd

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	sp "github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/qri-io/dataset"
	"github.com/qri-io/history"
	"github.com/qri-io/namespace"
	"github.com/spf13/cobra"
)

var noColor bool
var printPrompt = color.New(color.FgWhite).PrintfFunc()
var spinner = sp.New(sp.CharSets[24], 100*time.Millisecond)

func SetNoColor() {
	color.NoColor = noColor
}

func PrintSuccess(msg string, params ...interface{}) {
	color.Green(msg, params...)
}

func PrintInfo(msg string, params ...interface{}) {
	color.White(msg, params...)
}

func PrintWarning(msg string, params ...interface{}) {
	color.Yellow(msg, params...)
}

// TODO - remove this shit. wtf, PrintRed!?
func PrintRed(msg string, params ...interface{}) {
	color.Red(msg, params...)
}

func PrintErr(err error, params ...interface{}) {
	color.Red(err.Error(), params...)
}

func PrintNotYetFinished(cmd *cobra.Command) {
	color.Yellow("%s command is not yet implemented", cmd.Name())
}

func PrintValidationErrors(errs map[string][]*history.ValidationError) {
	for key, es := range errs {
		color.Yellow("%s:", key)
		for _, e := range es {
			color.Yellow("\t%s", e.String())
		}
	}
}

func PrintNamespace(ns namespace.Namespace) {
	color.Cyan("%s: ", ns.String())
}

func PrintDatasetShortInfo(i int, ds *dataset.Dataset) {
	white := color.New(color.FgWhite).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	fmt.Printf("%s\tdataset: %s\n", cyan(i), white(ds.Address))
	fmt.Printf("\tdescription: %s\n", white(ds.Description))

	fmt.Println("\tfields:")
	fmt.Printf("\t\t")
	for _, f := range ds.Fields {
		fmt.Printf("%s|%s\t", cyan(f.Name), blue(f.Type.String()))
	}
	fmt.Println()
	// table := tablewriter.NewWriter(os.Stdout)
	// table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	// table.SetCenterSeparator("")
	// table.Append(ds.FieldNames())
	// table.Append(ds.FieldTypeStrings())
	// table.Render()
	// fmt.Println()
}

func PrintDatasetDetailedInfo(ds *dataset.Dataset) {
	fmt.Println("")
	white := color.New(color.FgWhite).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	fmt.Printf("\taddress: %s\n", white(ds.Address))
	fmt.Printf("\tname: %s\n", white(ds.Name))
	if ds.Description != "" {
		fmt.Printf("\tdescription: %s\n", white(ds.Description))
	}

	fmt.Println("\tfields:")
	for _, f := range ds.Fields {
		fmt.Printf("\t%s", cyan(f.Name))
	}
	fmt.Printf("\n")
	for _, f := range ds.Fields {
		fmt.Printf("\t%s", blue(f.Type.String()))
	}
	fmt.Printf("\n")
}

func PrintResults(ds *dataset.Dataset, data []byte, format dataset.DataFormat) {
	switch format {
	case dataset.JsonDataFormat:
		fmt.Println()
		fmt.Println(string(data))
	case dataset.CsvDataFormat:
		fmt.Println()
		table := tablewriter.NewWriter(os.Stdout)
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.SetHeader(ds.FieldNames())
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

func PrintTree(ds *dataset.Dataset, indent int) {
	fmt.Println(strings.Repeat(" ", indent), ds.Address.String())
	for i, d := range ds.Datasets {
		if i < len(ds.Datasets)-1 {
			fmt.Println(strings.Repeat(" ", indent), "├──", d.Address.String())
		} else {
			fmt.Println(strings.Repeat(" ", indent), "└──", d.Address.String())

		}
	}
}
