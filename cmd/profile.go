package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

var (
	getProfileFilepath string
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

		data, err := editableProfileJSONBytes(res)
		ExitIfErr(err)

		if getProfileFilepath != "" {
			if !strings.HasPrefix(getProfileFilepath, ".json") {
				getProfileFilepath = getProfileFilepath + ".json"
			}
			err := ioutil.WriteFile(getProfileFilepath, data, os.ModePerm)
			ExitIfErr(err)
			printSuccess("file saved to: %s", getProfileFilepath)
		} else {
			printSuccess(string(data))
		}
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

		if setProfileFilepath == "" {
			printErr(fmt.Errorf("please provide a JSON file of profile info with the --file flag"))
			return
		}

		r, err := profileRequests(false)
		ExitIfErr(err)

		in := true
		p := &core.Profile{}
		err = r.GetProfile(&in, p)
		ExitIfErr(err)

		dataFile, err = loadFileIfPath(setProfileFilepath)
		ExitIfErr(err)

		set := &core.Profile{}
		err = json.NewDecoder(dataFile).Decode(set)
		ExitIfErr(err)

		if set.Name != "" {
			p.Name = set.Name
		}
		if set.Description != "" {
			p.Description = set.Description
		}
		if set.HomeURL != "" {
			p.HomeURL = set.HomeURL
		}
		if set.Twitter != "" {
			p.Twitter = set.Twitter
		}
		if set.Email != "" {
			p.Email = set.Email
		}
		if set.Color != "" {
			p.Color = set.Color
		}

		res := &core.Profile{}
		err = r.SaveProfile(p, res)
		ExitIfErr(err)

		data, err := editableProfileJSONBytes(res)
		ExitIfErr(err)
		printSuccess(string(data))
	},
}

func init() {
	profileGetCmd.Flags().StringVarP(&getProfileFilepath, "file", "f", "", "json file to save profile info to")
	profileCmd.AddCommand(profileGetCmd)

	profileSetCmd.Flags().StringVarP(&setProfileFilepath, "file", "f", "", "json file to update profile info")
	profileCmd.AddCommand(profileSetCmd)

	RootCmd.AddCommand(profileCmd)
}

func editableProfileJSONBytes(res *core.Profile) ([]byte, error) {
	mapr := map[string]interface{}{
		"name":        res.Name,
		"description": res.Description,
		"homeUrl":     res.HomeURL,
		"twitter":     res.Twitter,
		"email":       res.Email,
		"color":       res.Color,
	}

	return json.MarshalIndent(mapr, "", "  ")
}
