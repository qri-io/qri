package apiutil

import (
	"net/http"
	"strconv"
)

func ReqParamInt(key string, r *http.Request) (int, error) {
	i, err := strconv.ParseInt(r.FormValue(key), 10, 0)
	return int(i), err
}

func ReqParamBool(key string, r *http.Request) (bool, error) {
	return strconv.ParseBool(r.FormValue(key))
}
