package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewListCommand creates new `qri list` command that lists datasets for the local peer & others
func NewListCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ListOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "list [FILTER]",
		Aliases: []string{"ls"},
		Short:   "show a list of datasets",
		Long: `List shows lists of datasets, including names and current hashes. 

The default list is the latest version of all datasets you have on your local 
qri repository. The first argument can be used to find datasets with a certain 
substring in their name.

When used in conjunction with ` + "`qri connect`" + `, list can list a peer's dataset. You
must have ` + "`qri connect`" + ` running in a separate terminal window.`,
		Example: `  # Show all of your datasets:
  $ qri list

  # Show datasets with the substring "new" in their name:
  $ qri list new

  # To view the list of a peer's datasets...
  # In one terminal window:
  $ qri connect
  # In a separate terminal window, show all of b5's datasets:
  $ qri list --peer b5`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json|simple]")
	cmd.Flags().IntVar(&o.Offset, "offset", 0, "skip this number of records from the results, default 0")
	cmd.Flags().IntVar(&o.Limit, "limit", 25, "size of results, default 25")
	cmd.Flags().BoolVar(&o.All, "all", false, "get all results")
	cmd.Flags().BoolVarP(&o.Public, "public", "p", false, "list only publically visible")
	cmd.Flags().BoolVarP(&o.ShowNumVersions, "num-versions", "n", false, "show number of versions")
	cmd.Flags().StringVar(&o.Username, "user", "", "user whose datasets to list")
	cmd.MarkFlagCustom("user", "__qri_get_user_flag_suggestions")
	cmd.Flags().BoolVarP(&o.Raw, "raw", "r", false, "to show raw references")

	return cmd
}

// ListOptions encapsulates state for the List command
type ListOptions struct {
	ioes.IOStreams

	Format          string
	Offset          int
	Limit           int
	All             bool
	Term            string
	Username        string
	Public          bool
	ShowNumVersions bool
	Raw             bool

	inst *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ListOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		o.Term = args[0]
	}
	o.inst, err = f.Instance()
	return
}

// Run executes the list command
func (o *ListOptions) Run() (err error) {
	ctx := context.TODO()

	if o.Raw {
		text, err := o.inst.Collection().ListRawRefs(ctx, &lib.EmptyParams{})
		if err != nil {
			return err
		}
		printSuccess(o.Out, text)
		return nil
	}

	p := &lib.CollectionListParams{
		Term:     o.Term,
		Username: o.Username,
		List: params.List{
			Offset: o.Offset,
			Limit:  o.Limit,
		},
		Public: o.Public,
	}
	infos, cur, err := o.inst.Collection().List(ctx, p)
	if err != nil {
		if errors.Is(err, lib.ErrListWarning) {
			printWarning(o.ErrOut, fmt.Sprintf("%s\n", err))
			err = nil
		} else {
			return err
		}
	}

	// TODO(dustmop): Generics (Go1.17?) will make this refactorable
	// Consume the entire Cursor to list all references
	if cur != nil && o.All {
		isDone := false
		for {
			more, err := cur.Next(ctx)
			if err == lib.ErrCursorComplete {
				// Don't break just yet, `more` may have remaining items to append
				isDone = true
			} else if err != nil {
				return err
			}
			if vals, ok := more.([]dsref.VersionInfo); ok {
				if len(vals) == 0 {
					isDone = true
				}
				infos = append(infos, vals...)
			}
			if isDone {
				break
			}
		}
	}

	for _, ref := range infos {
		// remove profileID so names print pretty
		ref.ProfileID = ""
	}

	if len(infos) == 0 {
		pn := fmt.Sprintf("%s has", o.Username)
		if o.Username == "" {
			pn = "you have"
		}

		if o.Term == "" {
			printInfo(o.Out, "%s no datasets", pn)
		} else {
			printInfo(o.Out, "%s no datasets that match \"%s\"", pn, o.Term)
		}
		return
	}

	switch o.Format {
	case "":
		items := make([]fmt.Stringer, len(infos))
		for i, r := range infos {
			items[i] = versionInfoStringer(r)
		}
		printItems(o.Out, items, o.Offset)
		return nil
	case "simple":
		items := make([]string, len(infos))
		for i, r := range infos {
			items[i] = r.SimpleRef().Alias()
		}
		printlnStringItems(o.Out, items)
		return nil
	case dataset.JSONDataFormat.String():
		data, err := json.MarshalIndent(infos, "", "  ")
		if err != nil {
			return err
		}
		buf := bytes.NewBuffer(data)
		printToPager(o.Out, buf)
		return nil
	default:
		return fmt.Errorf("unrecognized format: %s", o.Format)
	}
}
