package api

import (
	"github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	"io"
	"net/http"
)

func (s *Server) HandleIPFSPath(w http.ResponseWriter, r *http.Request) {
	file, err := s.qriNode.Store.Get(datastore.NewKey(r.URL.Path))
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	io.Copy(w, file)
}

// WebappHandler renders the home page
func WebappHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "webapp")
}
