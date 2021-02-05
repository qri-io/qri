package lib

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/profile"
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
		return checkRPCError(m.inst.rpc.Call("RegistryClientMethods.CreateProfile", p, ok))
	}

	ctx := context.TODO()
	// TODO(arqu): this should take the profile PK instead of active PK once multi tenancy is supported
	pro, err := m.inst.registry.CreateProfile(p, m.inst.repo.PrivateKey(ctx))
	if err != nil {
		return err
	}

	log.Debugf("create profile response: %v", pro)
	*p = *pro

	return m.updateConfig(pro)
}

// ProveProfileKey sends proof to the registry that this user has control of a
// specified private key, and modifies the user's config in order to reconcile
// it with any already existing identity the registry knows about
func (m RegistryClientMethods) ProveProfileKey(p *RegistryProfile, ok *bool) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("RegistryClientMethods.CreateProfile", p, ok))
	}

	ctx := context.Background()

	// For signing the outgoing message
	// TODO(arqu): this should take the profile PK instead of active PK once multi tenancy is supported
	privKey := m.inst.repo.PrivateKey(ctx)

	// Get public key to send to server
	pro, err := m.inst.repo.Profile(ctx)
	if err != nil {
		return err
	}
	pubkeybytes, err := pro.PubKey.Bytes()
	if err != nil {
		return err
	}

	p.ProfileID = pro.ID.String()
	p.PublicKey = base64.StdEncoding.EncodeToString(pubkeybytes)
	// TODO(dustmop): Expand the signature to sign more than just the username
	sigbytes, err := privKey.Sign([]byte(p.Username))
	p.Signature = base64.StdEncoding.EncodeToString(sigbytes)

	// Send proof to the registry
	res, err := m.inst.registry.ProveKeyForProfile(p)
	if err != nil {
		return err
	}
	log.Debugf("prove profile response: %v", res)

	// Convert the profile to a configuration, assign the registry provided profileID
	cfg := m.configFromProfile(p)
	if profileID, ok := res["profileID"]; ok {
		cfg.Profile.ID = profileID
	} else {
		return fmt.Errorf("prove: server response invalid, did not have profileID")
	}
	// TODO(dustmop): Also get logbook

	// Save the modified config
	return m.inst.ChangeConfig(cfg)
}

// Construct a config with the same values as the profile
func (m RegistryClientMethods) configFromProfile(pro *registry.Profile) *config.Config {
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
	cfg := m.configFromProfile(pro)

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

	if err := m.inst.Repo().Profiles().SetOwner(repoPro); err != nil {
		return err
	}

	return m.inst.ChangeConfig(cfg)
}
