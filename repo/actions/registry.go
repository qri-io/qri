package actions

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regclient"
)

// Registry wraps a repo.Repo, adding actions related to working
// with registries
type Registry struct {
	repo.Repo
}

// Publish a dataset to a repo's specified registry
func (act Registry) Publish(ref repo.DatasetRef) (err error) {
	cli, pub, ds, err := act.dsParams(&ref)
	if err != nil {
		return err
	}
	if err = act.permission(ref); err != nil {
		return
	}

	return cli.PutDataset(ref.Peername, ref.Name, ds.Encode(), pub)
}

// Unpublish a dataset from a repo's specified registry
func (act Registry) Unpublish(ref repo.DatasetRef) (err error) {
	cli, pub, ds, err := act.dsParams(&ref)
	if err != nil {
		return err
	}
	if err = act.permission(ref); err != nil {
		return
	}
	return cli.DeleteDataset(ref.Peername, ref.Name, ds.Encode(), pub)
}

// Pin requests a dataset be pinned to the designated registry
func (act Registry) Pin(ref repo.DatasetRef, addrs []string) (err error) {
	cli, _, ds, err := act.dsParams(&ref)
	if err != nil {
		return err
	}
	if err = act.permission(ref); err != nil {
		return
	}

	// TODO - better test for this
	if ds.Structure.Length > 1000000*250 {
		return fmt.Errorf("dataset size exceeds 250Mb limit for pinning")
	}

	return cli.Pin(ref.Path, act.Repo.PrivateKey(), addrs)
}

// Unpin removes a pin from the remote registry
func (act Registry) Unpin(ref repo.DatasetRef, addrs []string) (err error) {
	var cli *regclient.Client
	if cli, _, _, err = act.dsParams(&ref); err != nil {
		return err
	}
	if err = act.permission(ref); err != nil {
		return
	}

	return cli.Unpin(ref.Path, act.Repo.PrivateKey())
}

// dsParams is a convenience func that collects params for registry dataset interaction
func (act Registry) dsParams(ref *repo.DatasetRef) (cli *regclient.Client, pub crypto.PubKey, ds *dataset.Dataset, err error) {
	if cli = act.Registry(); cli == nil {
		err = repo.ErrNoRegistry
		return
	}

	pk := act.PrivateKey()
	if pk == nil {
		err = fmt.Errorf("repo has no configured private key")
		return
	}
	pub = pk.GetPublic()

	if err = repo.CanonicalizeDatasetRef(act, ref); err != nil {
		err = fmt.Errorf("canonicalizing dataset reference: %s", err.Error())
		return
	}

	if ref.Path == "" {
		if *ref, err = act.GetRef(*ref); err != nil {
			return
		}
	}

	ds, err = dsfs.LoadDataset(act.Store(), datastore.NewKey(ref.Path))
	return
}

// permission returns an error if a repo's configured user does not have the right
// to publish ref to a registry
func (act Registry) permission(ref repo.DatasetRef) (err error) {
	var pro *profile.Profile
	if pro, err = act.Profile(); err != nil {
		return err
	}
	if pro.Peername != ref.Peername {
		return fmt.Errorf("peername mismatch. '%s' doesn't have permission to publish a dataset created by '%s'", pro.Peername, ref.Peername)
	}
	return nil
}
