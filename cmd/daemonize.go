package cmd

import (
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewDaemonizeCommand creates a new daemonize command
func NewDaemonizeCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &DaemonizeOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "daemonize",
		Short: "Load or interact with the qri daemon",
		Long: `
Load or interact with the qri daemon. The available commands are "install", "uninstall", "show"`,
		Example: `  # install the daemon:
  qri daemonize install

  # uninstall the daemon:
  qri daemonize uninstall

  # show the daemon's status:
  qri daemonize show`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}
	return cmd
}

// DaemonizeOptions holds data for daemonize commands
type DaemonizeOptions struct {
	ioes.IOStreams

	Action string

	DaemonizeRequests *lib.DaemonizeRequests
}

// Complete completes the daemonize command
func (o *DaemonizeOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		o.Action = args[0]
	}
	o.DaemonizeRequests, err = f.DaemonizeRequests()
	return err
}

// Validate validates the daemonize command
func (o *DaemonizeOptions) Validate() error {
	return nil
}

// Run runs the daemonize command
func (o *DaemonizeOptions) Run() error {
	params := lib.DaemonizeParams{
		Action: o.Action,
	}
	var out bool
	if err := o.DaemonizeRequests.Daemonize(&params, &out); err != nil {
		return err
	}
	return nil
}
