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
				Refstr: "peer/my_ds",
				Format: "json",
				All:    true,
			},
			map[string]string{"peername": "peer", "name": "my_ds"},
		},
		{
			"meta component",
			"/get/peer/my_ds/meta",
			&lib.GetParams{
				Refstr:   "peer/my_ds",
				Format:   "json",
				Selector: "meta",
				All:      true,
			},
			map[string]string{"peername": "peer", "name": "my_ds", "selector": "meta"},
		},
		{
			"body component",
			"/get/peer/my_ds/body",
			&lib.GetParams{
				Refstr:   "peer/my_ds",
				Format:   "json",
				Selector: "body",
				All:      true,
			},
			map[string]string{"peername": "peer", "name": "my_ds", "selector": "body"},
		},
		{
			"body.csv path suffix",
			"/get/peer/my_ds/body.csv",
			&lib.GetParams{
				Refstr:   "peer/my_ds",
				Format:   "csv",
				Selector: "body",
				All:      true,
			},
			map[string]string{"peername": "peer", "name": "my_ds", "selector": "body.csv"},
		},
		{
			"download body as csv",
			"/get/peer/my_ds/body?format=csv",
			&lib.GetParams{
				Refstr:   "peer/my_ds",
				Format:   "csv",
				Selector: "body",
				All:      true,
			},
			map[string]string{"peername": "peer", "name": "my_ds", "selector": "body"},
		},
		{
			"zip format",
			"/get/peer/my_ds?format=zip",
			&lib.GetParams{
				Refstr: "peer/my_ds",
				Format: "zip",
				All:    true,
			},
			map[string]string{"peername": "peer", "name": "my_ds"},
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
			map[string]string{"peername": "me", "name": "my_ds"},
		},
		{
			"bad parse",
			"/get/peer/my+ds",
			`unexpected character at position 7: '+'`,
			map[string]string{"peername": "peer", "name": "my+ds"},
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
