package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

const profilePrefix = "profile."

// NewConfigCommand creates a new `qri config` cobra command
// config represents commands that read & modify configuration settings
func NewConfigCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := ConfigOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set local configuration information",
		Annotations: map[string]string{
			"group": "other",
		},
		Long: `
'qri config' encapsulates all settings that control the behaviour of qri.
This includes all kinds of stuff: your profile details; enabling & disabling 
different services; what kind of output qri logs to; 
which ports on qri serves on; etc.

Configuration is stored as a .yaml file kept at $QRI_PATH, or provided at CLI 
runtime via command a line argument.

For details on each config field checkout: 
https://github.com/qri-io/qri/blob/master/config/readme.md`,
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

You can get particular parts of the config by using dot notation to
traverse the config object. For details on each config field checkout: 
https://github.com/qri-io/qri/blob/master/config/readme.md

The --with-private-keys option will show private keys.
PLEASE PLEASE PLEASE NEVER SHARE YOUR PRIVATE KEYS WITH ANYONE. EVER.
Anyone with your private keys can impersonate you on qri.`,
		Example: `  # get the entire config
  qri config get

  # get the config profile
  qri config get profile

  # get the profile description
  qri config get profile.description`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f); err != nil {
				return err
			}
			return o.Get(args)
		},
	}

	set := &cobra.Command{
		Use:   "set",
		Short: "Set configuration options",
		Long: `'qri config set' allows you to set configuration options. You can set 
particular parts of the config by using dot notation to traverse the 
config object. 

While the 'qri config get' command allows you to view the whole config,
or only parts of it, the 'qri config set' command is more specific.

If the config object were a tree and each field a branch, you can only
set the leaves of the branches. In other words, the you cannot set a 
field that is itself an object or array. For details on each config 
field checkout: https://github.com/qri-io/qri/blob/master/config/readme.md`,
		Example: `  # set a profile description
  qri config set profile.description "This is my new description that I
  am very proud of and want displayed in my profile"

  # disable rpc communication
  qri config set rpc.enabled false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			if err := o.Complete(f); err != nil {
				return err
			}
			return o.Set(args)
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
	ioes.IOStreams

	Format          string
	WithPrivateKeys bool
	Concise         bool
	Output          string

	inst           lib.Instance
	ConfigMethods  lib.ConfigMethods
	ProfileMethods lib.ProfileMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ConfigOptions) Complete(f Factory) (err error) {
	o.inst = f.Instance()
	o.ConfigMethods, err = f.ConfigMethods()
	if err != nil {
		return
	}

	o.ProfileMethods, err = f.ProfileMethods()
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

	if err = o.ConfigMethods.GetConfig(params, &data); err != nil {
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

	profile := o.inst.Config().Profile
	profileChanged := false

	for i := 0; i < len(args)-1; i = i + 2 {
		path := strings.ToLower(args[i])
		value := args[i+1]

		if ip[path] {
			ErrExit(o.ErrOut, fmt.Errorf("cannot set path %s", path))
		}

		if photoPaths[path] {
			if err = setPhotoPath(o.ProfileMethods, path, args[i+1]); err != nil {
				return err
			}
		} else if strings.HasPrefix(path, profilePrefix) {
			field := strings.ToLower(path[len(profilePrefix):])
			if err = profile.SetField(field, args[i+1]); err != nil {
				return err
			}
			profileChanged = true
		} else {
			// TODO (b5): I think this'll resule in configuration not getting set. should investigate
			if err = o.inst.Config().Set(path, value); err != nil {
				return err
			}
		}
	}
	var ok bool
	if err = o.ConfigMethods.SetConfig(o.inst.Config(), &ok); err != nil {
		return err
	}
	if profileChanged {
		var res config.ProfilePod
		if err = o.ProfileMethods.SaveProfile(profile, &res); err != nil {
			return err
		}
	}

	printSuccess(o.Out, "config updated")
	return nil
}

func setPhotoPath(m lib.ProfileMethods, proppath, filepath string) error {
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
		if err := m.SetProfilePhoto(p, res); err != nil {
			return err
		}
	case "profile.poster":
		if err := m.SetPosterPhoto(p, res); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized path to set photo: %s", proppath)
	}

	return nil
}
