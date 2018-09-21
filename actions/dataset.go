package actions

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/varName"
)

// NewDataset processes dataset input into it's necessary components for creation
func NewDataset(dsp *dataset.DatasetPod) (ds *dataset.Dataset, body cafs.File, secrets map[string]string, err error) {
	if dsp == nil {
		err = fmt.Errorf("dataset is required")
		return
	}

	if dsp.BodyPath == "" && dsp.BodyBytes == nil && dsp.Transform == nil {
		err = fmt.Errorf("either dataBytes, bodyPath, or a transform is required to create a dataset")
		return
	}

	if dsp.Transform != nil {
		secrets = dsp.Transform.Secrets
	}

	ds = &dataset.Dataset{}
	if err = ds.Decode(dsp); err != nil {
		err = fmt.Errorf("decoding dataset: %s", err.Error())
		return
	}

	if ds.Commit == nil {
		ds.Commit = &dataset.Commit{
			Title: "created dataset",
		}
	} else if ds.Commit.Title == "" {
		ds.Commit.Title = "created dataset"
	}

	// open a data file if we can
	if body, err = repo.DatasetPodBodyFile(dsp); err == nil {
		// defer body.Close()

		// validate / generate dataset name
		if dsp.Name == "" {
			dsp.Name = varName.CreateVarNameFromString(body.FileName())
		}
		if e := validate.ValidName(dsp.Name); e != nil {
			err = fmt.Errorf("invalid name: %s", e.Error())
			return
		}

		// read structure from InitParams, or detect from data
		if ds.Structure == nil && ds.Transform == nil {
			// use a TeeReader that writes to a buffer to preserve data
			buf := &bytes.Buffer{}
			tr := io.TeeReader(body, buf)
			var df dataset.DataFormat

			df, err = detect.ExtensionDataFormat(body.FileName())
			if err != nil {
				log.Debug(err.Error())
				err = fmt.Errorf("invalid data format: %s", err.Error())
				return
			}

			ds.Structure, _, err = detect.FromReader(df, tr)
			if err != nil {
				log.Debug(err.Error())
				err = fmt.Errorf("determining dataset schema: %s", err.Error())
				return
			}
			// glue whatever we just read back onto the reader
			body = cafs.NewMemfileReader(body.FileName(), io.MultiReader(buf, body))
		}

		// Ensure that dataset structure is valid
		if err = validate.Dataset(ds); err != nil {
			log.Debug(err.Error())
			err = fmt.Errorf("invalid dataset: %s", err.Error())
			return
		}

		// NOTE - if we have a data file, this overrides any transformation,
		// so we need to remove the transform to avoid having the data appear to be
		// the result of a transform process
		ds.Transform = nil

	} else if err.Error() == "not found" {
		err = nil
	} else {
		return
	}

	return
}

// UpdateDataset prepares a set of changes for submission to CreateDataset
func UpdateDataset(node *p2p.QriNode, dsp *dataset.DatasetPod) (ds *dataset.Dataset, body cafs.File, secrets map[string]string, err error) {
	ds = &dataset.Dataset{}
	updates := &dataset.Dataset{}

	if dsp == nil {
		err = fmt.Errorf("dataset is required")
		return
	}
	if dsp.Name == "" || dsp.Peername == "" {
		err = fmt.Errorf("peername & name are required to update dataset")
		return
	}

	if dsp.Transform != nil {
		secrets = dsp.Transform.Secrets
	}

	if err = updates.Decode(dsp); err != nil {
		err = fmt.Errorf("decoding dataset: %s", err.Error())
		return
	}

	prev := &repo.DatasetRef{Name: dsp.Name, Peername: dsp.Peername}
	if err = repo.CanonicalizeDatasetRef(node.Repo, prev); err != nil {
		err = fmt.Errorf("error with previous reference: %s", err.Error())
		return
	}

	if err = DatasetHead(node, prev); err != nil {
		err = fmt.Errorf("error getting previous dataset: %s", err.Error())
		return
	}

	if dsp.BodyBytes != nil || dsp.BodyPath != "" {
		if body, err = repo.DatasetPodBodyFile(dsp); err != nil {
			return
		}
	} else {
		// load data cause we need something to compare the structure to
		// prevDs := &dataset.Dataset{}
		// if err = prevDs.Decode(prev.Dataset); err != nil {
		// 	err = fmt.Errorf("error decoding previous dataset: %s", err)
		// 	return
		// }
	}

	prevds, err := prev.DecodeDataset()
	if err != nil {
		err = fmt.Errorf("error decoding dataset: %s", err.Error())
		return
	}

	// add all previous fields and any changes
	ds.Assign(prevds, updates)
	ds.PreviousPath = prev.Path

	// ds.Assign clobbers empty commit messages with the previous
	// commit message, reassign with updates
	if updates.Commit == nil {
		updates.Commit = &dataset.Commit{}
	}
	ds.Commit.Title = updates.Commit.Title
	ds.Commit.Message = updates.Commit.Message

	// TODO - this is so bad. fix. currently createDataset expects paths to
	// local files, so we're just making them up on the spot.
	if ds.Transform != nil && ds.Transform.ScriptPath[:len("/ipfs/")] == "/ipfs/" {
		tfScript, e := node.Repo.Store().Get(datastore.NewKey(ds.Transform.ScriptPath))
		if e != nil {
			err = e
			return
		}

		f, e := ioutil.TempFile("", "transform.sky")
		if e != nil {
			err = e
			return
		}
		if _, e := io.Copy(f, tfScript); err != nil {
			err = e
			return
		}
		ds.Transform.ScriptPath = f.Name()
	}
	if ds.Viz != nil && ds.Viz.ScriptPath[:len("/ipfs/")] == "/ipfs/" {
		vizScript, e := node.Repo.Store().Get(datastore.NewKey(ds.Viz.ScriptPath))
		if e != nil {
			err = e
			return
		}

		f, e := ioutil.TempFile("", "viz.html")
		if e != nil {
			err = e
			return
		}
		if _, e := io.Copy(f, vizScript); err != nil {
			err = e
			return
		}
		ds.Viz.ScriptPath = f.Name()
	}

	// Assign will assign any previous paths to the current paths
	// the dsdiff (called in dsfs.CreateDataset), will compare the paths
	// see that they are the same, and claim there are no differences
	// since we will potentially have changes in the Meta and Structure
	// we want the differ to have to compare each field
	// so we reset the paths
	if ds.Meta != nil {
		ds.Meta.SetPath("")
	}
	if ds.Structure != nil {
		ds.Structure.SetPath("")
	}

	return
}

// CreateDataset initializes a dataset from a dataset pointer and data file
func CreateDataset(node *p2p.QriNode, name string, ds *dataset.Dataset, data cafs.File, secrets map[string]string, dryRun, pin bool) (ref repo.DatasetRef, body cafs.File, err error) {
	var (
		r   = node.Repo
		pro *profile.Profile
		// NOTE - struct fields need to be instantiated to make assign set to
		// new pointer values
		userSet = &dataset.Dataset{
			Commit:    &dataset.Commit{},
			Meta:      &dataset.Meta{},
			Structure: &dataset.Structure{},
			Transform: &dataset.Transform{},
			Viz:       &dataset.Viz{},
		}
	)
	pro, err = r.Profile()
	if err != nil {
		return
	}

	userSet.Assign(ds)

	if ds.Commit != nil {
		// NOTE: add author ProfileID here to keep the dataset package agnostic to
		// all identity stuff except keypair crypto
		ds.Commit.Author = &dataset.User{ID: pro.ID.String()}
	}

	if ds.Transform != nil {
		node.LocalStreams.Print("🤖 executing transform\n")
		data, err = ExecTransform(node, ds, data, secrets)
		if err != nil {
			return
		}
		node.LocalStreams.Print("✅ transform complete\n")
		ds.Assign(userSet)
	}

	if err = PrepareViz(ds); err != nil {
		return
	}

	if dryRun {
		// dry-runs store to an in-memory repo
		node.LocalStreams.Print("🏃🏽‍♀️ dry run\n")
		r, err = repo.NewMemRepo(pro, cafs.NewMapstore(), profile.NewMemStore(), nil)
		if err != nil {
			return
		}
	}

	if ref, err = repo.CreateDataset(r, name, ds, data, pin); err != nil {
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

	if body, err = r.Store().Get(datastore.NewKey(ref.Dataset.BodyPath)); err != nil {
		fmt.Println("error getting from store:", err.Error())
	}

	return
}

// AddDataset fetches & pins a dataset to the store, adding it to the list of stored refs
func AddDataset(node *p2p.QriNode, ref *repo.DatasetRef) (err error) {
	err = repo.CanonicalizeDatasetRef(node.Repo, ref)
	if err == nil {
		return fmt.Errorf("error: dataset %s already exists in repo", ref)
	} else if err != repo.ErrNotFound {
		return fmt.Errorf("error with new reference: %s", err.Error())
	}

	if ref.Path == "" && node != nil {
		if err := node.RequestDataset(ref); err != nil {
			return fmt.Errorf("error requesting dataset: %s", err.Error())
		}
	}

	r := node.Repo
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

	if err = PinDataset(r, *ref); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error pinning root key: %s", err.Error())
	}

	if err = r.PutRef(*ref); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error putting dataset name in repo: %s", err.Error())
	}

	ds, err := dsfs.LoadDataset(r.Store(), path)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error loading newly saved dataset path: %s", path.String())
	}

	ref.Dataset = ds.Encode()
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

// RenameDataset alters a dataset name
func RenameDataset(node *p2p.QriNode, current, new *repo.DatasetRef) (err error) {
	r := node.Repo
	if err := validate.ValidName(new.Name); err != nil {
		return err
	}
	if err := repo.CanonicalizeDatasetRef(r, current); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error with existing reference: %s", err.Error())
	}
	err = repo.CanonicalizeDatasetRef(r, new)
	if err == nil {
		return fmt.Errorf("dataset '%s/%s' already exists", new.Peername, new.Name)
	} else if err != repo.ErrNotFound {
		log.Debug(err.Error())
		return fmt.Errorf("error with new reference: %s", err.Error())
	}
	new.Path = current.Path

	if err = r.DeleteRef(*current); err != nil {
		return err
	}
	if err = r.PutRef(*new); err != nil {
		return err
	}

	return r.LogEvent(repo.ETDsRenamed, *new)
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

// DeleteDataset removes a dataset from the store
func DeleteDataset(node *p2p.QriNode, ref *repo.DatasetRef) (err error) {
	r := node.Repo

	if err = repo.CanonicalizeDatasetRef(r, ref); err != nil {
		log.Debug(err.Error())
		return err
	}

	p, err := r.GetRef(*ref)
	if err != nil {
		log.Debug(err.Error())
		return err
	}
	if ref.Path != p.Path {
		return fmt.Errorf("given path does not equal most recent dataset path: cannot delete a specific save, can only delete entire dataset history. use `me/dataset_name` to delete entire dataset")
	}

	// ds, err := dsfs.LoadDataset(r.Store(), datastore.NewKey(ref.Path))
	// if err != nil {
	// 	return err
	// }

	if err = r.DeleteRef(*ref); err != nil {
		return err
	}

	if err = UnpinDataset(r, *ref); err != nil && err != repo.ErrNotPinner {
		return err
	}

	return r.LogEvent(repo.ETDsDeleted, *ref)
}
