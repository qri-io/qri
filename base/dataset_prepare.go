package base

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	datastore "github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/varName"
)

// PrepareDatasetNew processes dataset input into it's necessary components for creation
func PrepareDatasetNew(dsp *dataset.DatasetPod) (ds *dataset.Dataset, body cafs.File, secrets map[string]string, err error) {
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
	if body, err = DatasetPodBodyFile(dsp); err == nil {
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

	} else if err.Error() == "not found" {
		err = nil
	}

	return
}

// PrepareDatasetSave prepares a set of changes for submission to SaveDataset
func PrepareDatasetSave(r repo.Repo, dsp *dataset.DatasetPod) (ds *dataset.Dataset, body cafs.File, secrets map[string]string, err error) {
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
	if err = repo.CanonicalizeDatasetRef(r, prev); err != nil {
		err = fmt.Errorf("error with previous reference: %s", err.Error())
		return
	}

	if err = ReadDataset(r, prev); err != nil {
		err = fmt.Errorf("error getting previous dataset: %s", err.Error())
		return
	}

	if dsp.BodyBytes != nil || dsp.BodyPath != "" {
		if body, err = DatasetPodBodyFile(dsp); err != nil {
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
		tfScript, e := r.Store().Get(datastore.NewKey(ds.Transform.ScriptPath))
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
		vizScript, e := r.Store().Get(datastore.NewKey(ds.Viz.ScriptPath))
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
