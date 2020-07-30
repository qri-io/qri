package dsref

import (
	"context"
	"errors"
)

var (
	// ErrRefNotFound must be returned by a ref resolver that cannot resolve a
	// given reference
	ErrRefNotFound = errors.New("reference not found")
	// ErrPathRequired should be returned by functions that require a reference
	// have a path value, but got none.
	// ErrPathRequired should *not* be returned by implentations of the
	// ResolveRef interface
	ErrPathRequired = errors.New("reference path value is required")
)

// Resolver finds the identifier and HEAD path for a dataset reference
type Resolver interface {
	// ResolveRef uses ref as an outParam, setting ref.ID and ref.Path on success
	// some implementations of name resolution may make network calls
	// the returned "source" value should be either an empty string, indicating
	// the source was resolved locally, or the multiaddress of the network
	// address that performed the resolution
	ResolveRef(ctx context.Context, ref *Ref) (source string, err error)
}

// ParallelResolver composes multiple resolvers into one resolver that runs
// in parallel when called, using the first resolver that doesn't return
// ErrRefNotFound
func ParallelResolver(resolvers ...Resolver) Resolver {
	return parallelResolver(resolvers)
}

type parallelResolver []Resolver

func (rs parallelResolver) ResolveRef(ctx context.Context, ref *Ref) (string, error) {
	responses := make(chan struct {
		Ref    Ref
		Source string
	})
	errs := make(chan error)

	run := func(ctx context.Context, r Resolver) {
		if r == nil {
			errs <- ErrRefNotFound
			return
		}

		cpy := ref.Copy()
		source, err := r.ResolveRef(ctx, &cpy)
		if err != nil {
			errs <- err
			return
		}

		responses <- struct {
			Ref    Ref
			Source string
		}{cpy, source}
	}

	attempts := len(rs)
	for _, r := range rs {
		go run(ctx, r)
	}

	for {
		select {
		case res := <-responses:
			*ref = res.Ref
			return res.Source, nil
		case err := <-errs:
			attempts--
			if !errors.Is(err, ErrRefNotFound) {
				return "", err
			} else if attempts == 0 {
				return "", ErrRefNotFound
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// SequentialResolver composes multiple resolvers into one that runs each
// resolver in sequence, using the first resolver that doesn't return
// ErrorNotFound
func SequentialResolver(resolvers ...Resolver) Resolver {
	return sequentialResolver(resolvers)
}

type sequentialResolver []Resolver

func (sr sequentialResolver) ResolveRef(ctx context.Context, ref *Ref) (string, error) {
	for _, resolver := range sr {
		if resolver == nil {
			continue
		}

		resolvedSource, err := resolver.ResolveRef(ctx, ref)
		if err != nil {
			if errors.Is(err, ErrRefNotFound) {
				continue
			} else {
				return "", err
			}
		}
		return resolvedSource, nil
	}

	return "", ErrRefNotFound
}
