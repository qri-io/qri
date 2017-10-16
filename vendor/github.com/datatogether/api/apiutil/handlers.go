package apiutil

import (
	"net/http"
)

// HealthCheckHandler is a basic "hey I'm fine" for load balancers & co
// TODO - add Database connection & proper configuration checks here for more accurate
// health reporting
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{ "meta": { "code": 200, "status": "ok" }, "data": [] }`))
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{ "meta": { "code": 404, "status": "not found" }, "data": null }`))
}

// EmptyOkHandler is an empty 200 response, often used
// for OPTIONS requests that responds with headers set in addCorsHeaders
func EmptyOkHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
