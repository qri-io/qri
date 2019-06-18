package actions

import (
	"context"
	"fmt"

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
	node.LocalStreams.PrintErr("🗼 publishing dataset to registry\n")
	r := node.Repo
	cli, pub, ds, err := dsParams(r, &ref)
	if err != nil {
		return err
	}
	if err = permission(r, ref); err != nil {
		return
	}

	ds.Name = ref.Name
	ds.Peername = ref.Peername
	ds.Path = ref.Path
	ds.ProfileID = ref.ProfileID.String()
	preview := subset.Preview(ds)

	return cli.PutDataset(ds.Peername, ds.Name, preview, pub)
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

	ds.Name = ref.Name
	ds.Peername = ref.Peername
	ds.Path = ref.Path
	preview := subset.Preview(ds)

	return cli.DeleteDataset(ref.Peername, ref.Name, preview, pub)
}

// Pin asks a registry to host a copy of a dataset
func Pin(node *p2p.QriNode, ref repo.DatasetRef) (err error) {
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

	ng, err := newNodeGetter(node)
	if err != nil {
		return nil
	}

	node.LocalStreams.PrintErr("✈️  generating dataset manifest\n")
	mfst, err := NewManifest(node, ref.Path)
	if err != nil {
		return err
	}

	node.LocalStreams.PrintErr("🗼 syncing dataset graph to registry\n")
	if err = reg.DsyncSend(context.Background(), ng, mfst); err != nil {
		return err
	}

	node.LocalStreams.PrintErr("📌 pinning dataset\n")
	if err = reg.Pin(ref.Path, pk, nil); err != nil {
		if err == registry.ErrPinsetNotSupported {
			log.Info("this registry does not support pinning, dataset not pinned.")
		} else {
			return err
		}
	}

	node.LocalStreams.PrintErr("  done\n")
	return nil
}

// Unpin reverses the pin process
func Unpin(node *p2p.QriNode, ref repo.DatasetRef) (err error) {
	node.LocalStreams.PrintErr("📌 unpinning dataset\n")
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

// RegistryList gets a list of the published datasets of a repo's specific registry
func RegistryList(node *p2p.QriNode, limit, offset int) (datasets []*repo.DatasetRef, err error) {
	r := node.Repo
	cli := r.Registry()
	if cli == nil {
		err = repo.ErrNoRegistry
		return
	}

	regDatasets, err := cli.ListDatasets(limit, offset)
	if err != nil {
		return
	}

	for _, regDataset := range regDatasets {
		datasets = append(datasets, regToRepo(regDataset))
	}
	return
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

	ds, err = dsfs.LoadDataset(r.Store(), ref.Path)
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

func regToRepo(rds *registry.Dataset) *repo.DatasetRef {
	if rds == nil {
		return &repo.DatasetRef{}
	}

	return &repo.DatasetRef{
		Peername:  rds.Handle,
		Name:      rds.Name,
		Published: true,
		Dataset: &dataset.Dataset{
			Peername:  rds.Handle,
			Name:      rds.Name,
			Commit:    rds.Commit,
			Meta:      rds.Meta,
			Structure: rds.Structure,
			Path:      rds.Path,
		},
		Path:      rds.Path,
		ProfileID: profile.ID(rds.ProfileID),
	}
}

// RegistryDataset gets dataset info published to the registry, which is usually subset
// of the full dataset, and not suitable for hash-referencing.
func RegistryDataset(node *p2p.QriNode, ds *repo.DatasetRef) error {
	cli := node.Repo.Registry()
	if cli == nil {
		return repo.ErrNoRegistry
	}
	err := repo.CanonicalizeDatasetRef(node.Repo, ds)
	if err != nil && err != repo.ErrNotFound {
		log.Debug(err.Error())
		return err
	}
	if err == repo.ErrNotFound && node == nil {
		return fmt.Errorf("%s, and no network connection", err.Error())
	}

	dsReg, err := cli.GetDataset(ds.Peername, ds.Name, "", ds.Path)
	if err != nil {
		return err
	}

	*ds = *regToRepo(dsReg)
	return nil
}
