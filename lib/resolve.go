package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/remote"
)

// ParseAndResolveRef combines reference parsing and resolution
func (inst *Instance) ParseAndResolveRef(ctx context.Context, refStr, source string, useFSI bool) (dsref.Ref, string, error) {
	log.Debugf("inst.ParseAndResolveRef refStr=%q source=%q", refStr, source)
	ref, err := dsref.Parse(refStr)
	if err != nil {
		return ref, "", fmt.Errorf("%q is not a valid dataset reference: %w", refStr, err)
	}

	// Whether the reference came with an explicit version
	pathProvided := ref.Path != ""
	// Resolve the reference
	location, err := inst.ResolveReference(ctx, &ref, source)
	if err != nil {
		return ref, location, err
	}
	// If no version was given, and FSI is enabled for resolution, look
	// up if the dataset has a version on disk.
	if !pathProvided && useFSI {
		err = inst.fsi.ResolvedPath(&ref)
		if err == fsi.ErrNoLink {
			err = nil
		}
	}
	return ref, location, err
}

// ResolveReference finds the identifier & HEAD path for a dataset reference.
// the source parameter determines which subsystems of Qri to use when resolving
func (inst *Instance) ResolveReference(ctx context.Context, ref *dsref.Ref, source string) (string, error) {
	log.Debugf("inst.ResolveReference ref=%q source=%q", ref, source)
	if inst == nil {
		return "", dsref.ErrRefNotFound
	}

	// Handle the "me" convenience shortcut
	if ref.Username == "me" {
		ref.Username = inst.cfg.Profile.Peername
	}

	resolver, err := inst.resolverForSource(source)
	if err != nil {
		log.Debug("inst.resolverForSource error=%q", err)
		return "", err
	}

	return resolver.ResolveRef(ctx, ref)
}

func (inst *Instance) resolverForSource(source string) (dsref.Resolver, error) {
	switch source {
	case "":
		return inst.defaultResolver(), nil
	case "local":
		return dsref.SequentialResolver(
			inst.dscache,
			inst.repo,
		), nil
	case "network":
		return dsref.ParallelResolver(
			inst.registryResolver(),
			inst.p2pResolver(),
		), nil
	case "registry":
		return inst.registryResolver(), nil
	case "p2p":
		return inst.p2pResolver(), nil
	}

	// TODO (b5) - source could be one of:
	// * configured remote name
	// * peername
	// * peer multiaddress
	// add support for peername & multiaddress resolution
	addr, err := remote.Address(inst.GetConfig(), source)
	if err != nil {
		return nil, err
	}
	return inst.remoteClient.NewRemoteRefResolver(addr), nil
}

func (inst *Instance) defaultResolver() dsref.Resolver {
	return dsref.SequentialResolver(
		inst.dscache,
		inst.repo,
		dsref.ParallelResolver(
			inst.registryResolver(),
			// inst.node,
		),
	)
}

func (inst *Instance) registryResolver() dsref.Resolver {
	var location string
	if inst.cfg.Registry != nil {
		location = inst.cfg.Registry.Location
	}
	return inst.remoteClient.NewRemoteRefResolver(location)
}

func (inst *Instance) p2pResolver() dsref.Resolver {
	return inst.node.NewP2PRefResolver()
}
