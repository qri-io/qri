package config

import (
	"reflect"
	"testing"
)

func TestAPIValidate(t *testing.T) {
	err := DefaultAPI().Validate()
	if err != nil {
		t.Errorf("error validating default api: %s", err)
	}
}

func TestAPICopy(t *testing.T) {
	a := DefaultAPI()
	b := a.Copy()

	a.Enabled = !a.Enabled
	a.Address = "foo"
	a.Webui = !a.Webui
	a.ServeRemoteTraffic = !a.ServeRemoteTraffic
	a.AllowedOrigins = []string{"bar"}

	if a.Enabled == b.Enabled {
		t.Errorf("Enabled fields should not match")
	}
	if a.Address == b.Address {
		t.Errorf("Address fields should not match")
	}
	if a.Webui == b.Webui {
		t.Errorf("Webui fields should not match")
	}
	if a.ServeRemoteTraffic == b.ServeRemoteTraffic {
		t.Errorf("ServeRemoteTraffic fields should not match")
	}
	if reflect.DeepEqual(a.AllowedOrigins, b.AllowedOrigins) {
		t.Errorf("AllowedOrigins fields should not match")
	}
}
