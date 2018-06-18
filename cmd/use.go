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
		Short: "select datasets for use with other commands",
		Example: `  use dataset me/dataset_name, then get meta.title:
  $ qri data me/dataset_name
  $ qri get meta.title

  clear current selection:
  $ qri use --clear

  show current selected dataset references:
  $ qri use --list`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			if o.Clear == false && o.List == false && len(args) == 0 {
				err := cmd.Help()
				ExitIfErr(err)
			}
			ExitIfErr(o.Run())
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
