package actions

import (
	"fmt"
	"os"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/skytf"
)

// ExecTransform executes a designated transformation
func (act Dataset) ExecTransform(ds *dataset.Dataset, filepath string) (file cafs.File, err error) {
	rr, err := skytf.ExecFile(ds, filepath)
	if err != nil {
		return nil, err
	}

	st := &dataset.Structure{
		Format: dataset.JSONDataFormat,
		Schema: ds.Structure.Schema,
	}

	buf, err := dsio.NewEntryBuffer(st)
	if err != nil {
		return nil, fmt.Errorf("error allocating result buffer: %s", err)
	}

	for {
		val, err := rr.ReadEntry()
		if err != nil {
			if err.Error() == "EOF" {
				err = nil
				break
			}
			return nil, fmt.Errorf("row iteration error: %s", err.Error())
		}
		if err := buf.WriteEntry(val); err != nil {
			return nil, fmt.Errorf("error writing value to buffer: %s", err.Error())
		}
	}

	if err := buf.Close(); err != nil {
		return nil, fmt.Errorf("error closing row buffer: %s", err.Error())
	}

	// Add skylark file to store
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	// TODO - currently this is just pinning to the repo, this should *not* be pinned,
	// instead it should be added to the dataset merkleDAG
	tfPath, err := act.Repo.Store().Put(cafs.NewMemfileReader("transform.sky", f), true)
	if err != nil {
		return nil, err
	}

	// TODO - clean up events to handle this situation
	ref := repo.DatasetRef{
		Dataset: &dataset.DatasetPod{
			Transform: &dataset.TransformPod{
				Syntax: "skylark",
				Data:   tfPath.String(),
			},
		},
	}

	if err = act.LogEvent(repo.ETTransformExecuted, ref); err != nil {
		return
	}

	fmt.Println(string(buf.Bytes()))

	ds.Structure = st
	ds.Transform.Data = tfPath.String()
	return cafs.NewMemfileBytes(fmt.Sprintf("data.%s", st.Format.String()), buf.Bytes()), nil
}
