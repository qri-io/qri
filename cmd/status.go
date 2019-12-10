package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewStatusCommand creates a `qri status` command that compares working directory to prev version
func NewStatusCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &StatusOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Show status of working directory",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			if !o.Refs.IsLinked() {
				return o.RunAtVersion()
			}
			return o.Run()
		},
	}

	cmd.Flags().BoolVar(&o.ShowMtime, "show-mtime", false, "whether to show mtime for each component")

	return cmd
}

// StatusOptions encapsulates state for the Status command
type StatusOptions struct {
	ioes.IOStreams

	Refs      *RefSelect
	ShowMtime bool

	FSIMethods *lib.FSIMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *StatusOptions) Complete(f Factory, args []string) (err error) {
	o.FSIMethods, err = f.FSIMethods()
	if err != nil {
		return err
	}

	o.Refs, err = GetCurrentRefSelect(f, args, 1, o.FSIMethods)
	if err != nil {
		return err
	}

	return nil
}

// ColumnPositionForMtime is the column position at which to display mod times, if requested
const ColumnPositionForMtime = 40

// Run executes the status command
func (o *StatusOptions) Run() (err error) {
	printRefSelect(o.ErrOut, o.Refs)

	res := []lib.StatusItem{}
	dir := o.Refs.Dir()
	if err := o.FSIMethods.Status(&dir, &res); err != nil {
		printErr(o.ErrOut, err)
		return nil
	}

	clean := true
	valid := true
	for _, si := range res {
		line := ""
		switch si.Type {
		case fsi.STRemoved:
			line = fmt.Sprintf("%s:  %s", si.Type, si.Component)
			clean = false
		case fsi.STUnmodified:
			line = ""
		case fsi.STAdd, fsi.STChange:
			line = fmt.Sprintf("%s: %s (source: %s)", si.Type, si.Component, filepath.Base(si.SourceFile))
			clean = false
		default:
			// Represents various error states
			line = fmt.Sprintf("%s: %s (source: %s)", si.Type, si.Component, filepath.Base(si.SourceFile))
			clean = false
			valid = false
		}
		if line != "" {
			if o.ShowMtime && !si.Mtime.IsZero() {
				padding := ""
				if len(line) < ColumnPositionForMtime {
					padding = strings.Repeat(" ", ColumnPositionForMtime-len(line))
				}
				line = fmt.Sprintf("%s%s%s", line, padding, si.Mtime.Format("2006-01-02 15:04:05"))
			}
			printErr(o.Out, fmt.Errorf("  %s", line))
		}
		// TODO(dlong): Validate each file / component, set `valid` to false if any problems exist
	}

	if clean {
		printSuccess(o.Out, "working directory clean")
	} else if valid {
		printSuccess(o.Out, "\nrun `qri save` to commit this dataset")
	} else {
		printErr(o.Out, fmt.Errorf("\nfix these problems before saving this dataset"))
	}
	return nil
}

// RunAtVersion displays status for a reference at a specific version
func (o *StatusOptions) RunAtVersion() (err error) {
	printRefSelect(o.ErrOut, o.Refs)

	res := []lib.StatusItem{}
	ref := o.Refs.Ref()
	if err := o.FSIMethods.StatusAtVersion(&ref, &res); err != nil {
		printErr(o.ErrOut, err)
		return nil
	}

	for _, si := range res {
		printInfo(o.Out, fmt.Sprintf("  %s: %s", si.Component, si.Type))
	}

	return nil
}
