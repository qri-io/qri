package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
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
func DatasetRefFromReq(r *http.Request) (reporef.DatasetRef, error) {
	if r.URL.String() == "" || r.URL.Path == "" {
		return reporef.DatasetRef{}, nil
	}
	return DatasetRefFromPath(r.URL.Path)
}

// DatasetRefFromPath parses a path and returns a datasetRef
func DatasetRefFromPath(path string) (reporef.DatasetRef, error) {
	refstr := HTTPPathToQriPath(path)
	return repo.ParseDatasetRef(refstr)
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

// HTTPPathToQriPath converts a http path to a
// qri path
func HTTPPathToQriPath(path string) string {
	paramIndex := strings.Index(path, "?")
	if paramIndex != -1 {
		path = path[:paramIndex]
	}
	path = strings.Replace(path, "/at", "@", 1)
	path = strings.TrimPrefix(path, "/")
	return path
}
