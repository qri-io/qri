package apiutil

import (
	"encoding/json"
	"net/http"
)

func WriteResponse(w http.ResponseWriter, data interface{}) error {
	env := map[string]interface{}{
		"meta": map[string]interface{}{
			"code": http.StatusOK,
		},
		"data": data,
	}
	return jsonResponse(w, env)
}

func WritePageResponse(w http.ResponseWriter, data interface{}, r *http.Request, p Page) error {
	env := map[string]interface{}{
		"meta": map[string]interface{}{
			"code": http.StatusOK,
		},
		"data": data,
		"pagination": map[string]interface{}{
			"nextUrl": nextPageUrl(r, p),
		},
	}
	return jsonResponse(w, env)
}

// TODO
func nextPageUrl(r *http.Request, p Page) string {
	return r.URL.String()
}

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
	res, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(res)
	return err
}
