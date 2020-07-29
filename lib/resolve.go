package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/remote"
)

// ParseAndResolveRef combines reference parsing and resolution
func (inst *Instance) ParseAndResolveRef(ctx context.Context, refStr, source string) (dsref.Ref, string, error) {
	log.Debugf("inst.ParseAndResolveRef refStr=%q source=%q", refStr, source)
	ref, err := dsref.Parse(refStr)
	if err != nil {
		return ref, "", fmt.Errorf("%q is not a valid dataset reference: %w", refStr, err)
	}

	resolvedSource, err := inst.ResolveReference(ctx, &ref, source)
	if err != nil {
		return ref, resolvedSource, err
	}
	return ref, resolvedSource, err
}

// ParseAndResolveRefWithWorkingDir combines reference parsing and resolution,
// including setting default Path to a linked working directory if one exists
func (inst *Instance) ParseAndResolveRefWithWorkingDir(ctx context.Context, refStr, source string) (dsref.Ref, string, error) {
	ref, err := dsref.Parse(refStr)
	if err != nil && err != dsref.ErrBadCaseName {
		return ref, "", fmt.Errorf("%q is not a valid dataset reference: %w", refStr, err)
	}

	pathProvided := ref.Path != ""
	resolvedSource, err := inst.ResolveReference(ctx, &ref, source)
	if err != nil {
		return ref, resolvedSource, err
	}
	if !pathProvided {
		err = inst.fsi.ResolvedPath(&ref)
		if err == fsi.ErrNoLink {
			err = nil
		}
	}
	return ref, resolvedSource, err
}

// ResolveReference finds the identifier & HEAD path for a dataset reference.
// the mode parameter determines which subsystems of Qri to use when resolving
func (inst *Instance) ResolveReference(ctx context.Context, ref *dsref.Ref, mode string) (string, error) {
	log.Debugf("inst.ResolveReference ref=%q mode=%q", ref, mode)
	if inst == nil {
		return "", dsref.ErrRefNotFound
	}

	// Handle the "me" convenience shortcut
	if ref.Username == "me" {
		ref.Username = inst.cfg.Profile.Peername
	}

	resolver, err := inst.resolverForMode(mode)
	if err != nil {
		log.Debug("inst.resolverForMode error=%q", err)
		return "", err
	}

	return resolver.ResolveRef(ctx, ref)
}

func (inst *Instance) resolverForMode(mode string) (dsref.Resolver, error) {
	switch mode {
	case "":
		return inst.defaultResolver(), nil
	case "local":
		return dsref.SequentialResolver(
			inst.dscache,
			inst.repo,
		), nil
	case "network":
		// TODO(b5) - one day use registry & p2p in parallel here
		return inst.registryResolver(), nil
	case "registry":
		return inst.registryResolver(), nil
	case "p2p":
		return nil, fmt.Errorf("p2p network cannot be used to resolve references")
	}

	// TODO (b5) - mode could be one of:
	// * configured remote name
	// * peername
	// * peer multiaddress
	// add support for peername & multiaddress resolution
	addr, err := remote.Address(inst.Config(), mode)
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
	return inst.remoteClient.NewRemoteRefResolver(inst.cfg.Registry.Location)
}
