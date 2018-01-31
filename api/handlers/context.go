package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/qri-io/qri/repo"
)

const datasetRefCtxKey = "datasetRef"

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

func DatasetRefFromCtx(ctx context.Context) *repo.DatasetRef {
	iface := ctx.Value(datasetRefCtxKey)
	if ref, ok := iface.(*repo.DatasetRef); ok {
		return ref
	}
	return nil
}
