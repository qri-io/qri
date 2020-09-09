package util

import (
	"net/http"
	"strconv"
)

// ReqParamInt extracts an integer parameter from a request form value
func ReqParamInt(r *http.Request, key string, def int) int {
	v := r.FormValue(key)
	if v == "" {
		return def
	}
	i, err := strconv.ParseInt(v, 10, 0)
	if err != nil {
		return def
	}
	return int(i)
}

// ReqParamBool pulls a boolean parameter from a request form value
func ReqParamBool(r *http.Request, key string, def bool) bool {
	v := r.FormValue(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
