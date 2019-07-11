package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// FileSelectedRefs stores selection, is copied from github.com/qri-io/qri/repo/fs/files.go
const FileSelectedRefs = "/selected_refs.json"

// NewUseCommand creates a new `qri search` command that searches for datasets
func NewUseCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &UseOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "use",
		Short: "Select datasets for use with the qri get command",
		Long: `
Run the ` + "`use`" + ` command to have Qri remember references to a specific datasets. 
These datasets will be referenced for future commands, if no dataset reference 
is explicitly given for those commands.

We created this command to ease the typing/copy and pasting burden while using
Qri to explore a dataset.`,
		Example: `  # use dataset me/dataset_name, then get meta.title:
  qri use me/dataset_name
  qri get meta.title

  # clear current selection:
  qri use --clear

  # show current selected dataset references:
  qri use --list

  # add multiple references to the remembered list
  qri use me/population_2017 me/population_2018`,
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
	ioes.IOStreams

	Refs  []string
	List  bool
	Clear bool

	QriRepoPath string
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *UseOptions) Complete(f Factory, args []string) (err error) {
	o.QriRepoPath = f.QriRepoPath()
	o.Refs = args
	return
}

// Validate checks that any user input is valide
func (o *UseOptions) Validate() error {
	if o.Clear == false && o.List == false && len(o.Refs) == 0 {
		return lib.NewError(lib.ErrBadArgs, "please provide dataset name, or --clear flag, or --list flag\nsee `qri use --help` for more info")
	}
	if o.Clear == true && o.List == true || o.Clear == true && len(o.Refs) != 0 || o.List == true && len(o.Refs) != 0 {
		return lib.NewError(lib.ErrBadArgs, "please only give a dataset name, or a --clear flag, or  a --list flag")
	}
	return nil
}

// Run executes the search command
func (o *UseOptions) Run() (err error) {
	var refs []repo.DatasetRef
	fileSelectionPath := filepath.Join(o.QriRepoPath, FileSelectedRefs)

	if o.List {
		refs, err = readFile(fileSelectionPath)
		if err != nil {
			// File not exist, or can't parse: not an error, just don't show anything.
			return nil
		}
	} else if o.Clear {
		err := writeFile(fileSelectionPath, refs)
		if err != nil {
			return err
		}
		printInfo(o.Out, "cleared selected datasets")
	} else if len(o.Refs) > 0 {
		for _, refstr := range o.Refs {
			ref, err := repo.ParseDatasetRef(refstr)
			if err != nil {
				return err
			}
			refs = append(refs, ref)
		}

		err := writeFile(fileSelectionPath, refs)
		if err != nil {
			return err
		}
	}

	for _, ref := range refs {
		fmt.Fprintln(o.Out, ref.String())
	}
	return nil
}

// GetDatasetRefString returns the arg at the index, or otherwise the first selected reference
func GetDatasetRefString(f Factory, args []string, index int) (string, error) {
	// If reference is specified by the user provide command-line arguments, use that reference.
	if index < len(args) {
		return args[index], nil
	}
	
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// If in a working directory that is linked to a dataset, use that link's reference.
	data, ok := fsi.GetLinkedFilesysRef(wd)
	if ok {
		return data, nil
	}
	// Find what `use` is referencing and use that.
	refs, err := DefaultSelectedRefList(f)
	if err != nil {
		return "", err
	}
	if len(refs) == 0 {
		// If selected_refs.json is empty or doesn't exist, not an error.
		return "", nil
	}
	return refs[0], nil
}

// DefaultSelectedRefList returns the list of currently selected dataset references
func DefaultSelectedRefList(f Factory) ([]string, error) {
	fileSelectionPath := filepath.Join(f.QriRepoPath(), FileSelectedRefs)

	refs, err := readFile(fileSelectionPath)
	if err != nil {
		// If selected_refs.json is empty or doesn't exist, not an error.
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	res := make([]string, 0, len(refs))
	for _, r := range refs {
		res = append(res, r.String())
	}

	return res, nil
}

// writeFile serializes the list of refs to a file at path
func writeFile(path string, refs []repo.DatasetRef) error {
	data, err := json.Marshal(refs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, os.ModePerm)
}

// readFile deserializes a list of refs from a file at path
func readFile(path string) ([]repo.DatasetRef, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	res := []repo.DatasetRef{}
	if err = json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}
