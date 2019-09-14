package actions

import (
	"context"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/startf"
)

func mutatedComponentsFunc(dsp *dataset.Dataset) func(path ...string) error {
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
	if dsp.Viz != nil {
		components["viz"] = []string{}
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
func ExecTransform(ctx context.Context, node *p2p.QriNode, ds, prev *dataset.Dataset, scriptOut io.Writer, mutateCheck func(...string) error) error {
	if ds.Transform == nil {
		return fmt.Errorf("no transform provided")
	}

	secrets := ds.Transform.Secrets
	setSecrets := func(o *startf.ExecOpts) {
		if secrets != nil {
			// convert to map[string]interface{}, which the lower-level startf supports
			// until we're sure map[string]string is going to work in the majority of use cases
			s := map[string]interface{}{}
			for key, val := range secrets {
				s[key] = val
			}
			o.Secrets = s
		}
	}

	configs := []func(*startf.ExecOpts){
		startf.AddQriNodeOpt(node),
		startf.AddMutateFieldCheck(mutateCheck),
		startf.SetOutWriter(scriptOut),
		setSecrets,
	}

	if err := startf.ExecScript(ctx, ds, prev, configs...); err != nil {
		return err
	}

	return nil
}
