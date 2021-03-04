package cmd

import (
	"context"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewAccessCommand creates a new `qri access` cobra command for managing
// permissions
func NewAccessCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &AccessOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "access",
		Short:   "manage user permissions",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "other",
		},
	}

	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "create an access token",
		Long: `
token creates a JSON Web Token (JWT) that authenticates the given user.
Constructing an access token requires a private key that backs the given user.

In the course of normal operation you shouldn't need this command, It's mainly
here for crafting API requests in external progrmas`[1:],
		Example: `
  # create an access token to authenticate yourself else where:
  $ qri access token --for me
`[1:],
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			ctx := context.TODO()
			return o.CreateAccessToken(ctx)
		},
	}
	tokenCmd.Flags().StringVar(&o.GranteeUsername, "for", "", "user to create access token for")
	tokenCmd.MarkFlagRequired("for")

	cmd.AddCommand(tokenCmd)
	return cmd
}

// AccessOptions encapsulates state for the apply command
type AccessOptions struct {
	ioes.IOStreams
	Instance *lib.Instance

	GranteeUsername string
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *AccessOptions) Complete(f Factory, args []string) (err error) {
	o.Instance = f.Instance()
	return nil
}

// CreateAccessToken constructs an access token suitable for making
// authenticated requests
func (o *AccessOptions) CreateAccessToken(ctx context.Context) error {
	p := &lib.CreateAuthTokenParams{
		GranteeUsername: o.GranteeUsername,
	}
	token, err := o.Instance.Access().CreateAuthToken(ctx, p)
	if err != nil {
		return err
	}

	printInfo(o.Out, token)
	return nil
}
