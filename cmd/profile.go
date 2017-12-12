package cmd

import (
	"encoding/json"
	"os"

	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

var (
	setProfileFilepath string
)

// profileCmd represents the profile command
var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "show or edit user profile information",
}

var profileGetCmd = &cobra.Command{
	Use:   "get",
	Short: "get profile info",
	Run: func(cmd *cobra.Command, args []string) {
		r, err := ProfileRequests(false)
		ExitIfErr(err)

		in := true
		res := &core.Profile{}
		err = r.GetProfile(&in, res)
		ExitIfErr(err)

		data, err := json.MarshalIndent(res, "", "  ")
		ExitIfErr(err)
		PrintSuccess(string(data))
	},
}

var profileSetCmd = &cobra.Command{
	Use:   "set",
	Short: "add peers to the profile list",
	Run: func(cmd *cobra.Command, args []string) {
		var (
			dataFile *os.File
			err      error
		)

		r, err := ProfileRequests(false)
		ExitIfErr(err)

		dataFile, err = loadFileIfPath(setProfileFilepath)
		ExitIfErr(err)

		p := &core.Profile{}
		err = json.NewDecoder(dataFile).Decode(p)
		ExitIfErr(err)

		res := &core.Profile{}
		err = r.SaveProfile(p, res)
		ExitIfErr(err)

		data, err := json.MarshalIndent(res, "", "  ")
		ExitIfErr(err)
		PrintSuccess(string(data))
	},
}

func init() {
	profileSetCmd.Flags().StringVarP(&setProfileFilepath, "file", "f", "", "json file to update profile info")

	profileCmd.AddCommand(profileGetCmd)
	profileCmd.AddCommand(profileSetCmd)
	RootCmd.AddCommand(profileCmd)
}
