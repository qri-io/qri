package actions

import (
	"fmt"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/startf"
)

func mutatedComponentsFunc(dsp *dataset.DatasetPod) func(path ...string) error {
	components := map[string][]string{}
	if dsp.Transform != nil {
		components["transform"] = []string{}
	}
	if dsp.Meta != nil {
		components["meta"] = []string{}
	}
	if dsp.Structure != nil {
		components["structure"] = []string{}
	}
	if dsp.Body != nil || dsp.BodyBytes != nil || dsp.BodyPath != "" {
		components["body"] = []string{}
	}

	return func(path ...string) error {
		if len(path) > 0 && components[path[0]] != nil {
			return fmt.Errorf(`transform script and user-supplied dataset are both trying to set:
  %s

please adjust either the transform script or remove the supplied '%s'`, path[0], path[0])
		}
		return nil
	}
}

// ExecTransform executes a designated transformation
func ExecTransform(node *p2p.QriNode, ds *dataset.Dataset, script, bodyFile cafs.File, secrets map[string]string, mutateCheck func(...string) error) (file cafs.File, err error) {
	// filepath := ds.Transform.ScriptPath

	// TODO - consider making this a standard method on dataset.Transform:
	// script := cafs.NewMemfileReader(ds.Transform.ScriptPath, ds.Transform.Script)

	file, err = startf.ExecScript(ds, script, bodyFile, startf.AddQriNodeOpt(node), startf.AddMutateFieldCheck(mutateCheck), func(o *startf.ExecOpts) {
		if secrets != nil {
			// convert to map[string]interface{}, which the lower-level startf supports
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

	return
}
