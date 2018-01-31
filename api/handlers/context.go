package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/qri-io/qri/repo"
)

// QriCtxKey defines a distinct type for
// keys for context values should always use custom
// types to avoid collissions.
// see comment on context.WithValue for more info
type QriCtxKey string

// DatasetRefCtxKey is the key for adding a dataset reference
// to a context.Context
const DatasetRefCtxKey QriCtxKey = "datasetRef"

// DatasetRefFromReq examines the path element of a request URL
// to
func DatasetRefFromReq(r *http.Request) (*repo.DatasetRef, error) {
	if r.URL.String() == "" || r.URL.Path == "" {
		return nil, nil
	}
	refstr := strings.Replace(r.URL.Path, "/at/ipfs/", "@", 1)
	refstr = strings.Replace(refstr, "/at/", "@", 1)
	refstr = strings.TrimPrefix(refstr, "/")
	refstr = strings.TrimSuffix(refstr, "/")
	ref, err := repo.ParseDatasetRef(refstr)
	if err != nil {
		ref = nil
	}
	return ref, err
}

// DatasetRefFromCtx extracts a Dataset reference from a given
// context if one is set, returning nil otherwise
func DatasetRefFromCtx(ctx context.Context) *repo.DatasetRef {
	iface := ctx.Value(DatasetRefCtxKey)
	if ref, ok := iface.(*repo.DatasetRef); ok {
		return ref
	}
	return nil
}
