// print gathers all tools for formatting output
package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/qri-io/dataset"
	"github.com/qri-io/history"
	"github.com/spf13/cobra"
)

var noColor bool

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

func PrintDatasetShortInfo(ds *dataset.Dataset) {
	fmt.Printf("dataset: %s\n", ds.Address)
	if ds.Description != "" {
		fmt.Printf("description:\n%s", ds.Description)
	}

	fmt.Println("fields:")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.SetCenterSeparator("")
	table.Append(ds.FieldNames())
	table.Append(ds.FieldTypeStrings())
	table.Render()
	fmt.Println()
}

func PrintDatasetDetailedInfo(ds *dataset.Dataset) {
	fmt.Printf("dataset: %s\n", ds.Address)
	if ds.Description != "" {
		fmt.Printf("description:\n%s", ds.Description)
	}

	fmt.Println("fields:")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.SetCenterSeparator("")
	table.Append(ds.FieldNames())
	table.Append(ds.FieldTypeStrings())
	table.Render()
	fmt.Println()
}
