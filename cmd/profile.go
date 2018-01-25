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
	Short: "get and set user profile information",
	Long: `
profile is a bit of a cheat right now. We’re going to break profile out into 
proper commands later on, but for now we’re hoping you can edit a JSON file of 
profile information. 

For now running qri profile get will write a file called profile.json containing 
current profile info. Edit that file and run qri profile set <file> to write 
configuration details back.

Expect the profile command to change in future releases.`,
}

var profileGetCmd = &cobra.Command{
	Use:   "get",
	Short: "get profile info",
	Run: func(cmd *cobra.Command, args []string) {
		r, err := profileRequests(false)
		ExitIfErr(err)

		in := true
		res := &core.Profile{}
		err = r.GetProfile(&in, res)
		ExitIfErr(err)

		data, err := json.MarshalIndent(res, "", "  ")
		ExitIfErr(err)
		printSuccess(string(data))
	},
}

var profileSetCmd = &cobra.Command{
	Use:   "set",
	Short: "set profile details",
	Run: func(cmd *cobra.Command, args []string) {
		var (
			dataFile *os.File
			err      error
		)

		r, err := profileRequests(false)
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
		printSuccess(string(data))
	},
}

func init() {
	profileSetCmd.Flags().StringVarP(&setProfileFilepath, "file", "f", "", "json file to update profile info")

	profileCmd.AddCommand(profileGetCmd)
	profileCmd.AddCommand(profileSetCmd)
	RootCmd.AddCommand(profileCmd)
}
