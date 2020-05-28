package base

import (
	"context"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
	"github.com/qri-io/qri/startf"
)

// TODO(dustmop): Tests. Especially once the `apply` command exists.

// TransformApply applies the transform script to order to modify the changing dataset
func TransformApply(ctx context.Context, ds *dataset.Dataset, r repo.Repo, str ioes.IOStreams, scriptOut io.Writer, secrets map[string]string) error {
	pro, err := r.Profile()
	if err != nil {
		return err
	}

	headPath := ""
	if ds.Name != "" {
		// Lookup the dataset to retrieve the head version
		lookup := &reporef.DatasetRef{Peername: pro.Peername, Name: ds.Name}
		err = repo.CanonicalizeDatasetRef(r, lookup)
		if err == repo.ErrNotFound || err == repo.ErrNoHistory {
			// Dataset either does not exist yet, or has no history. Not an error.
		} else if err != nil {
			return err
		}
		headPath = lookup.Path
	}

	target := ds
	head := &dataset.Dataset{}

	if headPath != "" {
		// Load the dataset's most recent version
		if head, err = dsfs.LoadDataset(ctx, r.Store(), headPath); err != nil {
			return nil
		}
		if head.BodyPath != "" {
			var body qfs.File
			body, err = dsfs.LoadBody(ctx, r.Store(), head)
			if err != nil {
				return err
			}
			head.SetBodyFile(body)
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
	}

	if err = startf.ExecScript(ctx, target, head, opts...); err != nil {
		return err
	}

	str.PrintErr("✅ transform complete\n")

	return nil
}
