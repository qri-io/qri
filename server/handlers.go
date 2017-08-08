package server

import (
	"github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	"net/http"
)

func (s *Server) HandleIPFSPath(w http.ResponseWriter, r *http.Request) {
	v, err := s.store.Get(datastore.NewKey(r.URL.Path))
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	switch t := v.(type) {
	case []byte:
		w.Write(t)
		return
	}

	apiutil.WriteResponse(w, v)
}
