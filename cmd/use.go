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
		Long:  ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Run())
		},
	}

	return cmd
}

// UseOptions encapsulates state for the search command
type UseOptions struct {
	IOStreams

	Refs []string

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

	for _, ref := range refs {
		fmt.Fprintln(o.Out, ref.String())
	}
	return nil
}
