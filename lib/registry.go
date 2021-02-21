package lib

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
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

	// TODO(arqu): this should take the profile PK instead of active PK once multi tenancy is supported
	ownerPk := m.inst.repo.Profiles().Owner().PrivKey
	pro, err := m.inst.registry.CreateProfile(p, ownerPk)
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

	// Check if the repository has any saved datasets. If so, calling prove is
	// not allowed, because doing so would essentially throw away the old profile,
	// making those references unreachable. In the future, this can be changed
	// such that the old identity is given a different username, and is merged
	// into the client's collection.
	numRefs, err := m.inst.repo.RefCount()
	if err != nil {
		return err
	}
	if numRefs > 0 {
		return fmt.Errorf("cannot prove with a non-empty repository")
	}

	// For signing the outgoing message
	// TODO(arqu): this should take the profile PK instead of active PK once multi tenancy is supported
	privKey := m.inst.repo.Profiles().Owner().PrivKey

	// Get public key to send to server
	pro := m.inst.repo.Profiles().Owner()
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
		if pid := profile.IDB58DecodeOrEmpty(profileID); pid != "" {
			// Assign to profile struct as well
			pro.ID = pid
		}
	} else {
		return fmt.Errorf("prove: server response invalid, did not have profileID")
	}

	// If an existing user of this profileID pushed something, get the previous logbook data
	// and write it to our repository. This enables push and pull to continue to work
	if logbookBin, ok := res["logbookInit"]; ok && len(logbookBin) > 0 {
		logbookBytes, err := base64.StdEncoding.DecodeString(logbookBin)
		if err != nil {
			return err
		}
		lg := &oplog.Log{}
		if err := lg.UnmarshalFlatbufferBytes(logbookBytes); err != nil {
			return err
		}
		err = m.inst.repo.Logbook().ReplaceAll(ctx, lg)
		if err != nil {
			return err
		}
	} else {
		// Otherwise, nothing was ever pushed. Create new logbook data using the
		// profileID we got back.
		logbookPath := filepath.Join(m.inst.repoPath, "logbook.qfb")
		logbook, err := logbook.NewJournalOverwriteWithProfileID(privKey, p.Username, m.inst.bus,
			m.inst.qfs, logbookPath, cfg.Profile.ID)
		if err != nil {
			return err
		}
		m.inst.logbook = logbook
	}

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
