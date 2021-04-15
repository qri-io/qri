package lib

import (
	"context"

	"github.com/qri-io/qri/logbook"
)

// LogMethods extends a lib.Instance with business logic for working with lists
// of dataset versions. think "git log".
type LogMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m LogMethods) Name() string {
	return "log"
}

// Attributes defines attributes for each method
func (m LogMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"log":            {endpoint: denyRPC},
		"rawlogbook":     {endpoint: denyRPC},
		"logbooksummary": {endpoint: denyRPC},
	}
}

// RefListParams encapsulates parameters for requests to a single reference
// that will produce a paginated result
type RefListParams struct {
	// String value of a reference
	Ref string
	// Pagination Parameters
	Offset, Limit int
}

// LogEntry is a record in a log of operations on a dataset
type LogEntry = logbook.LogEntry

// Log lists log entries for actions taken on a given dataset
func (m LogMethods) Log(ctx context.Context, p *RefListParams) ([]LogEntry, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "log"), p)
	if res, ok := got.([]LogEntry); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// RawLogbookParams enapsulates parameters for the RawLogbook methods
type RawLogbookParams struct {
	// no options yet
}

// RawLogs is an alias for a human representation of a plain-old-data logbook
type RawLogs = []logbook.PlainLog

// RawLogbook encodes the full logbook as human-oriented json
func (m LogMethods) RawLogbook(ctx context.Context, p *RawLogbookParams) (*RawLogs, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "rawlogbook"), p)
	if res, ok := got.(*RawLogs); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// LogbookSummary returns a string overview of the logbook
func (m LogMethods) LogbookSummary(ctx context.Context, p *struct{}) (*string, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "logbooksummary"), p)
	if res, ok := got.(*string); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// logImpl holds the method implementations for LogMethods
type logImpl struct{}

// Entries lists log entries for actions taken on a given dataset
func (logImpl) Log(scope scope, p *RefListParams) ([]LogEntry, error) {
	res := []LogEntry{}
	var err error

	ref, _, err := scope.ParseAndResolveRef(scope.Context(), p.Ref)
	if err != nil {
		return nil, err
	}

	book := scope.Logbook()
	res, err = book.LogEntries(scope.Context(), ref, p.Offset, p.Limit)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// RawLogbook encodes the full logbook as human-oriented json
func (logImpl) RawLogbook(scope scope, p *RawLogbookParams) (*RawLogs, error) {
	res := &RawLogs{}
	var err error

	*res, err = scope.Logbook().PlainLogs(scope.Context())
	if err != nil {
		return nil, err
	}
	return res, nil
}

// LogbookSummary returns a string overview of the logbook
func (logImpl) LogbookSummary(scope scope, p *struct{}) (*string, error) {
	res := ""
	res = scope.Logbook().SummaryString(scope.Context())
	return &res, nil
}
