package util

import (
	"encoding/json"
	"net/http"
)

// WriteResponse wraps response data in an envelope & writes it
func WriteResponse(w http.ResponseWriter, data interface{}) error {
	env := map[string]interface{}{
		"meta": map[string]interface{}{
			"code": http.StatusOK,
		},
		"data": data,
	}
	return jsonResponse(w, env)
}

// WritePageResponse wraps response data and pagination information in an
// envelope and writes it
func WritePageResponse(w http.ResponseWriter, data interface{}, r *http.Request, p Page) error {
	if p.PrevPageExists() {
		p.PrevURL = p.Prev().SetQueryParams(r.URL).String()
	}
	if p.NextPageExists() {
		p.NextURL = p.Next().SetQueryParams(r.URL).String()
	}

	env := map[string]interface{}{
		"meta": map[string]interface{}{
			"code": http.StatusOK,
		},
		"data":       data,
		"pagination": p,
	}
	return jsonResponse(w, env)
}

// WriteMessageResponse includes a message with a data response
func WriteMessageResponse(w http.ResponseWriter, message string, data interface{}) error {
	env := map[string]interface{}{
		"meta": map[string]interface{}{
			"code":    http.StatusOK,
			"message": message,
		},
		"data": data,
	}

	return jsonResponse(w, env)
}

// WriteErrResponse writes a JSON error response message & HTTP status
func WriteErrResponse(w http.ResponseWriter, code int, err error) error {
	env := map[string]interface{}{
		"meta": map[string]interface{}{
			"code":  code,
			"error": err.Error(),
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
