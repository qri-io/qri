package transform

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/transform/startf"
)

// TODO(dustmop): Tests. Especially once the `apply` command exists.

// Apply applies the transform script to order to modify the changing dataset
func Apply(
	ctx context.Context,
	ds *dataset.Dataset,
	loader dsref.ParseResolveLoad,
	str ioes.IOStreams,
	scriptOut io.Writer,
	secrets map[string]string,
) (err error) {
	var (
		target = ds
		head   *dataset.Dataset
	)

	if target.Transform == nil || target.Transform.ScriptFile() == nil {
		return errors.New("apply requires a transform component with a script file")
	}

	if ds.Name != "" {
		head, err = loader(ctx, fmt.Sprintf("%s/%s", ds.Peername, ds.Name))
		if errors.Is(err, dsref.ErrRefNotFound) || errors.Is(err, dsref.ErrNoHistory) {
			// Dataset either does not exist yet, or has no history. Not an error
			head = &dataset.Dataset{}
			err = nil
		} else if err != nil {
			return err
		}
	}

	// create a check func from a record of all the parts that the datasetPod is changing,
	// the startf package will use this function to ensure the same components aren't modified
	mutateCheck := startf.MutatedComponentsFunc(target)

	opts := []func(*startf.ExecOpts){
		startf.AddMutateFieldCheck(mutateCheck),
		startf.SetErrWriter(scriptOut),
		startf.SetSecrets(secrets),
		startf.AddDatasetLoader(loader),
	}

	if err = startf.ExecScript(ctx, target, head, opts...); err != nil {
		return err
	}

	str.PrintErr("✅ transform complete\n")

	return nil
}
