package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewRestoreCommand creates new `qri restore` command
func NewRestoreCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &RestoreOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "restore [DATASET] [VERSION] [COMPONENT]",
		Short: "restore a checked out dataset's files to a previous state",
		Long: `Restore resets some or all the files in a checked out dataset to a previous
state (it does not alter the dataset's history or commits). It will operate on
the current directory unless a DATASET name is specified, in which case it will
alter the directory that dataset is checked out to.

Specify a specific VERSION (e.g. ` + "`/ipfs/QmU...`" + `) to restore to that
version. (Use ` + "`qri log`" + ` to find a version's name.)

Specify a COMPONENT to only restore a particular component (e.g. ` + "`structure`" + `).
Note this is the *component* name, not the file name (e.g. ` + "`structure`" + `,
not ` + "`structure.json`" + `)`,
		Example: `  # Discard all the changes in the current directory:
  $ qri restore
  
  # Reset the files in a directory to an earlier version (note you need to run
  # ` + "`qri save`" + ` afterward to actually save a commit reverting the
  # dataset to this version):
  $ qri restore /ipfs/QmU1grTDSM375BvdNirYLgLTgNkUHPss3FnGxkHHVXwQmk
  
  # Discard just the changes to structure.json:
  $ qri restore structure
  
  # Reset the structure.json file to a specific version:
  $ qri restore /ipfs/QmU1grTDSM375BvdNirYLgLTgNkUHPss3FnGxkHHVXwQmk structure`,
		Annotations: map[string]string{
			"group": "workdir",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	return cmd
}

// RestoreOptions encapsulates state for the `restore` command
type RestoreOptions struct {
	ioes.IOStreams

	Instance *lib.Instance

	Refs          *RefSelect
	Path          string
	ComponentName string
}

// Complete configures the restore command
func (o *RestoreOptions) Complete(f Factory, args []string) (err error) {
	dsRefList := []string{}
	o.Path = ""
	o.ComponentName = ""

	if o.Instance, err = f.Instance(); err != nil {
		return
	}

	// TODO(dlong): Add low-level utilities that parse strings like "peername/ds_name", and
	// "/ipfs/QmFoo", "meta.description", etc and use those everywhere. Use real regexs so
	// that we properly handle user input everywhere. Too much code is duplicating half working
	// input handling for various stringified identifiers.

	// Process arguments to get dataset name, component name, and/or ref path.
	for _, arg := range args {
		if strings.HasPrefix(arg, "/ipfs/") {
			if o.Path != "" {
				return fmt.Errorf("cannot provide more than one ref Path")
			}
			o.Path = arg
			continue
		}

		// Treat "schema" as "structure.schema"
		if arg == "schema" {
			arg = "structure.schema"
		}

		if component.IsDatasetField.MatchString(arg) {
			if o.ComponentName != "" {
				return fmt.Errorf("cannot provide more than one dataset field")
			}
			o.ComponentName = arg
			continue
		}

		pos := strings.Index(arg, "/")
		if pos > -1 {
			if len(dsRefList) != 0 {
				return fmt.Errorf("cannot provide more than one dataset name")
			}
			dsRefList = []string{arg}
			continue
		}

		return fmt.Errorf("unknown argument \"%s\"", arg)
	}

	o.Refs, err = GetCurrentRefSelect(f, dsRefList, 1, EnsureFSIAgrees(o.Instance))
	return
}

// Run executes the `restore` command
func (o *RestoreOptions) Run() (err error) {
	printRefSelect(o.ErrOut, o.Refs)

	ctx := context.TODO()
	inst := o.Instance

	ref := o.Refs.Ref()
	if o.Path != "" {
		ref += o.Path
	}

	params := lib.RestoreParams{
		Refstr:    ref,
		Dir:       o.Refs.Dir(),
		Component: o.ComponentName,
	}

	if err = inst.Filesys().Restore(ctx, &params); err != nil {
		return err
	}
	if o.ComponentName != "" && o.Path == "" {
		printSuccess(o.Out, fmt.Sprintf("Restored %s of dataset %s", o.ComponentName, ref))
	} else if o.Path != "" && o.ComponentName == "" {
		printSuccess(o.Out, fmt.Sprintf("Restored dataset version %s", ref))
	}
	// TODO(dlong): Print message when both component and path are specified.
	return nil
}
