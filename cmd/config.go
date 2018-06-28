package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewConfigCommand creates a new `qri config` cobra command
// config represents commands that read & modify configuration settings
func NewConfigCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := ConfigOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "get and set local configuration information",
		Annotations: map[string]string{
			"group": "other",
		},
		Long: `
config encapsulates all settings that control the behaviour of qri.
This includes all kinds of stuff: your profile details; enabling & disabling 
different services; what kind of output qri logs to; 
which ports on qri serves on; etc.

Configuration is stored as a .yaml file kept at $QRI_PATH, or provided at CLI 
runtime via command a line argument.`,
		Example: `  # get your profile information
  $ qri config get profile

  # set your api port to 4444
  $ qri config set api.port 4444

  # disable rpc connections:
  $ qri config set rpc.enabled false`,
	}

	get := &cobra.Command{
		Use:   "get",
		Short: "get configuration settings",
		Long: `get outputs your current configuration file with private keys 
removed by default, making it easier to share your qri configuration settings.

The --with-private-keys option will show private keys.
PLEASE PLEASE PLEASE NEVER SHARE YOUR PRIVATE KEYS WITH ANYONE. EVER.
Anyone with your private keys can impersonate you on qri.`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.ErrOut, o.Complete(f))
			ExitIfErr(o.ErrOut, o.Get(args))
		},
	}

	set := &cobra.Command{
		Use:   "set",
		Short: "Set a configuration option",
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.ErrOut, o.Complete(f))
			ExitIfErr(o.ErrOut, o.Set(args))
		},
	}

	get.Flags().BoolVar(&o.WithPrivateKeys, "with-private-keys", false, "include private keys in export")
	get.Flags().BoolVarP(&o.Concise, "concise", "c", false, "print output without indentation, only applies to json format")
	get.Flags().StringVarP(&o.Format, "format", "f", "yaml", "data format to export. either json or yaml")
	get.Flags().StringVarP(&o.Output, "output", "o", "", "path to export to")
	cmd.AddCommand(get)
	cmd.AddCommand(set)

	return cmd
}

// ConfigOptions encapsulates state for the config command
type ConfigOptions struct {
	IOStreams

	Format          string
	WithPrivateKeys bool
	Concise         bool
	Output          string

	ProfileRequests *lib.ProfileRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ConfigOptions) Complete(f Factory) (err error) {
	o.ProfileRequests, err = f.ProfileRequests()
	return
}

// Get a configuration option
func (o *ConfigOptions) Get(args []string) (err error) {
	params := &lib.GetConfigParams{
		WithPrivateKey: o.WithPrivateKeys,
		Format:         o.Format,
		Concise:        o.Concise,
	}

	if len(args) == 1 {
		params.Field = args[0]
	}

	var data []byte

	if err = lib.GetConfig(params, &data); err != nil {
		return err
	}

	if o.Output != "" {
		if err = ioutil.WriteFile(o.Output, data, os.ModePerm); err != nil {
			return err
		}
		printSuccess(o.Out, "config file written to: %s", o.Output)
		return
	}

	fmt.Fprintln(o.Out, string(data))
	return
}

// Set a configuration option
func (o *ConfigOptions) Set(args []string) (err error) {
	if len(args)%2 != 0 {
		return fmt.Errorf("wrong number of arguments. arguments must be in the form: [path value]")
	}
	ip := config.ImmutablePaths()
	photoPaths := map[string]bool{
		"profile.photo":  true,
		"profile.poster": true,
		"profile.thumb":  true,
	}

	for i := 0; i < len(args)-1; i = i + 2 {
		var value interface{}
		path := strings.ToLower(args[i])
		if ip[path] {
			ErrExit(o.ErrOut, fmt.Errorf("cannot set path %s", path))
		}

		if photoPaths[path] {
			if err = setPhotoPath(o.ProfileRequests, path, args[i+1]); err != nil {
				return err
			}
		} else {
			if err = yaml.Unmarshal([]byte(args[i+1]), &value); err != nil {
				return err
			}

			if err = lib.Config.Set(path, value); err != nil {
				return err
			}
		}
	}
	if err = lib.SetConfig(lib.Config); err != nil {
		return err
	}

	printSuccess(o.Out, "config updated")
	return nil
}

func setPhotoPath(req *lib.ProfileRequests, proppath, filepath string) error {
	f, err := loadFileIfPath(filepath)
	if err != nil {
		return err
	}

	p := &lib.FileParams{
		Filename: f.Name(),
		Data:     f,
	}
	res := &config.ProfilePod{}

	switch proppath {
	case "profile.photo", "profile.thumb":
		if err := req.SetProfilePhoto(p, res); err != nil {
			return err
		}
	case "profile.poster":
		if err := req.SetPosterPhoto(p, res); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized path to set photo: %s", proppath)
	}

	return nil
}
