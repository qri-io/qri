package config

import (
	"testing"
)

func TestCLIValidate(t *testing.T) {
	cliDefault := CLI{
		ColorizeOutput: true,
	}
	err := cliDefault.Validate()
	if err != nil {
		t.Errorf("error validating cliDefault: %s", err)
	}
}
