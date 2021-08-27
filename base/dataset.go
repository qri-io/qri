package base

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/profile"
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
			return fmt.Errorf("opening body file: %w", err)
		}
	}
	if ds.Transform != nil && ds.Transform.ScriptFile() == nil {
		if err = ds.Transform.OpenScriptFile(ctx, fsys); err != nil {
			log.Debug(err)
			return fmt.Errorf("opening transform file: %w", err)
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
				return fmt.Errorf("opening viz scriptFile: %w", err)
			}
		}
	}

	if err = openReadme(ctx, fsys, ds); err != nil {
		return fmt.Errorf("opening readme file: %w", err)
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
func ListDatasets(ctx context.Context, r repo.Repo, term, profileID string, offset, limit int, publishedOnly, showVersions bool) ([]dsref.VersionInfo, error) {
	fs := r.Filesystem()
	num, err := r.RefCount()
	if err != nil {
		return nil, err
	}
	if limit < 0 {
		limit = num
	}
	refs, err := r.References(0, num)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error getting dataset list: %s", err.Error())
	}

	if publishedOnly {
		pub := make([]reporef.DatasetRef, len(refs))
		i := 0
		for _, ref := range refs {
			if ref.Published {
				pub[i] = ref
				i++
			}
		}
		refs = pub[:i]
	}
	// if offset is too high, return empty list
	if offset >= len(refs) {
		return []dsref.VersionInfo{}, nil
	}
	refs = refs[offset:]

	// Collect references that get resolved and match any given filters
	matches := make([]dsref.VersionInfo, 0, len(refs))
	hasUnlistableRefs := false

	for _, ref := range refs {
		if pros, err := r.Profiles().ProfilesForUsername(ctx, ref.Peername); err != nil || len(pros) > 1 {
			// This occurs when two profileIDs map to the same username, which can happen
			// when a user creates a new profile using an old username. We should ignore
			// references that can't be resolved this way, since other references in
			// the repository are still usable
			hasUnlistableRefs = true
			continue
		}

		if term != "" {
			// If this operation has a term to filter on, skip references that don't match
			if !strings.Contains(ref.AliasString(), term) {
				continue
			}
		}
		if profileID != "" {
			if profileID != ref.ProfileID.Encode() {
				continue
			}
		}

		if ref.Path != "" {
			ds, err := dsfs.LoadDataset(ctx, fs, ref.Path)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					ref.Foreign = true
					err = nil
					continue
				}
				return nil, fmt.Errorf("error loading ref: %s, err: %s", ref.String(), err.Error())
			}
			ds.Peername = ref.Peername
			ds.Name = ref.Name
			ref.Dataset = ds

			if showVersions {
				dsVersions, err := DatasetLog(ctx, r, reporef.ConvertToDsref(ref), 1000000, 0, false)
				if err != nil {
					return nil, err
				}
				ref.Dataset.NumVersions = len(dsVersions)
			}
		}

		matches = append(matches, reporef.ConvertToVersionInfo(&ref))
	}

	if limit < len(matches) {
		matches = matches[:limit]
	}

	// If some references could not be listed, return the other, valid references
	// and return a known error, which callers can handle as desired.
	if hasUnlistableRefs {
		return matches, ErrUnlistableReferences
	}

	return matches, nil
}

// RawDatasetRefs converts the dataset refs to a string
func RawDatasetRefs(ctx context.Context, pid profile.ID, s collection.Set) (string, error) {
	res, err := s.List(ctx, pid, params.ListAll)
	if err != nil {
		return "", err
	}

	// Calculate the largest index, and get its length
	width := len(fmt.Sprintf("%d", len(res)-1))
	// Padding for each row to stringify
	padding := strings.Repeat(" ", width)
	// A printf template for stringifying indexes, such that they all have the same size
	numTemplate := fmt.Sprintf("%%%dd", width)

	builder := strings.Builder{}
	for n, ref := range res {
		datasetNum := fmt.Sprintf(numTemplate, n)
		fmt.Fprintf(&builder, "%s Peername:  %s\n", datasetNum, ref.Username)
		fmt.Fprintf(&builder, "%s ProfileID: %s\n", padding, ref.ProfileID)
		fmt.Fprintf(&builder, "%s Name:      %s\n", padding, ref.Name)
		fmt.Fprintf(&builder, "%s Path:      %s\n", padding, ref.Path)
		fmt.Fprintf(&builder, "%s Published: %v\n", padding, ref.Published)
	}
	return builder.String(), nil
}

// ReadDataset grabs a dataset from the store
//
// Deprecated - use LoadDataset instead
func ReadDataset(ctx context.Context, r repo.Repo, path string) (ds *dataset.Dataset, err error) {
	fs := r.Filesystem()
	if fs == nil {
		return nil, qfs.ErrNotFound
	}

	return dsfs.LoadDataset(ctx, fs, path)
}
