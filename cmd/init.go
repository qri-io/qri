package cmd

import (
	"os"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/qri-io/ioes"
	"github.com/spf13/cobra"
)

// NewInitCommand creates new `qri init` command that connects a directory to 
// the local filesystem
func NewInitCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &InitOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"ls"},
		Short:   "initialize a dataset directory",
		Long: ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run()
		},
	}

	cmd.PersistentFlags().StringVar(&o.Peername, "peername", "me", "ref peername")

	return cmd
}

// InitOptions encapsulates state for the List command
type InitOptions struct {
	ioes.IOStreams

	Peername string
}

// Run executes the init command
func (o *InitOptions) Run() (err error) {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(".qri_ref", []byte(fmt.Sprintf("%s/%s", o.Peername, filepath.Base(pwd))), os.ModePerm)
}
