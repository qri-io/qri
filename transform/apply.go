package transform

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/transform/startf"
)

// TODO(dustmop): Tests. Especially once the `apply` command exists.

// Apply applies the transform script to order to modify the changing dataset
func Apply(
	ctx context.Context,
	ds *dataset.Dataset,
	r repo.Repo,
	loader dsref.ParseResolveLoad,
	pub event.Publisher,
	wait bool,
	str ioes.IOStreams,
	scriptOut io.Writer,
	secrets map[string]string,
) (string, error) {
	pro, err := r.Profile()
	if err != nil {
		return "", err
	}

	var (
		target = ds
		head   *dataset.Dataset
		runID  = startf.NewRunID()
	)

	if target.Transform == nil || target.Transform.ScriptFile() == nil {
		return runID, errors.New("apply requires a transform component with a script file")
	}

	if ds.Name != "" {
		head, err = loader(ctx, fmt.Sprintf("%s/%s", pro.Peername, ds.Name))
		if errors.Is(err, dsref.ErrRefNotFound) || errors.Is(err, dsref.ErrNoHistory) {
			// Dataset either does not exist yet, or has no history. Not an error
			head = &dataset.Dataset{}
			err = nil
		} else if err != nil {
			return runID, err
		}
	}

	// create a check func from a record of all the parts that the datasetPod is changing,
	// the startf package will use this function to ensure the same components aren't modified
	mutateCheck := startf.MutatedComponentsFunc(target)

	opts := []func(*startf.ExecOpts){
		startf.AddQriRepo(r),
		startf.AddMutateFieldCheck(mutateCheck),
		startf.SetErrWriter(scriptOut),
		startf.SetSecrets(secrets),
		startf.AddDatasetLoader(loader),
	}

	doneCh := make(chan error)

	go func() {
		if !wait {
			doneCh <- nil
		}

		err = startf.ExecScript(ctx, pub, runID, target, head, opts...)
		if err == nil {
			str.PrintErr("✅ transform complete\n")
		}
		doneCh <- err
	}()

	err = <-doneCh
	return runID, err
}
