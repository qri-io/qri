package api

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"github.com/qri-io/qri/lib"
)

func TestUnmarshal(t *testing.T) {
	cases := []struct {
		description  string
		url          string
		params       RequestUnmarshaller
		expectParams RequestUnmarshaller
		muxVars      map[string]string
	}{
		{
			"basic get",
			"/get/peer/my_ds",
			&lib.GetParams{},
			&lib.GetParams{
				Ref: "peer/my_ds",
				All: true,
			},
			map[string]string{"username": "peer", "name": "my_ds"},
		},
		{
			"meta component",
			"/get/peer/my_ds/meta",
			&lib.GetParams{},
			&lib.GetParams{
				Ref:      "peer/my_ds",
				Selector: "meta",
				All:      true,
			},
			map[string]string{"username": "peer", "name": "my_ds", "selector": "meta"},
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			r, _ := http.NewRequest("GET", c.url, nil)
			if c.muxVars != nil {
				r = mux.SetURLVars(r, c.muxVars)
			}
			r = mustSetMuxVarsOnRequest(t, r, c.muxVars)
			err := UnmarshalParams(r, c.params)
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(c.expectParams, c.params); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
