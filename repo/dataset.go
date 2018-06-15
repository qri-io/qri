package repo

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
)

// DatasetPodBodyFile creates a streaming data file from a DatasetPod using the following precedence:
// * dsp.BodyBytes not being nil (requires dsp.Structure.Format be set to know data format)
// * dsp.BodyPath being a url
// * dsp.BodyPath being a path on the local filesystem
// This func is in the repo package b/c it has a destiny. And that destiny is to become a method on a
// forthcoming Dataset struct. see https://github.com/qri-io/qri/issues/414 for deets
// TODO - this feels out of place. maybe this belongs in core? actions?
func DatasetPodBodyFile(dsp *dataset.DatasetPod) (cafs.File, error) {
	if dsp.BodyBytes != nil {
		if dsp.Structure == nil || dsp.Structure.Format == "" {
			return nil, fmt.Errorf("specifying bodyBytes requires format be specified in dataset.structure")
		}
		return cafs.NewMemfileBytes(fmt.Sprintf("body.%s", dsp.Structure.Format), dsp.BodyBytes), nil
	}

	loweredPath := strings.ToLower(dsp.BodyPath)

	// if opening protocol is http/s, we're dealing with a web request
	if strings.HasPrefix(loweredPath, "http://") || strings.HasPrefix(loweredPath, "https://") {
		// TODO - attempt to determine file format based on response headers
		filename := filepath.Base(dsp.BodyPath)

		res, err := http.Get(dsp.BodyPath)
		if err != nil {
			return nil, fmt.Errorf("fetching body url: %s", err.Error())
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("invalid status code fetching body url: %d", res.StatusCode)
		}

		// TODO - should this happen here? probs not.
		// consider moving to repo/actions.CreateDataset
		if dsp.Meta == nil {
			dsp.Meta = &dataset.Meta{}
		}
		if dsp.Meta.DownloadPath == "" {
			dsp.Meta.DownloadPath = dsp.BodyPath
		}
		// if we're adding from a dataset url, set a default accrual periodicity of once a week
		// this'll set us up to re-check urls over time
		// TODO - make this configurable via a param?
		if dsp.Meta.AccrualPeriodicity == "" {
			dsp.Meta.AccrualPeriodicity = "R/P1W"
		}

		return cafs.NewMemfileReader(filename, res.Body), nil
	} else if dsp.BodyPath != "" {
		// convert yaml input to json as a hack to support yaml input for now
		ext := strings.ToLower(filepath.Ext(dsp.BodyPath))
		if ext == ".yaml" || ext == ".yml" {
			yamlBody, err := ioutil.ReadFile(dsp.BodyPath)
			if err != nil {
				return nil, fmt.Errorf("reading body file: %s", err.Error())
			}
			jsonBody, err := yaml.YAMLToJSON(yamlBody)
			if err != nil {
				return nil, fmt.Errorf("converting yaml body to json: %s", err.Error())
			}

			filename := fmt.Sprintf("%s.json", strings.TrimSuffix(filepath.Base(dsp.BodyPath), ext))
			return cafs.NewMemfileBytes(filename, jsonBody), nil
		}

		file, err := os.Open(dsp.BodyPath)
		if err != nil {
			return nil, fmt.Errorf("reading body file: %s", err.Error())
		}

		return cafs.NewMemfileReader(filepath.Base(dsp.BodyPath), file), nil
	}

	// TODO - standardize this error:
	return nil, fmt.Errorf("not found")
}
