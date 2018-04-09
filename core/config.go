package core

import (
	"fmt"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/config"
)

var (
	// Config is the global configuration object
	Config *config.Config
	// ConfigFilepath is the default location for a config file
	ConfigFilepath string
)

// LoadConfig loads the global default configuration
func LoadConfig(path string) (err error) {
	var cfg *config.Config
	cfg, err = config.ReadFromFile(path)

	if err == nil && cfg.Profile == nil {
		err = fmt.Errorf("missing profile")
	}

	if err != nil {
		str := `couldn't read config file. error
  %s
if you've recently updated qri your config file may no longer be valid.
The easiest way to fix this is to delete your repository at:
  %s
and start with a fresh qri install by running 'qri setup' again.
Sorry, we know this is not exactly a great experience, from this point forward
we won't be shipping changes that require starting over.
`
		err = fmt.Errorf(str, err.Error(), path)
	}

	// configure logging straight away
	if cfg != nil && cfg.Logging != nil {
		for name, level := range cfg.Logging.Levels {
			golog.SetLogLevel(name, level)
		}
	}

	Config = cfg

	return err
}
