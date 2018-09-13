package actions

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regclient"
)

// Publish a dataset to a repo's specified registry
func Publish(node *p2p.QriNode, ref repo.DatasetRef) (err error) {
	r := node.Repo
	cli, pub, ds, err := dsParams(r, &ref)
	if err != nil {
		return err
	}
	if err = permission(r, ref); err != nil {
		return
	}

	return cli.PutDataset(ref.Peername, ref.Name, ds.Encode(), pub)
}

// Unpublish a dataset from a repo's specified registry
func Unpublish(node *p2p.QriNode, ref repo.DatasetRef) (err error) {
	r := node.Repo
	cli, pub, ds, err := dsParams(r, &ref)
	if err != nil {
		return err
	}
	if err = permission(r, ref); err != nil {
		return
	}
	return cli.DeleteDataset(ref.Peername, ref.Name, ds.Encode(), pub)
}

// Status checks to see if a dataset is published to a repo's specific registry
func Status(node *p2p.QriNode, ref repo.DatasetRef) (err error) {
	r := node.Repo
	cli, _, _, err := dsParams(r, &ref)
	if err != nil {
		return err
	}
	if err = permission(r, ref); err != nil {
		return err
	}
	if _, err := cli.GetDataset(ref.Peername, ref.Name, ref.ProfileID.String(), ref.Path); err != nil {
		return err
	}
	return nil
}

// dsParams is a convenience func that collects params for registry dataset interaction
func dsParams(r repo.Repo, ref *repo.DatasetRef) (cli *regclient.Client, pub crypto.PubKey, ds *dataset.Dataset, err error) {
	if cli = r.Registry(); cli == nil {
		err = repo.ErrNoRegistry
		return
	}

	pk := r.PrivateKey()
	if pk == nil {
		err = fmt.Errorf("repo has no configured private key")
		return
	}
	pub = pk.GetPublic()

	if err = repo.CanonicalizeDatasetRef(r, ref); err != nil {
		err = fmt.Errorf("canonicalizing dataset reference: %s", err.Error())
		return
	}

	if ref.Path == "" {
		if *ref, err = r.GetRef(*ref); err != nil {
			return
		}
	}

	ds, err = dsfs.LoadDataset(r.Store(), datastore.NewKey(ref.Path))
	return
}

// permission returns an error if a repo's configured user does not have the right
// to publish ref to a registry
func permission(r repo.Repo, ref repo.DatasetRef) (err error) {
	var pro *profile.Profile
	if pro, err = r.Profile(); err != nil {
		return err
	}
	if pro.Peername != ref.Peername {
		return fmt.Errorf("peername mismatch. '%s' doesn't have permission to publish a dataset created by '%s'", pro.Peername, ref.Peername)
	}
	return nil
}
