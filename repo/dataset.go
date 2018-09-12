package repo

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo/profile"
)

// CreateDataset uses dsfs to add a dataset to a repo's store, updating all
// references within the repo if successful. CreateDataset is a lower-level
// component of github.com/qri-io/qri/actions.CreateDataset
func CreateDataset(r Repo, name string, ds *dataset.Dataset, body cafs.File, pin bool) (ref DatasetRef, err error) {
	var (
		path datastore.Key
		pro  *profile.Profile
	)
	if pro, err = r.Profile(); err != nil {
		return
	}
	if path, err = dsfs.CreateDataset(r.Store(), ds, body, r.PrivateKey(), pin); err != nil {
		return
	}

	if ds.PreviousPath != "" && ds.PreviousPath != "/" {
		prev := DatasetRef{
			ProfileID: pro.ID,
			Peername:  pro.Peername,
			Name:      name,
			Path:      ds.PreviousPath,
		}

		// should be ok to skip this error. we may not have the previous
		// reference locally
		_ = r.DeleteRef(prev)
	}

	ref = DatasetRef{
		ProfileID: pro.ID,
		Peername:  pro.Peername,
		Name:      name,
		Path:      path.String(),
	}

	err = r.PutRef(ref)
	return
}

// DatasetPodBodyFile creates a streaming data file from a DatasetPod using the following precedence:
// * dsp.BodyBytes not being nil (requires dsp.Structure.Format be set to know data format)
// * dsp.BodyPath being a url
// * dsp.BodyPath being a path on the local filesystem
// TODO - consider moving this func to some other package. maybe actions?
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
		// consider moving to actions.CreateDataset
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
