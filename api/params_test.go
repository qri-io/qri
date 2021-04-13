package api

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"github.com/qri-io/qri/lib"
)

func TestUnmarshalGetParams(t *testing.T) {
	cases := []struct {
		description string
		url         string
		expectArgs  *lib.GetParams
		muxVars     map[string]string
	}{
		{
			"basic get",
			"/get/peer/my_ds",
			&lib.GetParams{
				Ref:    "peer/my_ds",
				Format: "json",
				All:    true,
			},
			map[string]string{"username": "peer", "name": "my_ds"},
		},
		{
			"meta component",
			"/get/peer/my_ds/meta",
			&lib.GetParams{
				Ref:      "peer/my_ds",
				Format:   "json",
				Selector: "meta",
				All:      true,
			},
			map[string]string{"username": "peer", "name": "my_ds", "selector": "meta"},
		},
		{
			"body component",
			"/get/peer/my_ds/body",
			&lib.GetParams{
				Ref:      "peer/my_ds",
				Format:   "json",
				Selector: "body",
				All:      true,
			},
			map[string]string{"username": "peer", "name": "my_ds", "selector": "body"},
		},
		{
			"body.csv path suffix",
			"/get/peer/my_ds/body.csv",
			&lib.GetParams{
				Ref:      "peer/my_ds",
				Format:   "csv",
				Selector: "body",
				All:      true,
			},
			map[string]string{"username": "peer", "name": "my_ds", "selector": "body.csv"},
		},
		{
			"download body as csv",
			"/get/peer/my_ds/body?format=csv",
			&lib.GetParams{
				Ref:      "peer/my_ds",
				Format:   "csv",
				Selector: "body",
				All:      true,
			},
			map[string]string{"username": "peer", "name": "my_ds", "selector": "body"},
		},
		{
			"zip format",
			"/get/peer/my_ds?format=zip",
			&lib.GetParams{
				Ref:    "peer/my_ds",
				Format: "zip",
				All:    true,
			},
			map[string]string{"username": "peer", "name": "my_ds"},
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			r, _ := http.NewRequest("GET", c.url, nil)
			if c.muxVars != nil {
				r = mux.SetURLVars(r, c.muxVars)
			}
			setRefStringFromMuxVars(r)
			args := &lib.GetParams{}
			err := UnmarshalParams(r, args)
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(c.expectArgs, args); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}

	badCases := []struct {
		description string
		url         string
		expectErr   string
		muxVars     map[string]string
	}{
		{
			"get me",
			"/get/me/my_ds",
			`username "me" not allowed`,
			map[string]string{"username": "me", "name": "my_ds"},
		},
		{
			"bad parse",
			"/get/peer/my+ds",
			`unexpected character at position 7: '+'`,
			map[string]string{"username": "peer", "name": "my+ds"},
		},
	}
	for i, c := range badCases {
		t.Run(c.description, func(t *testing.T) {
			r, _ := http.NewRequest("GET", c.url, nil)
			if c.muxVars != nil {
				r = mux.SetURLVars(r, c.muxVars)
			}
			setRefStringFromMuxVars(r)
			args := &lib.GetParams{}
			err := UnmarshalParams(r, args)
			if err == nil {
				t.Errorf("case %d: expected error, but did not get one", i)
				return
			}
			if diff := cmp.Diff(c.expectErr, err.Error()); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseGetParamsAcceptHeader(t *testing.T) {
	// Construct a request with "Accept: text/csv"
	r, _ := http.NewRequest("GET", "/get/peer/my_ds", nil)
	r.Header.Add("Accept", "text/csv")
	r = mux.SetURLVars(r, map[string]string{"username": "peer", "name": "my_ds"})
	setRefStringFromMuxVars(r)
	args := &lib.GetParams{}
	err := UnmarshalParams(r, args)
	if err != nil {
		t.Fatal(err)
	}
	expectArgs := &lib.GetParams{
		Ref:      "peer/my_ds",
		Selector: "body",
		Format:   "csv",
		All:      true,
	}

	if diff := cmp.Diff(expectArgs, args); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Construct a request with format=csv and "Accept: text/csv", which is ok
	r, _ = http.NewRequest("GET", "/get/peer/my_ds?format=csv", nil)
	r.Header.Add("Accept", "text/csv")
	r = mux.SetURLVars(r, map[string]string{"username": "peer", "name": "my_ds"})
	setRefStringFromMuxVars(r)
	args = &lib.GetParams{}
	err = UnmarshalParams(r, args)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectArgs, args); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Construct a request with format=json and "Accept: text/csv", which is an error
	r, _ = http.NewRequest("GET", "/get/peer/my_ds?format=json", nil)
	r.Header.Add("Accept", "text/csv")
	r = mux.SetURLVars(r, map[string]string{"username": "peer", "name": "my_ds"})
	setRefStringFromMuxVars(r)
	args = &lib.GetParams{}
	err = UnmarshalParams(r, args)
	if err == nil {
		t.Error("expected to get an error, but did not get one")
	}
	expectErr := `format "json" conflicts with header "Accept: text/csv"`
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect: %q, got %q", expectErr, err)
	}
}
