package cmd

import (
	"github.com/qri-io/dsdiff"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewDiffCommand creates a new `qri diff` cobra command for comparing changes between datasets
func NewDiffCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := DiffOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare differences between two datasets",
		Long: `
Diff compares two datasets from your repo and prints a representation 
of the differences between them.  You can specifify the datasets
either by name or by their hash. You can compare different versions of 
the same dataset.`,
		Example: `  show diff between two versions of the same dataset:
  $ qri diff me/annual_pop@/ipfs/QmcBZoEQ7ot4UYKn1JM3gwd4LHorj6FJ4Ep19rfLBT3VZ8 
  me/annual_pop@/ipfs/QmVvqsge5wqp4piJbLArwVB6iJSTrdM8ZRpHY7fikASrr8

  show diff between two different datasets:
  $ qri diff me/population_2016 me/population_2017`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Display, "display", "d", "", "set display format [reg|short|delta|detail]")
	// datasetDiffCmd.Flags().BoolP("color", "c", false, "set ")

	return cmd
}

// DiffOptions encapsulates options for the diff command
type DiffOptions struct {
	IOStreams

	Display string
	Left    string
	Right   string

	UsingRPC        bool
	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *DiffOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 1 {
		o.Left = args[0]
		o.Right = args[1]
	}
	if len(args) == 1 {
		o.Right = args[0]
	}
	o.UsingRPC = f.RPC() != nil
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the diff command
func (o *DiffOptions) Run() error {
	if o.UsingRPC {
		return usingRPCError("diff")
	}

	left, err := repo.ParseDatasetRef(o.Left)
	if err != nil && err != repo.ErrEmptyRef {
		return err
	}
	right, err := repo.ParseDatasetRef(o.Right)
	if err != nil && err != repo.ErrEmptyRef {
		return err
	}

	diffs := make(map[string]*dsdiff.SubDiff)
	p := &lib.DiffParams{
		Left:    left,
		Right:   right,
		DiffAll: true,
	}

	if err = o.DatasetRequests.Diff(p, &diffs); err != nil {
		return err
	}

	displayFormat := "listKeys"
	switch o.Display {
	case "reg", "regular":
		displayFormat = "listKeys"
	case "short", "s":
		displayFormat = "simple"
	case "delta":
		displayFormat = "delta"
	case "detail":
		displayFormat = "plusMinus"
	}

	result, err := dsdiff.MapDiffsToString(diffs, displayFormat)
	if err != nil {
		return err
	}

	printDiffs(o.Out, result)
	return nil
}
