package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/gorilla/mux"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/lib"
)

func TestParseGetParamsFromRequest(t *testing.T) {
	cases := []struct {
		description  string
		url          string
		expectParams *lib.GetParams
		muxVars      map[string]string
	}{
		{
			"basic get",
			"/get/peer/my_ds",
			&lib.GetParams{
				Ref: "peer/my_ds",
				All: true,
			},
			map[string]string{"username": "peer", "name": "my_ds"},
		},
		{
			"get request with ref",
			"/get/peer/my_ds",
			&lib.GetParams{
				Ref: "peer/my_ds",
				All: true,
			},
			map[string]string{"ref": "peer/my_ds"},
		},
		{
			"meta component",
			"/get/peer/my_ds/meta",
			&lib.GetParams{
				Ref:      "peer/my_ds",
				Selector: "meta",
				All:      true,
			},
			map[string]string{"username": "peer", "name": "my_ds", "selector": "meta"},
		},
		{
			"get request with limit and offset",
			"/get/peer/my_ds/body",
			&lib.GetParams{
				Ref:      "peer/my_ds",
				Selector: "body",
				Limit:    0,
				Offset:   10,
			},
			map[string]string{"ref": "peer/my_ds", "selector": "body", "limit": "0", "offset": "10"},
		},
		{
			"get request with 'all'",
			"/get/peer/my_ds/body",
			&lib.GetParams{
				Ref:      "peer/my_ds",
				Selector: "body",
				All:      true,
			},
			map[string]string{"ref": "peer/my_ds", "selector": "body", "all": "true"},
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			r, _ := http.NewRequest("GET", c.url, nil)
			if c.muxVars != nil {
				r = mux.SetURLVars(r, c.muxVars)
			}
			r = mustSetMuxVarsOnRequest(t, r, c.muxVars)
			gotParams := &lib.GetParams{}
			if err := parseGetParamsFromRequest(r, gotParams); err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(c.expectParams, gotParams); diff != "" {
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
			map[string]string{"ref": "me/my_ds"},
		},
		{
			"bad parse",
			"/get/peer/my+ds",
			`unexpected character at position 7: '+'`,
			map[string]string{"ref": "peer/my+ds"},
		},
	}
	for i, c := range badCases {
		t.Run(c.description, func(t *testing.T) {
			r := httptest.NewRequest("GET", c.url, nil)
			r = mustSetMuxVarsOnRequest(t, r, c.muxVars)
			gotParams := &lib.GetParams{}
			err := parseGetParamsFromRequest(r, gotParams)
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

func TestParseSaveParamsFromRequest(t *testing.T) {
	expect := &lib.SaveParams{
		Ref:                 "test/ref",
		Title:               "test title",
		Message:             "test message",
		Apply:               true,
		Replace:             true,
		ConvertFormatToPrev: true,
		Drop:                "drop",
		Force:               true,
		NewName:             true,
		Dataset: &dataset.Dataset{
			Name:     "dataset name",
			Peername: "test peername",
			Meta: &dataset.Meta{
				Title: "test meta title",
				Qri:   "md:0",
			},
		},
	}

	dsBytes, err := json.Marshal(expect.Dataset)
	if err != nil {
		t.Fatalf("error marshaling dataset: %s", err)
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("ref", expect.Ref)
	writer.WriteField("title", expect.Title)
	writer.WriteField("message", expect.Message)
	writer.WriteField("apply", "true")
	writer.WriteField("replace", "true")
	writer.WriteField("convertFormatToPrev", "true")
	writer.WriteField("drop", "drop")
	writer.WriteField("force", "true")
	writer.WriteField("newName", "true")
	writer.WriteField("dataset", string(dsBytes))
	writer.Close()

	r, err := http.NewRequest(http.MethodPost, "save", body)
	if err != nil {
		t.Fatalf("error creating new request: %s", err)
	}
	r.Header.Add("Content-Type", writer.FormDataContentType())
	got := &lib.SaveParams{}
	err = parseSaveParamsFromRequest(r, got)
	if err != nil {
		t.Fatalf("error saving params from request: %s", err)
	}
	if diff := cmp.Diff(expect, got, cmpopts.IgnoreUnexported(dataset.Dataset{}), cmpopts.IgnoreUnexported(dataset.Meta{})); diff != "" {
		t.Errorf("SaveParams mismatch (-want +got):%s\n", diff)
	}
}

func TestParseDatasetFromRequest(t *testing.T) {
	r := newFormFileRequest(t, "/", nil, nil)
	dsp := &dataset.Dataset{}
	if err := parseDatasetFromRequest(r, dsp); err != nil {
		t.Error("expected 'empty' request to be ok")
	}

	r = newFormFileRequest(t, "/", map[string]string{
		"file":      dstestTestdataFile("cities/init_dataset.json"),
		"viz":       dstestTestdataFile("cities/template.html"),
		"transform": dstestTestdataFile("cities/transform.star"),
		"readme":    dstestTestdataFile("cities/readme.md"),
		"body":      dstestTestdataFile("cities/data.csv"),
	}, nil)
	if err := parseDatasetFromRequest(r, dsp); err != nil {
		t.Error(err)
	}

	r = newFormFileRequest(t, "/", map[string]string{
		"file": "testdata/cities/dataset.yml",
		"body": dstestTestdataFile("cities/data.csv"),
	}, nil)
	if err := parseDatasetFromRequest(r, dsp); err != nil {
		t.Error(err)
	}
}
