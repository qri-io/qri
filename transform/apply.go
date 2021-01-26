package transform

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/transform/startf"
)

var log = golog.Logger("transform")

// Apply applies the transform script to order to modify the changing dataset
func Apply(
	ctx context.Context,
	ds *dataset.Dataset,
	loader dsref.ParseResolveLoad,
	runID string,
	pub event.Publisher,
	wait bool,
	str ioes.IOStreams,
	scriptOut io.Writer,
	secrets map[string]string,
) error {
	var (
		target = ds
		head   *dataset.Dataset
		err    error
	)

	if target.Transform == nil || target.Transform.ScriptFile() == nil {
		return errors.New("apply requires a transform component with a script file")
	}
	if runID == "" {
		return errors.New("apply requires a runID")
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

	pub.PublishID(ctx, event.ETTransformStart, runID, "")

	if err = startf.ExecScript(ctx, target, head, opts...); err != nil {
		pub.PublishID(ctx, event.ETTransformFailure, runID, "")
		return err
	}

	pub.PublishID(ctx, event.ETTransformComplete, runID, "")
	return nil
}

// NewRunID creates a run identifier
func NewRunID() string {
	return uuid.New().String()
}
