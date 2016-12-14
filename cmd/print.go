// print gathers all tools for formatting output
package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/qri-io/dataset"
	"github.com/qri-io/history"
	"github.com/spf13/cobra"
)

var noColor bool
var printPrompt = color.New(color.FgWhite).PrintfFunc()

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

func PrintDatasetShortInfo(ds *dataset.Dataset) {
	white := color.New(color.FgWhite).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	fmt.Println()
	fmt.Printf("dataset: %s\n", white(ds.Address))
	fmt.Printf("description: %s\n", white(ds.Description))

	fmt.Println("fields:")
	for _, f := range ds.Fields {
		fmt.Printf("\t%s\t%s", cyan(f.Name), blue(f.Type.String()))
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
