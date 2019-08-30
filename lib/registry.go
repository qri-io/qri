package lib

import (
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/registry"
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

	*p = *pro

	cfg := m.configChanges(pro)
	return m.inst.ChangeConfig(cfg)
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

	*p = *pro

	cfg := m.configChanges(pro)
	return m.inst.ChangeConfig(cfg)
}

func (m RegistryClientMethods) configChanges(pro *registry.Profile) *config.Config {
	cfg := m.inst.cfg.Copy()
	cfg.Profile.Peername = pro.Username
	// TODO (b5) - *all* profile details should come back from registry server
	// set them here
	return cfg
}
