package lib

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/event"
	qhttp "github.com/qri-io/qri/lib/http"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/registry"
)

// RegistryClientMethods defines business logic for working with registries
type RegistryClientMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m RegistryClientMethods) Name() string {
	return "registry"
}

// Attributes defines attributes for each method
func (m RegistryClientMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"createprofile":   {Endpoint: qhttp.DenyHTTP},
		"proveprofilekey": {Endpoint: qhttp.DenyHTTP},
	}
}

// RegistryProfile is a user profile as stored on a registry
type RegistryProfile = registry.Profile

// RegistryProfileParams encapsulates arguments for creating or proving a registry profile
type RegistryProfileParams struct {
	Profile *RegistryProfile
}

// CreateProfile creates a profile
func (m RegistryClientMethods) CreateProfile(ctx context.Context, p *RegistryProfileParams) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "createprofile"), p)
	return dispatchReturnError(nil, err)
}

// ProveProfileKey sends proof to the registry that this user has control of a
// specified private key, and modifies the user's config in order to reconcile
// it with any already existing identity the registry knows about
func (m RegistryClientMethods) ProveProfileKey(ctx context.Context, p *RegistryProfileParams) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "proveprofilekey"), p)
	return dispatchReturnError(nil, err)
}

// registryImpl holds the method implementations for RegistryMethods
type registryImpl struct{}

// CreateProfile creates a profile
func (registryImpl) CreateProfile(scope scope, p *RegistryProfileParams) error {
	pro, err := scope.RegistryClient().CreateProfile(p.Profile, scope.ActiveProfile().PrivKey)
	if err != nil {
		return err
	}

	err = scope.Bus().Publish(scope.Context(), event.ETRegistryProfileCreated, event.RegistryProfileCreated{
		RegistryLocation: scope.Config().Registry.Location,
		ProfileID:        pro.ProfileID,
		Username:         pro.Username,
	})
	if err != nil {
		return err
	}

	log.Debugf("create profile response: %v", pro)
	*p.Profile = *pro

	return updateConfig(scope, pro)
}

// ProveProfileKey sends proof to the registry that this user has control of a
// specified private key, and modifies the user's config in order to reconcile
// it with any already existing identity the registry knows about
func (registryImpl) ProveProfileKey(scope scope, p *RegistryProfileParams) error {
	// Check if the repository has any saved datasets. If so, calling prove is
	// not allowed, because doing so would essentially throw away the old profile,
	// making those references unreachable. In the future, this can be changed
	// such that the old identity is given a different username, and is merged
	// into the client's collection.
	numRefs, err := scope.Repo().RefCount()
	if err != nil {
		return err
	}
	if numRefs > 0 {
		return fmt.Errorf("cannot prove with a non-empty repository")
	}

	// For signing the outgoing message
	privKey := scope.ActiveProfile().PrivKey

	// Get public key to send to server
	pro := scope.ActiveProfile()
	pubkeybytes, err := pro.PubKey.Bytes()
	if err != nil {
		return err
	}

	p.Profile.ProfileID = pro.ID.Encode()
	p.Profile.PublicKey = base64.StdEncoding.EncodeToString(pubkeybytes)
	// TODO(dustmop): Expand the signature to sign more than just the username
	sigbytes, err := privKey.Sign([]byte(p.Profile.Username))
	p.Profile.Signature = base64.StdEncoding.EncodeToString(sigbytes)

	// Send proof to the registry
	res, err := scope.RegistryClient().ProveKeyForProfile(p.Profile)
	if err != nil {
		return err
	}
	log.Debugf("prove profile response: %v", res)

	// Convert the profile to a configuration, assign the registry provided profileID
	cfg := configFromProfile(scope, p.Profile)
	if profileID, ok := res["profileID"]; ok {
		// Store the old profile as the keyID since it came from the original keypair
		cfg.Profile.KeyID = cfg.Profile.ID
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
		err = scope.Logbook().ReplaceAll(scope.Context(), lg)
		if err != nil {
			return err
		}
	} else {
		// Otherwise, nothing was ever pushed. Create new logbook data using the
		// profileID we got back.
		logbookPath := filepath.Join(scope.RepoPath(), "logbook.qfb")
		logbook, err := logbook.NewJournalOverwriteWithProfile(*pro, scope.Bus(),
			scope.Filesystem(), logbookPath)
		if err != nil {
			return err
		}
		scope.SetLogbook(logbook)
	}

	// Save the modified config
	return scope.ChangeConfig(cfg)
}

// Construct a config with the same values as the profile
func configFromProfile(scope scope, pro *registry.Profile) *config.Config {
	cfg := scope.Config().Copy()
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

func updateConfig(scope scope, pro *registry.Profile) error {
	cfg := configFromProfile(scope, pro)

	// TODO (b5) - this should be automatically done by inst.ChangeConfig
	repoPro, err := profile.NewProfile(cfg.Profile)
	if err != nil {
		return err
	}

	// TODO (b5) - this is the lowest level place I could find to monitor for
	// profile name changes, not sure this makes the most sense to have this here.
	// we should consider a separate track for any change that affects the peername,
	// it should always be verified by any set registry before saving
	if cfg.Profile.Peername != scope.ActiveProfile().Peername {
		if err := base.ModifyRepoUsername(scope.Context(), scope.Repo(), scope.Logbook(), scope.ActiveProfile().Peername, cfg.Profile.Peername); err != nil {
			return err
		}
	}

	if err := scope.Profiles().SetOwner(repoPro); err != nil {
		return err
	}

	return scope.ChangeConfig(cfg)
}
