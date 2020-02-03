package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// RegistryClientMethods defines business logic for working with registries
type RegistryClientMethods struct {
	inst *Instance
}

// NewRegistryClientMethods creates client methods from an instance
func NewRegistryClientMethods(inst *Instance) *RegistryClientMethods {
	return &RegistryClientMethods{
		inst: inst,
	}
}

// CoreRequestsName implements the Requests interface
func (RegistryClientMethods) CoreRequestsName() string { return "registry" }

// RegistryProfile is a user profile as stored on a registry
type RegistryProfile = registry.Profile

// CreateProfile creates a profile
func (m RegistryClientMethods) CreateProfile(p *RegistryProfile, ok *bool) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("RegistryClientMethods.CreateProfile", p, ok)
	}

	pro, err := m.inst.registry.CreateProfile(p, m.inst.repo.PrivateKey())
	if err != nil {
		return err
	}

	log.Debugf("create profile response: %v", pro)
	*p = *pro

	return m.updateConfig(pro)
}

// ProveProfileKey asserts to a registry that this user has control of a
// specified private key
func (m RegistryClientMethods) ProveProfileKey(p *RegistryProfile, ok *bool) error {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("RegistryClientMethods.CreateProfile", p, ok)
	}

	pro, err := m.inst.registry.ProveProfileKey(p, m.inst.repo.PrivateKey())
	if err != nil {
		return err
	}

	log.Errorf("prove profile response: %v", pro)
	*p = *pro

	return m.updateConfig(pro)
}

func (m RegistryClientMethods) configChanges(pro *registry.Profile) *config.Config {
	cfg := m.inst.cfg.Copy()
	cfg.Profile.Peername = pro.Username
	cfg.Profile.Created = pro.Created
	cfg.Profile.Email = pro.Email
	cfg.Profile.Photo = pro.Photo
	cfg.Profile.Thumb = pro.Thumb
	cfg.Profile.Name = pro.Name
	cfg.Profile.Description = pro.Description
	cfg.Profile.HomeURL = pro.HomeURL
	cfg.Profile.Twitter = pro.Twitter

	return cfg
}

func (m RegistryClientMethods) updateConfig(pro *registry.Profile) error {
	ctx := context.TODO()
	cfg := m.configChanges(pro)

	// TODO (b5) - this should be automatically done by m.inst.ChangeConfig
	repoPro, err := profile.NewProfile(cfg.Profile)
	if err != nil {
		return err
	}

	// TODO (b5) - this is the lowest level place I could find to monitor for
	// profile name changes, not sure this makes the most sense to have this here.
	// we should consider a separate track for any change that affects the peername,
	// it should always be verified by any set registry before saving
	if cfg.Profile.Peername != m.inst.cfg.Profile.Peername {
		if err := base.ModifyRepoUsername(ctx, m.inst.Repo(), m.inst.logbook, m.inst.cfg.Profile.Peername, cfg.Profile.Peername); err != nil {
			return err
		}
	}

	if err := m.inst.Repo().SetProfile(repoPro); err != nil {
		return err
	}

	return m.inst.ChangeConfig(cfg)
}

// Home returns a listing of datasets from a number of feeds like featured and
// popular. Each feed is keyed by string in the response
func (m *RegistryClientMethods) Home(p *bool, res *map[string][]*dataset.Dataset) error {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("RegistryClientMethods.Home", p, res)
	}
	ctx := context.TODO()

	if m.inst.registry == nil {
		return fmt.Errorf("Feed isn't available without a configured registry")
	}

	feed, err := m.inst.registry.HomeFeed(ctx)
	if err != nil {
		return err
	}

	*res = feed
	return nil
}

// Featured asks a registry for a curated list of datasets
func (m *RegistryClientMethods) Featured(p *ListParams, res *[]*dataset.Dataset) error {
	return fmt.Errorf("featured dataset feed is not yet implemented")
}

// Recent is a feed of network datasets in reverse chronological order
// it currently can only come from a registry, but could easily be assembled
// via p2p methods
func (m *RegistryClientMethods) Recent(p *ListParams, res *[]*dataset.Dataset) error {
	return fmt.Errorf("recent dataset feed is not yet implemented")
}

// Preview creates
func (m *RegistryClientMethods) Preview(refstr *string, res *dataset.Dataset) error {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("RegistryClientMethods.Preview", refstr, res)
	}
	ctx := context.TODO()

	ref, err := repo.ParseDatasetRef(*refstr)
	if err != nil {
		return err
	}

	pre, err := m.inst.registry.Preview(ctx, repo.ConvertToDsref(ref))
	if err != nil {
		return err
	}

	*res = *pre
	return nil
}
