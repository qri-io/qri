package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/lib"
	qhttp "github.com/qri-io/qri/lib/http"
)

const (
	// AEToken is the token provider endpoint
	AEToken qhttp.APIEndpoint = "/oauth/token"
)

// TokenHandler is a handler to authenticate and generate access & refresh tokens
func TokenHandler(inst *lib.Instance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenRequest, err := parseTokenRequest(r)
		if err != nil {
			log.Debugf("tokenHandler failed to parse request: %q", err.Error())
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		tokenResponse, err := inst.TokenProvider().Token(r.Context(), tokenRequest)
		if err != nil {
			log.Debugf("tokenHandler failed to create token: %q", err.Error())
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		util.WriteResponse(w, tokenResponse)
	}
}

// parseTokenRequest extracts the token.Request from the incoming http request
func parseTokenRequest(r *http.Request) (*token.Request, error) {
	tr := &token.Request{}
	if r.Header.Get("Content-Type") == "application/json" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Errorf("parseTokenRequest failed to parse request: %s", err.Error())
			return nil, token.ErrInvalidRequest
		}
		err = json.Unmarshal(body, tr)
		if err != nil {
			log.Errorf("parseTokenRequest failed to parse request: %s", err.Error())
			return nil, token.ErrInvalidRequest
		}
	}
	if tr.GrantType.String() == "" {
		tr.GrantType = token.GrantType(r.FormValue("grant_type"))
	}
	if tr.Code == "" {
		tr.Code = r.FormValue("code")
	}
	if tr.RedirectURI == "" {
		tr.RedirectURI = r.FormValue("redirect_uri")
	}
	if tr.Username == "" {
		tr.Username = r.FormValue("username")
	}
	if tr.Password == "" {
		tr.Password = r.FormValue("password")
	}
	if tr.RefreshToken == "" {
		tr.RefreshToken = r.FormValue("refresh_token")
	}
	return tr, nil
}
