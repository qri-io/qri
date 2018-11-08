package base

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
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// ListDatasets lists datasets from a repo
func ListDatasets(r repo.Repo, limit, offset int, RPC, publishedOnly bool) (res []repo.DatasetRef, err error) {
	store := r.Store()
	res, err = r.References(limit, offset)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error getting dataset list: %s", err.Error())
	}

	if publishedOnly {
		pub := make([]repo.DatasetRef, len(res))
		i := 0
		for _, ref := range res {
			if ref.Published {
				pub[i] = ref
				i++
			}
		}
		res = pub[:i]
	}

	renames := repo.NewNeedPeernameRenames()
	for i, ref := range res {
		// May need to change peername.
		if err := repo.CanonicalizeProfile(r, &res[i], &renames); err != nil {
			return nil, fmt.Errorf("error canonicalizing dataset peername: %s", err.Error())
		}

		ds, err := dsfs.LoadDataset(store, datastore.NewKey(ref.Path))
		if err != nil {
			return nil, fmt.Errorf("error loading path: %s, err: %s", ref.Path, err.Error())
		}
		res[i].Dataset = ds.Encode()
		if RPC {
			res[i].Dataset.Structure.Schema = nil
		}
	}

	// TODO: If renames.Renames is non-empty, apply it to r
	return
}

// CreateDataset uses dsfs to add a dataset to a repo's store, updating all
// references within the repo if successful. CreateDataset is a lower-level
// component of github.com/qri-io/qri/actions.CreateDataset
func CreateDataset(r repo.Repo, streams ioes.IOStreams, name string, ds *dataset.Dataset, body cafs.File, dryRun, pin bool) (ref repo.DatasetRef, resBody cafs.File, err error) {
	var (
		pro  *profile.Profile
		path datastore.Key
	)

	pro, err = r.Profile()
	if err != nil {
		return
	}

	if ds.Commit != nil {
		// NOTE: add author ProfileID here to keep the dataset package agnostic to
		// all identity stuff except keypair crypto
		ds.Commit.Author = &dataset.User{ID: pro.ID.String()}
	}

	if err = prepareViz(ds); err != nil {
		return
	}

	if ds.Transform != nil && ds.Transform.IsEmpty() {
		ds.Transform = nil
	}

	if dryRun {
		// dry-runs store to an in-memory repo
		// TODO - memRepo needs to be able to load a previous dataset from our actual repo
		// memRepo should be able to wrap another repo & check that before returning not found
		r, err = repo.NewMemRepo(pro, cafs.NewMapstore(), profile.NewMemStore(), nil)
		if err != nil {
			return
		}
		// TODO - to compensate for the above issue, using this hack to get around that
		// dsfs.CreateDataset will attempt to load the previous dataset if there is a
		// PreviousPath set in the dataset ds
		ds.PreviousPath = ""
	}

	if path, err = dsfs.CreateDataset(r.Store(), ds, body, r.PrivateKey(), pin); err != nil {
		return
	}
	if ds.PreviousPath != "" && ds.PreviousPath != "/" {
		prev := repo.DatasetRef{
			ProfileID: pro.ID,
			Peername:  pro.Peername,
			Name:      name,
			Path:      ds.PreviousPath,
		}

		// should be ok to skip this error. we may not have the previous
		// reference locally
		_ = r.DeleteRef(prev)
	}
	ref = repo.DatasetRef{
		ProfileID: pro.ID,
		Peername:  pro.Peername,
		Name:      name,
		Path:      path.String(),
	}
	if err = r.PutRef(ref); err != nil {
		return
	}
	if err = r.LogEvent(repo.ETDsCreated, ref); err != nil {
		return
	}
	_, storeIsPinner := r.Store().(cafs.Pinner)
	if pin && storeIsPinner {
		r.LogEvent(repo.ETDsPinned, ref)
	}

	if err = ReadDataset(r, &ref); err != nil {
		return
	}
	if resBody, err = r.Store().Get(datastore.NewKey(ref.Dataset.BodyPath)); err != nil {
		fmt.Println("error getting from store:", err.Error())
	}
	return
}

// FetchDataset grabs a dataset from a remote source
func FetchDataset(r repo.Repo, ref *repo.DatasetRef, pin, load bool) (err error) {
	key := datastore.NewKey(strings.TrimSuffix(ref.Path, "/"+dsfs.PackageFileDataset.String()))
	path := datastore.NewKey(key.String() + "/" + dsfs.PackageFileDataset.String())

	fetcher, ok := r.Store().(cafs.Fetcher)
	if !ok {
		err = fmt.Errorf("this store cannot fetch from remote sources")
		return
	}

	// TODO: This is asserting that the target is Fetch-able, but inside dsfs.LoadDataset,
	// only Get is called. Clean up the semantics of Fetch and Get to get this expection
	// more correctly in line with what's actually required.
	_, err = fetcher.Fetch(cafs.SourceAny, key)
	if err != nil {
		return fmt.Errorf("error fetching file: %s", err.Error())
	}

	if pin {
		if err = PinDataset(r, *ref); err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error pinning root key: %s", err.Error())
		}
	}

	if load {
		ds, err := dsfs.LoadDataset(r.Store(), path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error loading newly saved dataset path: %s", path.String())
		}

		ref.Dataset = ds.Encode()
	}

	return
}

// ReadDataset grabs a dataset from the store
func ReadDataset(r repo.Repo, ref *repo.DatasetRef) (err error) {
	if store := r.Store(); store != nil {
		ds, e := dsfs.LoadDataset(store, datastore.NewKey(ref.Path))
		if e != nil {
			return e
		}
		ref.Dataset = ds.Encode()
		return
	}

	return datastore.ErrNotFound
}

// PinDataset marks a dataset for retention in a store
func PinDataset(r repo.Repo, ref repo.DatasetRef) error {
	if pinner, ok := r.Store().(cafs.Pinner); ok {
		pinner.Pin(datastore.NewKey(ref.Path), true)
		return r.LogEvent(repo.ETDsPinned, ref)
	}
	return repo.ErrNotPinner
}

// UnpinDataset unmarks a dataset for retention in a store
func UnpinDataset(r repo.Repo, ref repo.DatasetRef) error {
	if pinner, ok := r.Store().(cafs.Pinner); ok {
		pinner.Unpin(datastore.NewKey(ref.Path), true)
		return r.LogEvent(repo.ETDsUnpinned, ref)
	}
	return repo.ErrNotPinner
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
		if dsp.Meta.DownloadURL == "" {
			dsp.Meta.DownloadURL = dsp.BodyPath
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
				return nil, fmt.Errorf("body file: %s", err.Error())
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
			return nil, fmt.Errorf("body file: %s", err.Error())
		}

		return cafs.NewMemfileReader(filepath.Base(dsp.BodyPath), file), nil
	}

	// TODO - standardize this error:
	return nil, fmt.Errorf("not found")
}
