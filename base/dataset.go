package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// OpenDataset prepares a dataset for use, checking each component
// for populated Path or Byte suffixed fields, consuming those fields to
// set File handlers that are ready for reading
func OpenDataset(ctx context.Context, fsys qfs.Filesystem, ds *dataset.Dataset) (err error) {
	if ds.BodyFile() == nil {
		if err = ds.OpenBodyFile(ctx, fsys); err != nil {
			log.Debug(err)
			return
		}
	}
	if ds.Transform != nil && ds.Transform.ScriptFile() == nil {
		if err = ds.Transform.OpenScriptFile(ctx, fsys); err != nil {
			log.Debug(err)
			return
		}
	}
	if ds.Viz != nil && ds.Viz.ScriptFile() == nil {
		if err = ds.Viz.OpenScriptFile(ctx, fsys); err != nil {
			log.Debug(err)
			return
		}
	}

	// TODO (b5) - this is an error sometimes caused by failing to properly pin the
	// rendered file. this'll really trip up if we're online, b/c ipfs will hang
	// forever looking for the missing render dag :(
	// if ds.Viz != nil && ds.Viz.RenderedFile() == nil {
	// 	if err = ds.Viz.OpenRenderedFile(ctx, fsys); err != nil {
	// 		if strings.Contains(err.Error(), "not found") {
	// 			log.Debug("skipping not-found viz script")
	// 			err = nil
	// 		} else {
	// 			log.Debug(err)
	// 			return
	// 		}
	// 	}
	// }
	return
}

// CloseDataset ensures all open dataset files are closed
func CloseDataset(ds *dataset.Dataset) (err error) {
	if ds.BodyFile() != nil {
		if err = ds.BodyFile().Close(); err != nil {
			return
		}
	}
	if ds.Transform != nil && ds.Transform.ScriptFile() != nil {
		if err = ds.Transform.ScriptFile().Close(); err != nil {
			return
		}
	}
	if ds.Viz != nil && ds.Viz.ScriptFile() != nil {
		if err = ds.Viz.ScriptFile().Close(); err != nil {
			return
		}
	}
	if ds.Viz != nil && ds.Viz.RenderedFile() != nil {
		if err = ds.Viz.RenderedFile().Close(); err != nil {
			return
		}
	}

	return
}

// ListDatasets lists datasets from a repo
func ListDatasets(ctx context.Context, r repo.Repo, term string, limit, offset int, RPC, publishedOnly, showVersions bool) (res []repo.DatasetRef, err error) {
	store := r.Store()
	num, err := r.RefCount()
	if err != nil {
		return nil, err
	}
	res, err = r.References(0, num)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error getting dataset list: %s", err.Error())
	}

	if term != "" {
		matched := make([]repo.DatasetRef, len(res))
		i := 0
		for _, ref := range res {
			if strings.Contains(ref.Name, term) {
				matched[i] = ref
				i++
			}
		}
		res = matched[:i]
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
	// if offset is too high, return empty list
	if offset >= len(res) {
		return []repo.DatasetRef{}, nil
	}
	res = res[offset:]

	if limit < len(res) {
		res = res[:limit]
	}

	for i, ref := range res {
		// May need to change peername.
		if err := repo.CanonicalizeProfile(r, &res[i]); err != nil {
			return nil, fmt.Errorf("error canonicalizing dataset peername: %s", err.Error())
		}

		if ref.Path != "" {
			ds, err := dsfs.LoadDataset(ctx, store, ref.Path)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					res[i].Foreign = true
					err = nil
					continue
				}
				return nil, fmt.Errorf("error loading ref: %s, err: %s", ref.String(), err.Error())
			}
			res[i].Dataset = ds
			if RPC {
				res[i].Dataset.Structure.Schema = nil
			}

			if showVersions {
				dsVersions, err := DatasetLog(ctx, r, ref, 0, 0, false)
				if err != nil {
					return nil, err
				}
				res[i].Dataset.NumVersions = len(dsVersions)
			}
		}
	}

	return
}

// CreateDataset uses dsfs to add a dataset to a repo's store, updating all
// references within the repo if successful. CreateDataset is a lower-level
// component of github.com/qri-io/qri/actions.CreateDataset
func CreateDataset(ctx context.Context, r repo.Repo, streams ioes.IOStreams, ds, dsPrev *dataset.Dataset, dryRun, pin, force, shouldRender bool) (ref repo.DatasetRef, err error) {
	var (
		pro     *profile.Profile
		path    string
		resBody qfs.File
	)

	pro, err = r.Profile()
	if err != nil {
		return
	}

	if err = ValidateDataset(ds); err != nil {
		return
	}

	if path, err = dsfs.CreateDataset(ctx, r.Store(), ds, dsPrev, r.PrivateKey(), pin, force, shouldRender); err != nil {
		return
	}
	if ds.PreviousPath != "" && ds.PreviousPath != "/" {
		prev := repo.DatasetRef{
			ProfileID: pro.ID,
			Peername:  pro.Peername,
			Name:      ds.Name,
			Path:      ds.PreviousPath,
		}

		// should be ok to skip this error. we may not have the previous
		// reference locally
		_ = r.DeleteRef(prev)
	}
	ref = repo.DatasetRef{
		ProfileID: pro.ID,
		Peername:  pro.Peername,
		Name:      ds.Name,
		Path:      path,
	}

	if err = r.PutRef(ref); err != nil {
		return
	}

	// TODO (b5): confirm these assignments happen in dsfs.CreateDataset with tests
	ds.ProfileID = pro.ID.String()
	ds.Peername = pro.Peername
	ds.Path = path
	if err = r.Logbook().WriteVersionSave(ctx, ds); err != nil {
		return
	}

	if err = ReadDataset(ctx, r, &ref); err != nil {
		return
	}

	// need to open here b/c we might be doing a dry-run, which would mean we have
	// references to files in a store that won't exist after this function call
	// TODO (b5): this should be replaced with a call to OpenDataset with a qfs that
	// knows about the store
	if resBody, err = r.Store().Get(ctx, ref.Dataset.BodyPath); err != nil {
		log.Error("error getting from store:", err.Error())
	}
	ref.Dataset.SetBodyFile(resBody)
	return
}

// FetchDataset grabs a dataset from a remote source
func FetchDataset(ctx context.Context, r repo.Repo, ref *repo.DatasetRef, pin, load bool) (err error) {
	key := strings.TrimSuffix(ref.Path, "/"+dsfs.PackageFileDataset.String())
	// TODO (b5): use a function from a canonical place to produce this path, possibly from dsfs
	path := key + "/" + dsfs.PackageFileDataset.String()

	fetcher, ok := r.Store().(cafs.Fetcher)
	if !ok {
		err = fmt.Errorf("this store cannot fetch from remote sources")
		return
	}

	// TODO: This is asserting that the target is Fetch-able, but inside dsfs.LoadDataset,
	// only Get is called. Clean up the semantics of Fetch and Get to get this expection
	// more correctly in line with what's actually required.
	_, err = fetcher.Fetch(ctx, cafs.SourceAny, path)
	if err != nil {
		return fmt.Errorf("error fetching file: %s", err.Error())
	}

	if pin {
		if err = PinDataset(ctx, r, *ref); err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error pinning root key: %s", err.Error())
		}
	}

	if load {
		ds, err := dsfs.LoadDataset(ctx, r.Store(), path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error loading newly saved dataset path: %s", path)
		}
		ref.Dataset = ds
	}

	return
}

// ReadDatasetPath takes a path string, parses, canonicalizes, loads a dataset pointer, and opens the file
// The medium-term goal here is to obfuscate use of repo.DatasetRef, which we're hoping to deprecate
func ReadDatasetPath(ctx context.Context, r repo.Repo, path string) (ds *dataset.Dataset, err error) {
	ref, err := repo.ParseDatasetRef(path)
	if err != nil {
		return nil, fmt.Errorf("'%s' is not a valid dataset reference", path)
	}

	if err = repo.CanonicalizeDatasetRef(r, &ref); err != nil {
		return
	}

	loaded, err := dsfs.LoadDataset(ctx, r.Store(), ref.Path)
	if err != nil {
		return nil, fmt.Errorf("error loading dataset")
	}
	loaded.Name = ref.Name
	loaded.Peername = ref.Peername
	ds = loaded

	err = OpenDataset(ctx, r.Filesystem(), ds)
	return
}

// ReadDataset grabs a dataset from the store
func ReadDataset(ctx context.Context, r repo.Repo, ref *repo.DatasetRef) (err error) {
	if store := r.Store(); store != nil {
		ds, e := dsfs.LoadDataset(ctx, store, ref.Path)
		if e != nil {
			return e
		}
		ds.Name = ref.Name
		ds.Peername = ref.Peername
		ref.Dataset = ds
		return
	}

	return cafs.ErrNotFound
}

// PinDataset marks a dataset for retention in a store
func PinDataset(ctx context.Context, r repo.Repo, ref repo.DatasetRef) error {
	if pinner, ok := r.Store().(cafs.Pinner); ok {
		return pinner.Pin(ctx, ref.Path, true)
	}
	return repo.ErrNotPinner
}

// UnpinDataset unmarks a dataset for retention in a store
func UnpinDataset(ctx context.Context, r repo.Repo, ref repo.DatasetRef) error {
	if pinner, ok := r.Store().(cafs.Pinner); ok {
		return pinner.Unpin(ctx, ref.Path, true)
	}
	return repo.ErrNotPinner
}

// InlineJSONBody reads the contents dataset.BodyFile() into a json.RawMessage,
// assigning the result to dataset.Body
func InlineJSONBody(ds *dataset.Dataset) error {
	file := ds.BodyFile()
	if file == nil {
		log.Error("no body file")
		return fmt.Errorf("no response body file")
	}

	if ds.Structure.Format == dataset.JSONDataFormat.String() {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		ds.Body = json.RawMessage(data)
		return nil
	}

	in := ds.Structure
	st := &dataset.Structure{}
	st.Assign(in, &dataset.Structure{
		Format: "json",
		Schema: in.Schema,
	})

	data, err := ConvertBodyFile(file, in, st, 0, 0, true)
	if err != nil {
		log.Errorf("converting body file to JSON: %s", err)
		return fmt.Errorf("converting body file to JSON: %s", err)
	}

	ds.Body = json.RawMessage(data)
	return nil
}

// ConvertBodyFile takes an input file & structure, and converts a specified selection
// to the structure specified by out
func ConvertBodyFile(file qfs.File, in, out *dataset.Structure, limit, offset int, all bool) (data []byte, err error) {
	buf := &bytes.Buffer{}

	w, err := dsio.NewEntryWriter(out, buf)
	if err != nil {
		return
	}

	// TODO(dlong): Kind of a hacky one-off. Generalize this for other format options.
	if out.DataFormat() == dataset.JSONDataFormat {
		ok, pretty := out.FormatConfig["pretty"].(bool)
		if ok && pretty {
			w, err = dsio.NewJSONPrettyWriter(out, buf, " ")
		}
	}
	if err != nil {
		return
	}

	rr, err := dsio.NewEntryReader(in, file)
	if err != nil {
		err = fmt.Errorf("error allocating data reader: %s", err)
		return
	}

	if !all {
		rr = &dsio.PagedReader{
			Reader: rr,
			Limit:  limit,
			Offset: offset,
		}
	}
	err = dsio.Copy(rr, w)

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("error closing row buffer: %s", err.Error())
	}

	return buf.Bytes(), nil
}

// DatasetBodyFile creates a streaming data file from a Dataset using the following precedence:
// * ds.BodyBytes not being nil (requires ds.Structure.Format be set to know data format)
// * ds.BodyPath being a url
// * ds.BodyPath being a path on the local filesystem
// TODO - consider moving this func to some other package. maybe actions?
func DatasetBodyFile(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset) (qfs.File, error) {
	if ds.BodyBytes != nil {
		if ds.Structure == nil || ds.Structure.Format == "" {
			return nil, fmt.Errorf("specifying bodyBytes requires format be specified in dataset.structure")
		}
		return qfs.NewMemfileBytes(fmt.Sprintf("body.%s", ds.Structure.Format), ds.BodyBytes), nil
	}

	// all other methods are based on path, bail if we don't have one
	if ds.BodyPath == "" {
		return nil, nil
	}

	loweredPath := strings.ToLower(ds.BodyPath)

	// if opening protocol is http/s, we're dealing with a web request
	if strings.HasPrefix(loweredPath, "http://") || strings.HasPrefix(loweredPath, "https://") {
		// TODO - attempt to determine file format based on response headers
		filename := filepath.Base(ds.BodyPath)

		res, err := http.Get(ds.BodyPath)
		if err != nil {
			return nil, fmt.Errorf("fetching body url: %s", err.Error())
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("invalid status code fetching body url: %d", res.StatusCode)
		}

		return qfs.NewMemfileReader(filename, res.Body), nil
	}

	if strings.HasPrefix(ds.BodyPath, "/ipfs") || strings.HasPrefix(ds.BodyPath, "/cafs") || strings.HasPrefix(ds.BodyPath, "/map") {
		return store.Get(ctx, ds.BodyPath)
	}

	// convert yaml input to json as a hack to support yaml input for now
	ext := strings.ToLower(filepath.Ext(ds.BodyPath))
	if ext == ".yaml" || ext == ".yml" {
		yamlBody, err := ioutil.ReadFile(ds.BodyPath)
		if err != nil {
			return nil, fmt.Errorf("body file: %s", err.Error())
		}
		jsonBody, err := yaml.YAMLToJSON(yamlBody)
		if err != nil {
			return nil, fmt.Errorf("converting yaml body to json: %s", err.Error())
		}

		filename := fmt.Sprintf("%s.json", strings.TrimSuffix(filepath.Base(ds.BodyPath), ext))
		return qfs.NewMemfileBytes(filename, jsonBody), nil
	}

	file, err := os.Open(ds.BodyPath)
	if err != nil {
		return nil, fmt.Errorf("body file: %s", err.Error())
	}

	return qfs.NewMemfileReader(filepath.Base(ds.BodyPath), file), nil
}

// ConvertBodyFormat rewrites a body from a source format to a destination format.
func ConvertBodyFormat(bodyFile qfs.File, fromSt, toSt *dataset.Structure) (qfs.File, error) {
	// Reader for entries of the source body.
	r, err := dsio.NewEntryReader(fromSt, bodyFile)
	if err != nil {
		return nil, err
	}

	// Writes entries to a new body.
	buffer := &bytes.Buffer{}
	w, err := dsio.NewEntryWriter(toSt, buffer)
	if err != nil {
		return nil, err
	}

	err = dsio.Copy(r, w)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}

	return qfs.NewMemfileReader(fmt.Sprintf("body.%s", toSt.Format), buffer), nil
}
