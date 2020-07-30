package lib

import (
	"context"
	"errors"
	"testing"

	"github.com/qri-io/qri/dsref"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestResolveReference(t *testing.T) {
	tr, err := repotest.NewTempRepo("ruh_roh", "inst_resolve_ref", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Delete()

	cfg := tr.GetConfig()
	cfg.Registry = nil
	tr.WriteConfigFile()

	ctx := context.Background()
	inst, err := NewInstance(ctx, tr.QriPath)
	if err != nil {
		t.Fatal(err)
	}

	ref := &dsref.Ref{
		Username: "example",
		Name:     "dataset",
	}
	_, err = inst.ResolveReference(ctx, ref, "")
	if !errors.Is(err, dsref.ErrRefNotFound) {
		t.Errorf(`unexpected error resolving ref with 
default resolver and no configured registry. 
want: %q
got:  %q`, dsref.ErrRefNotFound, err)
	}
}
