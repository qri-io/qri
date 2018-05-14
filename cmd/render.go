package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

// renderCmd represents the render command
var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "execute a template against a dataset",
	Long:  `the most common use for render is to generate html from a qri dataset`,
	Example: `  render a dataset called me/schools:
  $ qri render -o=schools.html me/schools

  render a dataset with a custom template:
  $ qri render --template=template.html me/schools`,
	Annotations: map[string]string{
		"group": "dataset",
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var template []byte

		fp, err := cmd.Flags().GetString("template")
		ExitIfErr(err)
		out, err := cmd.Flags().GetString("output")
		ExitIfErr(err)
		limit, err := cmd.Flags().GetInt("limit")
		ExitIfErr(err)
		offset, err := cmd.Flags().GetInt("offset")
		ExitIfErr(err)
		all, err := cmd.Flags().GetBool("all")
		ExitIfErr(err)

		req, err := renderRequests(false)
		ExitIfErr(err)

		if fp != "" {
			template, err = ioutil.ReadFile(fp)
			ExitIfErr(err)
		}

		p := &core.RenderParams{
			Ref:            args[0],
			Template:       template,
			TemplateFormat: "html",
			All:            all,
			Limit:          limit,
			Offset:         offset,
		}

		res := []byte{}
		err = req.Render(p, &res)
		ExitIfErr(err)

		if out == "" {
			fmt.Print(string(res))
		} else {
			ioutil.WriteFile(out, res, 0777)
		}
	},
}

func init() {
	renderCmd.Flags().StringP("template", "t", "", "path to template file")
	renderCmd.Flags().StringP("output", "o", "", "path to write output file")
	renderCmd.Flags().BoolP("all", "a", false, "read all dataset entries (overrides limit, offest)")
	renderCmd.Flags().IntP("limit", "l", 50, "max number of records to read")
	renderCmd.Flags().IntP("offset", "s", 0, "number of records to skip")

	RootCmd.AddCommand(renderCmd)
}
