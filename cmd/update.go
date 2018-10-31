package cmd

import (
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewUpdateCommand creates a new `qri update` cobra command for updating datasets
func NewUpdateCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &UpdateOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "update",
		Short: "add/create the lastest version of a dataset",
		Long: `
Update fast-forwards your dataset to the latest known version. If the dataset
is not in your namespace (i.e. dataset name doesn't start with your peername), 
update will ask the peer for any new versions and download them. Updating a peer
dataset accepts no arguments other than the datsaet name and --dry-run flag.

**For peer update to work, the peer must be online at the time. We know this is
irritating, we're working on a solution.**

Calling update on a dataset in your namespace will advance your dataset by 
re-running any specified transform script, creating a new version of your 
dataset in the process. If your dataset doesn't have a transform script, update 
will error.`,
		Example: `  # get the freshest version of a dataset from a peer
  qri update other_person/dataset

  # update your local dataset by re-running the dataset transform
  qri update me/dataset_with_transform

  # supply secrets to an update, publish on successful run
  qri update me/dataset_with_transform -p --secrets=keyboard,cat`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Title, "title", "t", "", "title of commit message for update")
	cmd.Flags().StringVarP(&o.Message, "message", "m", "", "commit message for update")
	cmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	// cmd.Flags().BoolVarP(&o.Publish, "publish", "p", false, "publish this dataset to the registry")
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "simulate updating a dataset")

	return cmd
}

// UpdateOptions encapsulates state for the update command
type UpdateOptions struct {
	ioes.IOStreams

	Ref     string
	Title   string
	Message string
	Publish bool
	DryRun  bool
	Secrets []string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *UpdateOptions) Complete(f Factory, args []string) (err error) {
	if len(args) == 1 {
		o.Ref = args[0]
	}
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Validate checks that all user input is valid
func (o *UpdateOptions) Validate() error {
	if o.Ref == "" {
		return lib.NewError(lib.ErrBadArgs, "please provide a dataset reference for updating")
	}
	return nil
}

// Run executes the update command
func (o *UpdateOptions) Run() (err error) {
	var secrets map[string]string

	if o.Secrets != nil {
		if !confirm(o.Out, o.In, `
			Warning: You are providing secrets to a dataset transformation.
			Never provide secrets to a transformation you do not trust.
			continue?`, true) {
			return
		}

		secrets, err = parseSecrets(o.Secrets...)
		if err != nil {
			return err
		}
	}

	p := &lib.UpdateParams{
		Ref:        o.Ref,
		Title:      o.Title,
		Message:    o.Message,
		DryRun:     o.DryRun,
		Publish:    o.Publish,
		Secrets:    secrets,
		ReturnBody: false,
	}

	res := &repo.DatasetRef{}
	if err := o.DatasetRequests.Update(p, res); err != nil {
		return err
	}

	printSuccess(o.Out, "updated dataset %s", res.AliasString())
	return nil
}
