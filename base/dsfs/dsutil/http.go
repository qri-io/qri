package dsutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
)

// FormFileDataset extracts a dataset document from a http Request
func FormFileDataset(r *http.Request, ds *dataset.Dataset) (err error) {
	datafile, dataHeader, err := r.FormFile("file")
	if err == http.ErrMissingFile {
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
				err = fmt.Errorf("error reading dataset file: %s", err)
				return
			}
			if err = UnmarshalYAMLDataset(data, ds); err != nil {
				err = fmt.Errorf("error unmarshaling yaml file: %s", err)
				return
			}
		case ".json":
			if err = json.NewDecoder(datafile).Decode(ds); err != nil {
				err = fmt.Errorf("error decoding json file: %s", err)
				return
			}
		}
	}

	if peername := r.FormValue("peername"); peername != "" {
		ds.Peername = peername
	}
	if name := r.FormValue("name"); name != "" {
		ds.Name = name
	}
	if bp := r.FormValue("body_path"); bp != "" {
		ds.BodyPath = bp
	}

	tfFile, tfHeader, err := r.FormFile("transform")
	if err == http.ErrMissingFile {
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening transform file: %s", err)
		return
	}
	if tfFile != nil {
		var tfData []byte
		if tfData, err = ioutil.ReadAll(tfFile); err != nil {
			return
		}
		if ds.Transform == nil {
			ds.Transform = &dataset.Transform{}
		}
		ds.Transform.Syntax = "starlark"
		ds.Transform.ScriptBytes = tfData
		ds.Transform.ScriptPath = tfHeader.Filename
	}

	vizFile, vizHeader, err := r.FormFile("viz")
	if err == http.ErrMissingFile {
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening viz file: %s", err)
		return
	}
	if vizFile != nil {
		var vizData []byte
		if vizData, err = ioutil.ReadAll(vizFile); err != nil {
			return
		}
		if ds.Viz == nil {
			ds.Viz = &dataset.Viz{}
		}
		ds.Viz.Format = "html"
		ds.Viz.ScriptBytes = vizData
		ds.Viz.ScriptPath = vizHeader.Filename
	}

	bodyfile, bodyHeader, err := r.FormFile("body")
	if err == http.ErrMissingFile {
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening body file: %s", err)
		return
	}
	if bodyfile != nil {
		var bodyData []byte
		if bodyData, err = ioutil.ReadAll(bodyfile); err != nil {
			return
		}
		ds.BodyPath = bodyHeader.Filename
		ds.BodyBytes = bodyData

		if ds.Structure == nil {
			// TODO - this is silly and should move into base.PrepareDataset funcs
			ds.Structure = &dataset.Structure{}
			format, err := detect.ExtensionDataFormat(bodyHeader.Filename)
			if err != nil {
				return err
			}
			st, _, err := detect.FromReader(format, bytes.NewReader(ds.BodyBytes))
			if err != nil {
				return err
			}
			ds.Structure = st
		}
	}

	return
}
