package actions

import (
	"os"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/skytf"
)

// ExecTransform executes a designated transformation
func ExecTransform(node *p2p.QriNode, ds *dataset.Dataset, infile cafs.File, secrets map[string]string) (file cafs.File, err error) {
	filepath := ds.Transform.ScriptPath
	file, err = skytf.ExecFile(ds, filepath, infile, skytf.AddQriNodeOpt(node), func(o *skytf.ExecOpts) {
		if secrets != nil {
			// convert to map[string]interface{}, which the lower-level skytf supports
			// until we're sure map[string]string is going to work in the majority of use cases
			s := map[string]interface{}{}
			for key, val := range secrets {
				s[key] = val
			}
			o.Secrets = s
		}
	})
	if err != nil {
		return nil, err
	}

	// TODO - adding here just to get the content-addressed script path for the event.
	// clean up events to handle this situation
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	tfPath, err := node.Repo.Store().Put(cafs.NewMemfileReader("transform.sky", f), false)
	if err != nil {
		return nil, err
	}
	ref := repo.DatasetRef{
		Dataset: &dataset.DatasetPod{
			Transform: &dataset.TransformPod{
				Syntax:     "skylark",
				ScriptPath: tfPath.String(),
			},
		},
	}

	if err = node.Repo.LogEvent(repo.ETTransformExecuted, ref); err != nil {
		return
	}

	// ds.Structure = st
	// return cafs.NewMemfileBytes(fmt.Sprintf("data.%s", st.Format.String()), buf.Bytes()), nil
	return
}
