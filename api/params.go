package api

import (
	"net/http"

	"github.com/gorilla/schema"
	"github.com/qri-io/qri/lib"
)

// decoder maps HTTP requests to input structs
var decoder = schema.NewDecoder()

func init() {
	// TODO(arqu): once APIs have a strict mapping to Params this line
	// should be removed and should error out on unknown keys
	decoder.IgnoreUnknownKeys(true)
}

// RequestUnmarshaller is an interface for deserializing from an HTTP request
type RequestUnmarshaller interface {
	UnmarshalFromRequest(r *http.Request) error
}

// UnmarshalParams deserializes a lib req params stuct pointer from an HTTP
// request. Only used for handlers that do not use lib.NewHTTPMethodRequest.
func UnmarshalParams(r *http.Request, p interface{}) error {
	defer func() {
		if defSetter, ok := p.(lib.NZDefaultSetter); ok {
			defSetter.SetNonZeroDefaults()
		}
	}()

	if ru, ok := p.(RequestUnmarshaller); ok {
		return ru.UnmarshalFromRequest(r)
	}

	if err := r.ParseForm(); err != nil {
		return err
	}
	return decoder.Decode(p, r.Form)
}
