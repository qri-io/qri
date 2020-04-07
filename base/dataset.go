package base

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
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

	if ds.Readme != nil && ds.Readme.ScriptFile() == nil {
		readmeTimeoutCtx, cancel := context.WithTimeout(ctx, OpenFileTimeoutDuration)
		defer cancel()

		if err = ds.Readme.OpenScriptFile(readmeTimeoutCtx, fsys); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				err = nil
			} else if strings.Contains(err.Error(), "not found") {
				log.Debug("skipping not-found readme script")
				err = nil
			} else {
				log.Debug(err)
				return err
			}
		}
	}

	if ds.Viz != nil && ds.Viz.RenderedFile() == nil {
		vizRenderedTimeoutCtx, cancel := context.WithTimeout(ctx, OpenFileTimeoutDuration)
		defer cancel()

		if err = ds.Viz.OpenRenderedFile(vizRenderedTimeoutCtx, fsys); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				err = nil
			} else if strings.Contains(err.Error(), "not found") {
				log.Debug("skipping not-found viz script")
				err = nil
			} else {
				log.Debug(err)
				return
			}
		}
	}
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
	if ds.Readme != nil && ds.Readme.ScriptFile() != nil {
		if err = ds.Readme.ScriptFile().Close(); err != nil {
			return
		}
	}
	if ds.Readme != nil && ds.Readme.RenderedFile() != nil {
		if err = ds.Readme.RenderedFile().Close(); err != nil {
			return
		}
	}

	return
}

// ListDatasets lists datasets from a repo
func ListDatasets(ctx context.Context, r repo.Repo, term string, limit, offset int, RPC, publishedOnly, showVersions bool) (res []reporef.DatasetRef, err error) {
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

	if publishedOnly {
		pub := make([]reporef.DatasetRef, len(res))
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
		return []reporef.DatasetRef{}, nil
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
			ds.Peername = res[i].Peername
			ds.Name = res[i].Name
			res[i].Dataset = ds
			if RPC {
				res[i].Dataset.Structure.Schema = nil
			}

			if showVersions {
				dsVersions, err := DatasetLog(ctx, r, ref, 1000000, 0, false)
				if err != nil {
					return nil, err
				}
				res[i].Dataset.NumVersions = len(dsVersions)
			}
		}
	}

	if term != "" {
		matched := make([]reporef.DatasetRef, len(res))
		i := 0
		for _, ref := range res {
			if strings.Contains(ref.AliasString(), term) {
				matched[i] = ref
				i++
			}
		}
		res = matched[:i]
	}

	return
}

// RawDatasetRefs converts the dataset refs to a string
func RawDatasetRefs(ctx context.Context, r repo.Repo) (string, error) {
	num, err := r.RefCount()
	if err != nil {
		return "", err
	}
	res, err := r.References(0, num)
	if err != nil {
		log.Debug(err.Error())
		return "", fmt.Errorf("error getting dataset list: %s", err.Error())
	}

	// Calculate the largest index, and get its length
	width := len(fmt.Sprintf("%d", num-1))
	// Padding for each row to stringify
	padding := strings.Repeat(" ", width)
	// A printf template for stringifying indexes, such that they all have the same size
	numTemplate := fmt.Sprintf("%%%dd", width)

	builder := strings.Builder{}
	for n, ref := range res {
		datasetNum := fmt.Sprintf(numTemplate, n)
		fmt.Fprintf(&builder, "%s Peername:  %s\n", datasetNum, ref.Peername)
		fmt.Fprintf(&builder, "%s ProfileID: %s\n", padding, ref.ProfileID)
		fmt.Fprintf(&builder, "%s Name:      %s\n", padding, ref.Name)
		fmt.Fprintf(&builder, "%s Path:      %s\n", padding, ref.Path)
		fmt.Fprintf(&builder, "%s FSIPath:   %s\n", padding, ref.FSIPath)
		fmt.Fprintf(&builder, "%s Published: %v\n", padding, ref.Published)
	}
	return builder.String(), nil
}

// FetchDataset grabs a dataset from a remote source
func FetchDataset(ctx context.Context, r repo.Repo, ref *reporef.DatasetRef, pin, load bool) (err error) {
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
// The medium-term goal here is to obfuscate use of reporef.DatasetRef, which we're hoping to deprecate
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
func ReadDataset(ctx context.Context, r repo.Repo, ref *reporef.DatasetRef) (err error) {
	if err = repo.CanonicalizeDatasetRef(r, ref); err != nil {
		return
	}

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
func PinDataset(ctx context.Context, r repo.Repo, ref reporef.DatasetRef) error {
	if pinner, ok := r.Store().(cafs.Pinner); ok {
		return pinner.Pin(ctx, ref.Path, true)
	}
	return repo.ErrNotPinner
}

// UnpinDataset unmarks a dataset for retention in a store
func UnpinDataset(ctx context.Context, r repo.Repo, ref reporef.DatasetRef) error {
	if pinner, ok := r.Store().(cafs.Pinner); ok {
		return pinner.Unpin(ctx, ref.Path, true)
	}
	return repo.ErrNotPinner
}
