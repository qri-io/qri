package actions

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/subset"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry"
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

	enc := ds.Encode()
	enc.Name = ref.Name
	enc.Peername = ref.Peername
	enc.Path = ref.Path
	preview := subset.Preview(enc)

	return cli.PutDataset(ref.Peername, ref.Name, preview, pub)
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

	enc := ds.Encode()
	enc.Name = ref.Name
	enc.Peername = ref.Peername
	enc.Path = ref.Path
	preview := subset.Preview(enc)

	return cli.DeleteDataset(ref.Peername, ref.Name, preview, pub)
}

// Status checks to see if a dataset is published to a repo's specific registry
func Status(node *p2p.QriNode, ref repo.DatasetRef) (err error) {
	r := node.Repo
	cli, _, _, err := dsParams(r, &ref)
	if err != nil {
		return err
	}
	if _, err := cli.GetDataset(ref.Peername, ref.Name, ref.ProfileID.String(), ref.Path); err != nil {
		return err
	}
	return nil
}

// Pin asks a registry to host a copy of a dataset
func Pin(node *p2p.QriNode, ref repo.DatasetRef) (err error) {
	node.LocalStreams.Print("📌 pinning dataset")
	r := node.Repo
	reg := node.Repo.Registry()
	if reg == nil {
		return fmt.Errorf("no registry specified")
	}

	pk := r.PrivateKey()
	if pk == nil {
		err = fmt.Errorf("repo has no configured private key")
		return
	}

	if err = repo.CanonicalizeDatasetRef(r, &ref); err != nil {
		err = fmt.Errorf("canonicalizing dataset reference: %s", err.Error())
		return
	}

	if ref.Path == "" {
		if ref, err = r.GetRef(ref); err != nil {
			return
		}
	}

	if !node.Online {
		if err = node.GoOnline(); err != nil {
			return err
		}
	}

	var addrs []string
	for _, maddr := range node.EncapsulatedAddresses() {
		addrs = append(addrs, maddr.String())
	}

	if err = reg.Pin(ref.Path, pk, addrs); err != nil {
		if err == registry.ErrPinsetNotSupported {
			log.Info("this registry does not support pinning, dataset not pinned.")
		} else {
			return err
		}
	} else {
		log.Info("done")
	}

	return nil
}

// Unpin reverses the pin process
func Unpin(node *p2p.QriNode, ref repo.DatasetRef) (err error) {
	node.LocalStreams.Print("📌 unpinning dataset")
	r := node.Repo
	reg := node.Repo.Registry()
	if reg == nil {
		return fmt.Errorf("no registry specified")
	}

	pk := r.PrivateKey()
	if pk == nil {
		err = fmt.Errorf("repo has no configured private key")
		return
	}

	if err = repo.CanonicalizeDatasetRef(r, &ref); err != nil {
		err = fmt.Errorf("canonicalizing dataset reference: %s", err.Error())
		return
	}

	if ref.Path == "" {
		if ref, err = r.GetRef(ref); err != nil {
			return
		}
	}

	return reg.Unpin(ref.Path, pk)
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
		return fmt.Errorf("'%s' doesn't have permission to publish a dataset created by '%s'", pro.Peername, ref.Peername)
	}
	return nil
}
