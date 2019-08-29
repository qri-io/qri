package lib

import (
	"github.com/qri-io/qri/registry"
)

// RegistryClientMethods defines business logic for working with registries
type RegistryClientMethods Instance

// CoreRequestsName implements the Requests interface
func (RegistryClientMethods) CoreRequestsName() string { return "registry" }

// RegistryProfile is a user profile as stored on a registry
type RegistryProfile = registry.Profile

// CreateProfile creates a profile
func (m RegistryClientMethods) CreateProfile(p *RegistryProfile, ok *bool) (err error) {
	if m.rpc != nil {
		return m.rpc.Call("RegistryClientMethods.CreateProfile", p, ok)
	}

	pro, err := m.registry.CreateProfile(p, m.repo.PrivateKey())
	if pro != nil {
		*p = *pro
	}
	return err
}

// ProveProfileKey asserts to a registry that this user has control of a
// specified private key
func (m RegistryClientMethods) ProveProfileKey(p *RegistryProfile, ok *bool) error {
	if m.rpc != nil {
		return m.rpc.Call("RegistryClientMethods.CreateProfile", p, ok)
	}

	pro, err := m.registry.ProveProfileKey(p, m.repo.PrivateKey())
	if pro != nil {
		*p = *pro
	}
	return err
}
