package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/base/fill"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"gopkg.in/yaml.v2"
)

// parseGetParamsFromRequest parse the form from the request to create `lib.GetParams`
func parseGetParamsFromRequest(r *http.Request, p *lib.GetParams) error {
	log.Debugf("parseGetParams ref:%s", r.FormValue("ref"))
	p.Ref = r.FormValue("ref")

	ref, err := dsref.Parse(p.Ref)
	if err != nil {
		return err
	}

	if ref.Username == "me" {
		return fmt.Errorf("username \"me\" not allowed")
	}

	p.Selector = r.FormValue("selector")

	p.All = util.ReqParamBool(r, "all", true)
	p.Limit = util.ReqParamInt(r, "limit", 0)
	p.Offset = util.ReqParamInt(r, "offset", 0)
	if !(p.Offset == 0 && p.Limit == 0) {
		p.All = false
	}

	if p.All && p.Limit == 0 && p.Offset == 0 {
		page := -1
		pageSize := -1
		if i := util.ReqParamInt(r, "page", 0); i != 0 {
			page = i
		}
		if i := util.ReqParamInt(r, "pageSize", 0); i != 0 {
			pageSize = i
		}
		// we don't want the defaults to override this and also only want to
		// set values if they are present
		if page >= 0 && pageSize >= 0 {
			p.Limit = pageSize
			p.Offset = (page - 1) * pageSize
			p.All = false
		}
	}
	return nil
}

// parseSaveParamsFromRequest parses the form from the request
// it ignores `bodyFile`, `filePaths`, and `secrets`
// `dataset`, if it exists, is expected to be a json string in
// the form of a `dataset.Dataset`
func parseSaveParamsFromRequest(r *http.Request, p *lib.SaveParams) error {
	p.Ref = r.FormValue("ref")
	p.Title = r.FormValue("title")
	p.Message = r.FormValue("message")
	p.Apply = util.ReqParamBool(r, "apply", false)
	p.Replace = util.ReqParamBool(r, "replace", false)
	p.Private = util.ReqParamBool(r, "private", false)
	p.ConvertFormatToPrev = util.ReqParamBool(r, "convertFormatToPrev", false)
	p.Drop = r.FormValue("drop")
	p.Force = util.ReqParamBool(r, "force", false)
	p.ShouldRender = util.ReqParamBool(r, "shouldRender", false)
	p.NewName = util.ReqParamBool(r, "newName", false)
	dsBytes := []byte(r.FormValue("dataset"))
	if len(dsBytes) != 0 {
		p.Dataset = &dataset.Dataset{}
		err := p.Dataset.UnmarshalJSON(dsBytes)
		if err != nil {
			return err
		}
	}
	log.Debugw("parseSaveParamsFromRequest", "params", p)
	return nil
}

// parseDatasetFromRequest extracts a dataset document from a http Request
func parseDatasetFromRequest(r *http.Request, ds *dataset.Dataset) (err error) {
	datafile, dataHeader, err := r.FormFile("file")
	if err == http.ErrMissingFile {
		// users should be allowed to only upload certain parts of the dataset
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening dataset file: %s", err)
		return
	}
	if datafile != nil {
		switch strings.ToLower(filepath.Ext(dataHeader.Filename)) {
		case ".yaml", ".yml":
			var data []byte
			data, err = ioutil.ReadAll(datafile)
			if err != nil {
				err = fmt.Errorf("reading dataset file: %w", err)
				return
			}
			fields := &map[string]interface{}{}
			if err = yaml.Unmarshal(data, fields); err != nil {
				err = fmt.Errorf("deserializing YAML file: %w", err)
				return
			}
			if err = fill.Struct(*fields, ds); err != nil {
				return
			}
		case ".json":
			if err = json.NewDecoder(datafile).Decode(ds); err != nil {
				err = fmt.Errorf("error decoding json file: %w", err)
				return
			}
		}
	}

	if username := r.FormValue("username"); username != "" {
		ds.Peername = username
	}
	if name := r.FormValue("name"); name != "" {
		ds.Name = name
	}
	if bp := r.FormValue("body_path"); bp != "" {
		ds.BodyPath = bp
	}

	tfFile, tfHeader, err := r.FormFile("transform")
	if err == http.ErrMissingFile {
		// users should be allowed to only upload certain parts of the dataset
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening transform file: %w", err)
		return
	}
	if tfFile != nil {
		if ds.Transform == nil {
			ds.Transform = &dataset.Transform{}
		}
		ds.Transform.SetScriptFile(qfs.NewMemfileReader(tfHeader.Filename, tfFile))
	}

	vizFile, vizHeader, err := r.FormFile("viz")
	if err == http.ErrMissingFile {
		// users should be allowed to only upload certain parts of the dataset
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening viz file: %w", err)
		return
	}
	if vizFile != nil {
		if ds.Viz == nil {
			ds.Viz = &dataset.Viz{}
		}
		ds.Viz.SetScriptFile(qfs.NewMemfileReader(vizHeader.Filename, vizFile))
	}

	readmeFile, readmeHeader, err := r.FormFile("readme")
	if err == http.ErrMissingFile {
		// users should be allowed to only upload certain parts of the dataset
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening readme file: %w", err)
		return
	}
	if readmeFile != nil {
		if ds.Readme == nil {
			ds.Readme = &dataset.Readme{}
			ds.Readme.SetScriptFile(qfs.NewMemfileReader(readmeHeader.Filename, readmeFile))
		}
	}

	bodyfile, bodyHeader, err := r.FormFile("body")
	if err == http.ErrMissingFile {
		// users should be allowed to only upload certain parts of the dataset
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening body file: %w", err)
		return
	}
	if bodyfile != nil {
		ds.SetBodyFile(qfs.NewMemfileReader(bodyHeader.Filename, bodyfile))
	}

	return
}
