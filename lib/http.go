package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	apiutil "github.com/qri-io/qri/api/util"
	qhttp "github.com/qri-io/qri/lib/http"
)

// NewHTTPRequestHandler creates a JSON-API endpoint for a registered dispatch
// method
func NewHTTPRequestHandler(inst *Instance, libMethod string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			apiutil.WriteErrResponse(w, http.StatusNotFound, fmt.Errorf("%s only accepts http POST requests", libMethod))
			return
		}

		p := inst.NewInputParam(libMethod)
		if p == nil {
			log.Debugw("http request: input params returned nil", "libMethod", libMethod)
			apiutil.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("no params for method %s", libMethod))
			return
		}

		if err := DecodeParams(r, p); err != nil {
			log.Debugw("decode params:", "err", err)
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		source := SourceFromRequest(r)
		res, cursor, err := inst.WithSource(source).Dispatch(r.Context(), libMethod, p)
		if err != nil {
			log.Debugw("http request: dispatch", "err", err)
			apiutil.RespondWithError(w, err)
			return
		}

		if cursor != nil {
			apiutil.WritePageResponse(w, res, r, apiutil.PageFromRequest(r))
			return
		}

		apiutil.WriteResponse(w, res)
	}
}

// SourceFromRequest retrieves from the http request the source for resolving refs
func SourceFromRequest(r *http.Request) string {
	return r.Header.Get(qhttp.SourceResolver)
}

// DecodeParams decodes a json body into params
func DecodeParams(r *http.Request, p interface{}) error {
	defer func() {
		if defSetter, ok := p.(NZDefaultSetter); ok {
			defSetter.SetNonZeroDefaults()
		}
	}()

	body, err := snoop(&r.Body)
	if err != nil && err != io.EOF {
		return fmt.Errorf("unable to read request body: %w", err)
	}

	if err != io.EOF {
		if err := json.NewDecoder(body).Decode(p); err != nil {
			return fmt.Errorf("unable to decode params from request body: %w", err)
		}
	}
	// allow empty params
	return nil
}

// snoop reads from an io.ReadCloser and restores it so it can be read again
func snoop(body *io.ReadCloser) (io.ReadCloser, error) {
	if body != nil && *body != nil {
		result, err := ioutil.ReadAll(*body)
		(*body).Close()

		if err != nil {
			return nil, err
		}
		if len(result) == 0 {
			return nil, io.EOF
		}

		*body = ioutil.NopCloser(bytes.NewReader(result))
		return ioutil.NopCloser(bytes.NewReader(result)), nil
	}
	return nil, io.EOF
}
