package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"syscall"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/registry"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

// NewRegistryCommand creates a `qri registry` subcommand for working with the
// configured registry
func NewRegistryCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &RegistryOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "commands for working with a qri registry (qri.cloud)",
		Long: `Registries are federated public records of datasets and peers.
These records form a public facing central lookup for your datasets, so others
can find them through search tools and via web links.

Qri is designed to work without a registry should you want to opt out of
centralized listing entirely, but know that peers who *do* participate in
registries may choose to deprioritize connections with you. Opting out of a
registry is considered an advanced, experimental state at this point.

You can opt out of registries entirely by running:
$ qri config set registry.location ""`,

		Annotations: map[string]string{
			"group": "network",
		},
	}

	// status represents the status command
	status := &cobra.Command{
		Use:   "status DATASET",
		Short: "get the status of a reference on the registry",
		Long:  `Use status to see what version of a dataset the registry has on-record, if any.`,
		Example: `  # Get status of a dataset reference:
  $ qri registry status me/dataset_name`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			// return o.Status()
			return fmt.Errorf("TODO (b5) = restore")
		},
	}

	signup := &cobra.Command{
		Use:   "signup",
		Short: "create a registry profile & connect your local keypair",
		Long: `Signup creates a profile for you on the configured registry.
(qri is configred to use qri.cloud as a registry by default.)

Registry signup reserves a unique username, and connects your local keypair,
allowing your local copy of qri to make authenticated requests on your behalf.

You'll need to sign up before you can use ` + "`qri push`" + ` to push datasets
to a registry.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Signup()
		},
	}

	signup.Flags().StringVar(&o.Username, "username", "", "desired username")
	signup.Flags().StringVar(&o.Email, "email", "", "your email address")
	signup.MarkFlagRequired("username")
	signup.MarkFlagRequired("email")

	prove := &cobra.Command{
		Use:   "prove",
		Short: "authorize your local keypair for an existing registry profile",
		Long: `If you have an existing account on a registry, and local keypair
that is not yet connected to a registry profile, ` + "`prove`" + ` can connect
them.

The prove command connects the local repo to the registry by sending a signed
request to the registry containing login credentials, proving access to both
the unregistred keypair and your registry account. Your repo username will be
matched to the on-registry username.

A repo can only be associated with one registry profile.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Prove()
		},
	}

	prove.Flags().StringVar(&o.Username, "username", "", "your existing registry username")
	prove.Flags().StringVar(&o.Email, "email", "", "your email address")
	prove.MarkFlagRequired("username")
	prove.MarkFlagRequired("email")

	cmd.AddCommand(status, signup, prove)
	return cmd
}

// RegistryOptions encapsulates state for the registry command & subcommands
type RegistryOptions struct {
	ioes.IOStreams
	Refs []string

	Username string
	Password string
	Email    string

	RegistryClientMethods *lib.RegistryClientMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RegistryOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.RegistryClientMethods, err = f.RegistryClientMethods()
	return
}

// Signup registers a handle with the registry
func (o *RegistryOptions) Signup() error {
	password, err := o.PromptForPassword()
	if err != nil {
		return err
	}
	p := &registry.Profile{
		Username: o.Username,
		Email:    o.Email,
		Password: password,
	}

	var ok bool
	if err := o.RegistryClientMethods.CreateProfile(p, &ok); err != nil {
		return err
	}
	printSuccess(o.ErrOut, "user %s created on registry, connected local key", o.Username)
	return nil
}

// Prove associates a keypair with an account
func (o *RegistryOptions) Prove() error {
	password, err := o.PromptForPassword()
	if err != nil {
		return err
	}
	p := &registry.Profile{
		Username: o.Username,
		Email:    o.Email,
		Password: password,
	}

	var ok bool
	if err := o.RegistryClientMethods.ProveProfileKey(p, &ok); err != nil {
		return err
	}
	printSuccess(o.ErrOut, "proved user %s to registry, connected local key", o.Username)
	return nil
}

// PromptForPassword will prompt the user for a password without echoing it to the screen
func (o *RegistryOptions) PromptForPassword() (string, error) {
	io.WriteString(o.Out, "password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	io.WriteString(o.Out, "\n")
	if err != nil {
		// Reading from string buffer fails with one of these errors, depending on operating system
		// "inappropriate ioctl for device"
		// "operation not supported by device"
		if strings.Contains(err.Error(), "device") {
			bytePassword, err = ioutil.ReadAll(o.In)
		} else {
			return "", err
		}
	}
	return string(bytePassword), nil
}
