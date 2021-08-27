package util

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

var log = golog.Logger("qriapiutil")

// APIError is an error that specifies its http status code
type APIError struct {
	Code    int
	Message string
}

// NewAPIError returns a new APIError
func NewAPIError(code int, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

// Error renders the APIError as a string
func (err *APIError) Error() string {
	return err.Message
}

// RespondWithError writes the error, with meaningful text, to the http response
func RespondWithError(w http.ResponseWriter, err error) {
	if errors.Is(err, dsref.ErrRefNotFound) || errors.Is(err, qfs.ErrNotFound) {
		WriteErrResponse(w, http.StatusNotFound, err)
		return
	}
	if errors.Is(err, repo.ErrNotFound) {
		WriteErrResponse(w, http.StatusNotFound, err)
		return
	}
	if errors.Is(err, repo.ErrNoHistory) {
		WriteErrResponse(w, http.StatusUnprocessableEntity, err)
		return
	}
	if errors.Is(err, dsref.ErrBadCaseShouldRename) || errors.Is(err, dsref.ErrDescribeValidName) || errors.Is(err, dsref.ErrDescribeValidUsername) {
		WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	var perr *dsref.ParseError
	if errors.As(err, &perr) {
		WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	var aerr *APIError
	if errors.As(err, &aerr) {
		WriteErrResponse(w, aerr.Code, err)
		return
	}
	if strings.HasPrefix(err.Error(), "invalid selection path: ") {
		// This error comes from `pathValue` in base/select.go
		WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	if strings.HasPrefix(err.Error(), "error loading dataset: error getting file bytes") {
		WriteErrResponse(w, http.StatusNotFound, err)
		return
	}
	log.Errorf("%s: treating this as a 500 is a bug, see https://github.com/qri-io/qri/issues/959. The code path that generated this should return a known error type, which this function should map to a reasonable http status code", err)
	WriteErrResponse(w, http.StatusInternalServerError, err)
	return
}

// RespondWithDispatchTypeError writes an error describing a type mismatch error from using dispatch
func RespondWithDispatchTypeError(w http.ResponseWriter, got interface{}) {
	log.Errorf("type mismatch: %v of type %s", got, reflect.TypeOf(got))
	WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("type mismatch: %v of type %s", got, reflect.TypeOf(got)))
}
