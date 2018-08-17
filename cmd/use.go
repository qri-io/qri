package cmd

import (
	"fmt"

	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewUseCommand creates a new `qri search` command that searches for datasets
func NewUseCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &UseOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "use",
		Short: "Select datasets for use with the qri get command",
		Example: `  # use dataset me/dataset_name, then get meta.title:
  qri use me/dataset_name
  qri get meta.title

  # clear current selection:
  qri use --clear

  # show current selected dataset references:
  qri use --list`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().BoolVarP(&o.Clear, "clear", "c", false, "clear the current selection")
	cmd.Flags().BoolVarP(&o.List, "list", "l", false, "list selected references")

	return cmd
}

// UseOptions encapsulates state for the search command
type UseOptions struct {
	IOStreams

	Refs  []string
	List  bool
	Clear bool

	SelectionRequests *lib.SelectionRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *UseOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.SelectionRequests, err = f.SelectionRequests()
	return
}

// Validate checks that any user input is valide
func (o *UseOptions) Validate() error {
	if o.Clear == false && o.List == false && len(o.Refs) == 0 {
		return lib.NewError(ErrBadArgs, "please provide dataset name, or --clear flag, or --list flag\nsee `qri use --help` for more info")
	}
	if o.Clear == true && o.List == true || o.Clear == true && len(o.Refs) != 0 || o.List == true && len(o.Refs) != 0 {
		return lib.NewError(ErrBadArgs, "please only give a dataset name, or a --clear flag, or  a --list flag")
	}
	return nil
}

// Run executes the search command
func (o *UseOptions) Run() (err error) {
	var (
		refs []repo.DatasetRef
		res  bool
	)

	if o.List {
		if err = o.SelectionRequests.SelectedRefs(&res, &refs); err != nil {
			return err
		}
	} else if len(o.Refs) > 0 || o.Clear {
		for _, refstr := range o.Refs {
			ref, err := repo.ParseDatasetRef(refstr)
			if err != nil {
				return err
			}
			refs = append(refs, ref)
		}

		if err = o.SelectionRequests.SetSelectedRefs(&refs, &res); err != nil {
			return err
		}

		if len(refs) == 0 {
			printInfo(o.Out, "cleared selected datasets")
			return nil
		}
	}

	for _, ref := range refs {
		fmt.Fprintln(o.Out, ref.String())
	}
	return nil
}
