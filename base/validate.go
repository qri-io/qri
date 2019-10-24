package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/repo"
)

// Validate checks a dataset body for errors based on a schema
func Validate(ctx context.Context, r repo.Repo, ref repo.DatasetRef, body, schema qfs.File) (errors []jsonschema.ValError, err error) {
	if !ref.IsEmpty() {
		err = repo.CanonicalizeDatasetRef(r, &ref)
		if err != nil && err != repo.ErrNotFound {
			log.Debug(err.Error())
			err = fmt.Errorf("error with new reference: %s", err.Error())
			return
		}
	}

	var (
		st   = &dataset.Structure{}
		data []byte
	)

	// if a dataset is specified, load it
	if ref.Path != "" {
		if err = ReadDataset(ctx, r, &ref); err != nil {
			log.Debug(err.Error())
			return
		}

		ds := ref.Dataset
		st = ds.Structure
	} else if body == nil {
		err = fmt.Errorf("cannot find dataset: %s", ref)
		return
	}

	if body != nil {
		data, err = ioutil.ReadAll(body)
		if err != nil {
			log.Debug(err.Error())
			err = fmt.Errorf("error reading data: %s", err.Error())
			return
		}

		// if no schema, detect one
		if st.Schema == nil {
			var df dataset.DataFormat
			df, err = detect.ExtensionDataFormat(body.FileName())
			if err != nil {
				err = fmt.Errorf("detecting data format: %s", err.Error())
				return
			}
			str, _, e := detect.FromReader(df, bytes.NewBuffer(data))
			if e != nil {
				err = fmt.Errorf("error detecting from reader: %s", e)
				return
			}
			st = str
		}
	}

	// if a schema is specified, override with it
	if schema != nil {
		stbytes, e := ioutil.ReadAll(schema)
		if e != nil {
			log.Debug(e.Error())
			err = e
			return
		}
		sch := map[string]interface{}{}
		if e := json.Unmarshal(stbytes, &sch); e != nil {
			err = fmt.Errorf("error reading schema: %s", e.Error())
			return
		}
		st.Schema = sch
	}

	if data == nil && ref.Dataset != nil {
		ds := ref.Dataset

		f, e := dsfs.LoadBody(ctx, r.Store(), ds)
		if e != nil {
			log.Debug(e.Error())
			err = fmt.Errorf("error loading dataset data: %s", e.Error())
			return
		}
		data, err = ioutil.ReadAll(f)
		if err != nil {
			log.Debug(err.Error())
			err = fmt.Errorf("error loading dataset data: %s", err.Error())
			return
		}
	}

	er, err := dsio.NewEntryReader(st, bytes.NewBuffer(data))
	if err != nil {
		log.Debug(err.Error())
		err = fmt.Errorf("error reading data: %s", err.Error())
		return
	}

	return validate.EntryReader(er)
}
