package base

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/qri-io/dataset"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/repo"
)

// Validate checks a dataset body for errors based on the structure's schema
func Validate(ctx context.Context, r repo.Repo, body qfs.File, st *dataset.Structure) ([]jsonschema.KeyError, error) {
	if body == nil {
		return nil, fmt.Errorf("body passed to Validate must not be nil")
	}
	if st == nil {
		return nil, fmt.Errorf("structure passed to Validate must not be nil")
	}
	if st.Schema == nil {
		return nil, fmt.Errorf("structure.Schema passed to Validate must not be nil")
	}

	// jsonschema assumes body is json, convert the format if necessary
	if st.Format != "json" {
		convert := dataset.Structure{
			Format: "json",
			Schema: st.Schema,
		}
		file, err := ConvertBodyFormat(body, st, &convert)
		if err != nil {
			log.Debugf("base.Validate: ConvertBodyFormat error: %s", err)
			return nil, err
		}
		body = file
	}

	// jsonschema does not handle data streams, have to read the whole body
	data, err := ioutil.ReadAll(body)
	if err != nil {
		log.Debugf("base.Validate: ioutil.ReadAll error: %s", err)
		return nil, err
	}
	jsch, err := st.JSONSchema()
	if err != nil {
		log.Debugf("base.Validate: JSONSchema error: %s", err)
		return nil, err
	}
	return jsch.ValidateBytes(ctx, data)
}
