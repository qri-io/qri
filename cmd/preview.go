package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewPreviewCommand creates a `qri preview` subcommand for fetching dataset
// prewviews
func NewPreviewCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &PreviewOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "preview [DATASET]",
		Short: "fetch a dataset preview",
		Long: `Preview fetches a summary of a dataset but doesn't store it. Useful
for investigating a dataset before saving it locally.
`,
		Example: `  # Preview a dataset:
  $ qri preview user/dataset`,
		Annotations: map[string]string{
			"group": "network",
		},
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "", "pretty", "output format [pretty|json]")
	cmd.Flags().StringVarP(&o.Source, "source", "", "", "name of source to fetch preview from")

	return cmd
}

// PreviewOptions encapsulates state for the publish command
type PreviewOptions struct {
	ioes.IOStreams

	Refs   *RefSelect
	Format string
	Source string

	inst *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *PreviewOptions) Complete(f Factory, args []string) (err error) {
	if o.inst, err = f.Instance(); err != nil {
		return err
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, 1); err != nil {
		return err
	}
	return nil
}

// Run executes the publish command
func (o *PreviewOptions) Run() error {
	p := lib.PreviewParams{
		Ref: o.Refs.Ref(),
	}

	ctx := context.TODO()
	res, err := o.inst.WithSource(o.Source).Remote().Preview(ctx, &p)
	if err != nil {
		return err
	}
	var preview string

	switch o.Format {
	case "pretty":
		printToPager(o.Out, datasetPreview(res))
		return nil
	case "json":
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return err
		}
		preview = string(data)
	}

	printInfo(o.Out, preview)
	return nil
}

func datasetPreview(ds *dataset.Dataset) *bytes.Buffer {
	b := &bytes.Buffer{}

	b.WriteString(fmt.Sprintf("%s/%s@%s\n", ds.Peername, ds.Name, ds.Path))
	b.WriteString(ds.Meta.Title + "\n")
	b.WriteString(humanize.RelTime(ds.Commit.Timestamp, time.Now(), "ago", "from now") + "\n")
	b.WriteString(fmt.Sprintf("Details: %d commits | %d rows | %s | %s format\n", ds.NumVersions, ds.Structure.Entries, humanize.Bytes(uint64(ds.Structure.Length)), ds.Structure.Format))
	b.WriteString(fmt.Sprintf("Keywords: %s\n", strings.Join(ds.Meta.Keywords, ", ")))
	if ds.Meta.Description != "" {
		b.WriteString("Description:\n")
		b.WriteString(fmt.Sprintf("  %s", strings.ReplaceAll(ds.Meta.Description, "\n", "\n  ")))
	}
	b.WriteString("\n")

	if ds.Readme.Text != "" {
		b.WriteString("Readme:\n")
		b.WriteString(fmt.Sprintf("  %s", strings.ReplaceAll(ds.Readme.Text, "\n", "\n  ")))
	}

	return b
}
