package api

import (
	"context"
	"net/http"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	reporef "github.com/qri-io/qri/repo/ref"
)

// TODO(arqu): this needs to be reworked once are done with the
// RPC > HTTP transition and should move from query params to context params

// QriCtxKey defines a distinct type for
// keys for context values should always use custom
// types to avoid collisions.
// see comment on context.WithValue for more info
type QriCtxKey string

// DatasetRefCtxKey is the key for adding a dataset reference
// to a context.Context
const DatasetRefCtxKey QriCtxKey = "datasetRef"

// DatasetRefFromReq examines the path element of a request URL
// to
func DatasetRefFromReq(r *http.Request) (dsref.Ref, error) {
	if r.URL.String() == "" || r.URL.Path == "" {
		return dsref.Ref{}, nil
	}
	return lib.DsRefFromPath(r.URL.Path)
}

// DatasetRefFromCtx extracts a Dataset reference from a given
// context if one is set, returning nil otherwise
func DatasetRefFromCtx(ctx context.Context) reporef.DatasetRef {
	iface := ctx.Value(DatasetRefCtxKey)
	if ref, ok := iface.(reporef.DatasetRef); ok {
		return ref
	}
	return reporef.DatasetRef{}
}
