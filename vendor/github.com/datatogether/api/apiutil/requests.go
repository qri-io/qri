package apiutil

import (
	"net/http"
	"strconv"
)

// TODO - consider providing a default param & removing the error
func ReqParamInt(key string, r *http.Request) (int, error) {
	i, err := strconv.ParseInt(r.FormValue(key), 10, 0)
	return int(i), err
}

// TODO - consider providing a default param & removing the error
func ReqParamBool(key string, r *http.Request) (bool, error) {
	return strconv.ParseBool(r.FormValue(key))
}
