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
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// NewLocalDatasetLoader creates a dsfs.Loader that operates on a filestore
func NewLocalDatasetLoader(r repo.Repo) dsref.Loader {
	return loader{r}
}

type loader struct {
	r repo.Repo
}

// LoadDataset fetches, derefernces and opens a dataset from a reference
// implements the dsfs.Loader interface
func (l loader) LoadDataset(ctx context.Context, ref dsref.Ref, source string) (*dataset.Dataset, error) {
	if source != "" {
		return nil, fmt.Errorf("only local datasets can be loaded")
	}

	ds, err := dsfs.LoadDataset(ctx, l.r.Store(), ref.Path)
	if err != nil {
		return nil, err
	}

	err = OpenDataset(ctx, l.r.Filesystem(), ds)
	return ds, err
}

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
			// NOTE: Quick fix to avoid "merkledag: not found" errors that happen for new users.
			// Many datasets have default viz that point to an ipfs path that cloud will deliver,
			// but the pointed at ipfs file does not exist on the user's machine.
			if isMerkleDagError(err) {
				err = nil
			} else {
				log.Debug(err)
				return
			}
		}
	}

	if err = openReadme(ctx, fsys, ds); err != nil {
		return err
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

func isMerkleDagError(err error) bool {
	return err.Error() == "merkledag: not found"
}

func openReadme(ctx context.Context, fsys qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Readme != nil && ds.Readme.ScriptFile() == nil {
		readmeTimeoutCtx, cancel := context.WithTimeout(ctx, OpenFileTimeoutDuration)
		defer cancel()

		if err := ds.Readme.OpenScriptFile(readmeTimeoutCtx, fsys); err != nil {
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
	return nil
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
				dsVersions, err := DatasetLog(ctx, r, reporef.ConvertToDsref(ref), 1000000, 0, false)
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

// ReadDataset grabs a dataset from the store
//
// Deprecated - use LoadDataset instead
func ReadDataset(ctx context.Context, r repo.Repo, path string) (ds *dataset.Dataset, err error) {
	store := r.Store()
	if store == nil {
		return nil, cafs.ErrNotFound
	}

	return dsfs.LoadDataset(ctx, store, path)
}

// PinDataset marks a dataset for retention in a store
func PinDataset(ctx context.Context, r repo.Repo, path string) error {
	if pinner, ok := r.Store().(cafs.Pinner); ok {
		return pinner.Pin(ctx, path, true)
	}
	return repo.ErrNotPinner
}

// UnpinDataset unmarks a dataset for retention in a store
func UnpinDataset(ctx context.Context, r repo.Repo, path string) error {
	if pinner, ok := r.Store().(cafs.Pinner); ok {
		return pinner.Unpin(ctx, path, true)
	}
	return repo.ErrNotPinner
}
