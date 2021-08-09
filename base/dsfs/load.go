package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// LoadDataset reads a dataset from a cafs and dereferences structure, transform, and commitMsg if they exist,
// returning a fully-hydrated dataset
func LoadDataset(ctx context.Context, store qfs.Filesystem, path string) (*dataset.Dataset, error) {
	log.Debugw("LoadDataset", "path", path)
	if store == nil {
		return nil, fmt.Errorf("loading dataset: store is nil")
	}

	// set a timeout to handle long-lived requests when connected to IPFS.
	// if we don't have the dataset locally, IPFS will reach out onto the d.web to
	// attempt to resolve previous hashes. capping the duration yeilds quicker results.
	// TODO (b5) - The proper way to solve this is to feed a local-only IPFS store
	// to this entire function, or have a mechanism for specifying that a fetch
	// must be local
	ctx, cancel := context.WithTimeout(ctx, OpenFileTimeoutDuration)
	defer cancel()

	ds, err := LoadDatasetRefs(ctx, store, path)
	if err != nil {
		log.Debugf("loading dataset: %s", err)
		return nil, fmt.Errorf("loading dataset: %w", err)
	}
	if err := DerefDataset(ctx, store, ds); err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	return ds, nil
}

// LoadDatasetRefs reads a dataset from a content addressed filesystem without
// dereferencing components
func LoadDatasetRefs(ctx context.Context, fs qfs.Filesystem, path string) (*dataset.Dataset, error) {
	pathWithBasename := PackageFilepath(fs, path, PackageFileDataset)
	log.Debugw("LoadDatasetPath", "packageFilepath", pathWithBasename)
	data, err := fileBytes(fs.Get(ctx, pathWithBasename))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("reading %s file: %w", PackageFileDataset.String(), err)
	}

	ds, err := dataset.UnmarshalDataset(data)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("unmarshaling %s file: %w", PackageFileDataset.String(), err)
	}

	// assign path to retain reference to the path this dataset was read from
	ds.Path = path

	return ds, nil
}

// DerefDataset attempts to fully dereference a dataset
func DerefDataset(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if err := DerefMeta(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefStructure(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefTransform(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefViz(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefReadme(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefStats(ctx, store, ds); err != nil {
		return err
	}
	return DerefCommit(ctx, store, ds)
}

// LoadBody loads the data this dataset points to from the store
func LoadBody(ctx context.Context, fs qfs.Filesystem, ds *dataset.Dataset) (qfs.File, error) {
	return fs.Get(ctx, ds.BodyPath)
}

// DerefCommit derferences a dataset's Commit element if required should be a
// no-op if ds.Commit is nil or isn't a reference
func DerefCommit(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Commit != nil && ds.Commit.IsEmpty() && ds.Commit.Path != "" {
		cm, err := loadCommit(ctx, store, ds.Commit.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("loading dataset commit: %w", err)
		}
		cm.Path = ds.Commit.Path
		ds.Commit = cm
	}
	return nil
}

func loadCommit(ctx context.Context, fs qfs.Filesystem, path string) (st *dataset.Commit, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("loading commit file: %s", err.Error())
	}
	return dataset.UnmarshalCommit(data)
}

// DerefMeta derferences a dataset's transform element if required should be a
// no-op if ds.Meta is nil or isn't a reference
func DerefMeta(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Meta != nil && ds.Meta.IsEmpty() && ds.Meta.Path != "" {
		md, err := loadMeta(ctx, store, ds.Meta.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("loading dataset metadata: %w", err)
		}
		md.Path = ds.Meta.Path
		ds.Meta = md
	}
	return nil
}

func loadMeta(ctx context.Context, fs qfs.Filesystem, path string) (md *dataset.Meta, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("loading metadata file: %w", err)
	}
	md = &dataset.Meta{}
	err = md.UnmarshalJSON(data)
	return md, err
}

// DerefReadme dereferences a dataset's Readme element if required no-op if
// ds.Readme is nil or isn't a reference
func DerefReadme(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Readme != nil && ds.Readme.IsEmpty() && ds.Readme.Path != "" {
		rm, err := loadReadme(ctx, store, ds.Readme.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("loading dataset readme: %w", err)
		}
		rm.Path = ds.Readme.Path
		ds.Readme = rm
	}
	return nil
}

func loadReadme(ctx context.Context, fs qfs.Filesystem, path string) (st *dataset.Readme, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading readme file: %w", err)
	}
	return dataset.UnmarshalReadme(data)
}

// LoadReadmeScript loads script data from a dataset path if the given dataset has a readme script is specified
// the returned qfs.File will be the value of dataset.Readme.ScriptPath
func LoadReadmeScript(ctx context.Context, fs qfs.Filesystem, dspath string) (qfs.File, error) {
	ds, err := LoadDataset(ctx, fs, dspath)
	if err != nil {
		return nil, err
	}

	if ds.Readme == nil || ds.Readme.ScriptPath == "" {
		return nil, ErrNoReadme
	}

	return fs.Get(ctx, ds.Readme.ScriptPath)
}

// DerefStats derferences a dataset's stats component if required
// no-op if ds.Stats is nil or isn't a reference
func DerefStats(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Stats != nil && ds.Stats.IsEmpty() && ds.Stats.Path != "" {
		sa, err := loadStats(ctx, store, ds.Stats.Path)
		if err != nil {
			log.Debug(err)
			return fmt.Errorf("loading stats component: %w", err)
		}
		sa.Path = ds.Stats.Path
		ds.Stats = sa
	}
	return nil
}

func loadStats(ctx context.Context, fs qfs.Filesystem, path string) (sa *dataset.Stats, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("loading stats file: %w", err)
	}
	sa = &dataset.Stats{}
	err = sa.UnmarshalJSON(data)
	return sa, err
}

// DerefTransform derferences a dataset's transform element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefTransform(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Transform != nil && ds.Transform.IsEmpty() && ds.Transform.Path != "" {
		t, err := loadTransform(ctx, store, ds.Transform.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("loading dataset transform: %w", err)
		}
		t.Path = ds.Transform.Path
		ds.Transform = t
	}
	return nil
}

func loadTransform(ctx context.Context, fs qfs.Filesystem, path string) (q *dataset.Transform, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading transform raw data: %s", err.Error())
	}

	return dataset.UnmarshalTransform(data)
}

// DerefStructure derferences a dataset's structure element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefStructure(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Structure != nil && ds.Structure.IsEmpty() && ds.Structure.Path != "" {
		st, err := loadStructure(ctx, store, ds.Structure.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("loading dataset structure: %w", err)
		}
		// assign path to retain internal reference to path
		st.Path = ds.Structure.Path
		ds.Structure = st
	}
	return nil
}

func loadStructure(ctx context.Context, fs qfs.Filesystem, path string) (st *dataset.Structure, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading structure file: %s", err.Error())
	}
	return dataset.UnmarshalStructure(data)
}

// DerefViz dereferences a dataset's Viz element if required
// no-op if ds.Viz is nil or isn't a reference
func DerefViz(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Viz != nil && ds.Viz.IsEmpty() && ds.Viz.Path != "" {
		vz, err := loadViz(ctx, store, ds.Viz.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("loading dataset viz: %w", err)
		}
		vz.Path = ds.Viz.Path
		ds.Viz = vz
	}
	return nil
}

func loadViz(ctx context.Context, fs qfs.Filesystem, path string) (st *dataset.Viz, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading viz file: %s", err.Error())
	}
	return dataset.UnmarshalViz(data)
}
