package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewCheckoutCommand creates new `qri checkout` command that connects a working directory in
// the local filesystem to a dataset your repo.
func NewCheckoutCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &CheckoutOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "checkout",
		Short:   "checkout created a linked directory and writes dataset files to that directory",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Complete(f, args)
			return o.Run()
		},
	}

	return cmd
}

// CheckoutOptions encapsulates state for the `checkout` command
type CheckoutOptions struct {
	ioes.IOStreams

	Args []string

	DatasetRequests *lib.DatasetRequests
	FSIMethods      *lib.FSIMethods
}

// Complete completes a the command
func (o *CheckoutOptions) Complete(f Factory, args []string) (err error) {
	o.Args = args
	if o.DatasetRequests, err = f.DatasetRequests(); err != nil {
		return err
	}
	o.FSIMethods, err = f.FSIMethods()
	return err
}

// Run executes the `checkout` command
func (o *CheckoutOptions) Run() (err error) {
	// TODO: Finalize UI for command-line checkout command.
	ref := o.Args[0]

	// Derive directory name from the dataset name.
	pos := strings.Index(ref, "/")
	if pos == -1 {
		return fmt.Errorf("expect '/' in dataset ref")
	}
	folderName := ref[pos+1:]

	// If directory exists, error.
	if _, err = os.Stat(folderName); !os.IsNotExist(err) {
		return fmt.Errorf("directory with name \"%s\" already exists", folderName)
	}

	// Get the dataset from your repo.
	p := lib.GetParams{
		Path:     ref,
		Selector: "",
	}
	res := lib.GetResult{}
	if err = o.DatasetRequests.Get(&p, &res); err != nil {
		return err
	}

	// Create a directory.
	if err = os.Mkdir(folderName, os.ModePerm); err != nil {
		return err
	}

	// Create the link file, containing the dataset reference.
	lnkp := &lib.LinkParams{
		Dir: folderName,
		Ref: ref,
	}
	lnkres := ""
	if err = o.FSIMethods.CreateLink(lnkp, &lnkres); err != nil {
		return err
	}

	// Prepare dataset.
	// TODO(dlong): Move most of this into FSI?
	ds := res.Dataset

	// Get individual components out of the dataset.
	meta := ds.Meta
	ds.Meta = nil
	schema := ds.Structure.Schema
	ds.Structure.Schema = nil

	// Structure is kept in the dataset.
	bodyFormat := ds.Structure.Format
	ds.Structure.Format = ""
	ds.Structure.Qri = ""

	// Commit, viz, transform are never checked out.
	ds.Commit = nil
	ds.Viz = nil
	ds.Transform = nil

	// Meta component.
	if meta != nil {
		meta.DropDerivedValues()
		if !meta.IsEmpty() {
			data, err := json.MarshalIndent(meta, "", " ")
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(filepath.Join(folderName, "meta.json"), data, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}

	// Schema component.
	if len(schema) > 0 {
		data, err := json.MarshalIndent(schema, "", " ")
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(folderName, "schema.json"), data, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Body component.
	bf := ds.BodyFile()
	data, err := ioutil.ReadAll(bf)
	if err != nil {
		return err
	}
	ds.BodyPath = ""
	var bodyFilename string
	switch bodyFormat {
	case "csv":
		bodyFilename = "body.csv"
	case "json":
		bodyFilename = "body.json"
	default:
		return fmt.Errorf("unknown body format: %s", bodyFormat)
	}
	err = ioutil.WriteFile(filepath.Join(folderName, bodyFilename), data, os.ModePerm)
	if err != nil {
		return err
	}

	// Dataset (everything else).
	ds.DropDerivedValues()
	// TODO(dlong): Should more of these move to DropDerivedValues?
	ds.Qri = ""
	ds.Name = ""
	ds.Peername = ""
	ds.PreviousPath = ""
	if ds.Structure.IsEmpty() {
		ds.Structure = nil
	}
	if !ds.IsEmpty() {
		data, err := json.MarshalIndent(ds, "", " ")
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(folderName, "dataset.json"), data, os.ModePerm)
		if err != nil {
			return err
		}
	}

	printSuccess(o.Out, "created and linked working directory %s for existing dataset", folderName)
	return nil
}
