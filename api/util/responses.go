package util

import (
	"encoding/json"
	"net/http"
)

// Response is the JSON API response object wrapper
type Response struct {
	Data     interface{}  `json:"data,omitempty"`
	Meta     *Meta        `json:"meta,omitempty"`
	NextPage *NextPageReq `json:"nextPage,omitempty"`
}

// Meta is the JSON API response meta object wrapper
type Meta struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NextPageReq is the request to get the next page of results
type NextPageReq struct {
	Method      string `json:"method"`
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	JSONBody    string `json:"jsonBody"`
}

// WriteResponse wraps response data in an envelope & writes it
func WriteResponse(w http.ResponseWriter, data interface{}) error {
	env := Response{
		Meta: &Meta{
			Code: http.StatusOK,
		},
		Data: data,
	}
	return jsonResponse(w, env)
}

// WriteResponseWithNextPageJSON writes the http response and
// includes the json call to get the next page of results
func WriteResponseWithNextPageJSON(w http.ResponseWriter, data interface{}, nextURL, nextBody string) error {
	env := Response{
		Meta: &Meta{
			Code: http.StatusOK,
		},
		NextPage: &NextPageReq{
			Method:      http.MethodPost,
			URL:         nextURL,
			ContentType: "application/json",
			JSONBody:    nextBody,
		},
		Data: data,
	}
	return jsonResponse(w, env)
}

// WriteMessageResponse includes a message with a data response
func WriteMessageResponse(w http.ResponseWriter, message string, data interface{}) error {
	env := Response{
		Meta: &Meta{
			Code:    http.StatusOK,
			Message: message,
		},
		Data: data,
	}

	return jsonResponse(w, env)
}

// WriteErrResponse writes a JSON error response message & HTTP status
func WriteErrResponse(w http.ResponseWriter, code int, err error) error {
	env := Response{
		Meta: &Meta{
			Code:  code,
			Error: err.Error(),
		},
	}

	res, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.WriteHeader(code)
	_, err = w.Write(res)
	return err
}

func jsonResponse(w http.ResponseWriter, env interface{}) error {
	res, err := json.Marshal(env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(res)
	return err
}
