package cmd

import (
	"context"
	"path/filepath"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewAccessCommand creates a new `qri apply` cobra command for applying transformations
func NewAccessCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &AccessOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "access",
		Short:   "manage access to a dataset",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
	}

	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "create an access token",
		RunE:  o.CreateAccessToken,
	}
	tokenCmd.Flags().StringVar(&o.GrantorUsername, "for", "", "user to create access token for")
	tokenCmd.MarkFlagRequired("for")

	cmd.AddCommand(tokenCmd)
	return cmd
}

// AccessOptions encapsulates state for the apply command
type AccessOptions struct {
	ioes.IOStreams

	Refs     *RefSelect
	FilePath string
	Secrets  []string

	GrantorUsername string

	AccessMethods *lib.AccessMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *AccessOptions) Complete(f Factory, args []string) (err error) {
	if o.AccessMethods, err = f.AccessMethods(); err != nil {
		return err
	}
	o.FilePath, err = filepath.Abs(o.FilePath)
	if err != nil {
		return err
	}
	return nil
}

// CreateAccessToken constructs an access token suitable for making
// authenticated requests
func (o *AccessOptions) CreateAccessToken(cmd *cobra.Command, args []string) error {
	ctx := context.TODO()

	p := &lib.CreateTokenParams{
		GrantorUsername: o.GrantorUsername,
	}
	token, err := o.AccessMethods.CreateToken(ctx, p)
	if err != nil {
		return err
	}

	printInfo(o.Out, token)
	return nil
}
