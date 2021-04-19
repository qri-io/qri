package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
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
		"history":        {endpoint: AEHistory, httpVerb: "POST"},
		"log":            {endpoint: AELog, httpVerb: "POST"},
		"rawlogbook":     {endpoint: denyRPC},
		"logbooksummary": {endpoint: denyRPC},
	}
}

// HistoryParams defines parameters for the Log method
type HistoryParams struct {
	ListParams
	// Reference to data to fetch history for
	Ref  string
	Pull bool
}

// History returns the history of changes for a given dataset
func (m LogMethods) History(ctx context.Context, params *HistoryParams) ([]dsref.VersionInfo, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "history"), params)
	if res, ok := got.([]dsref.VersionInfo); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
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

// History returns the history of changes for a given dataset
func (logImpl) History(scope scope, params *HistoryParams) ([]dsref.VersionInfo, error) {
	// ensure valid limit value
	if params.Limit <= 0 {
		params.Limit = 25
	}
	// ensure valid offset value
	if params.Offset < 0 {
		params.Offset = 0
	}

	if params.Pull && scope.SourceName() != "network" {
		return nil, fmt.Errorf("cannot pull without using network source")
	}

	ref, location, err := scope.ParseAndResolveRef(scope.Context(), params.Ref)
	if err != nil {
		return nil, err
	}

	if location == "" {
		// local resolution
		return base.DatasetLog(scope.Context(), scope.Repo(), ref, params.Limit, params.Offset, true)
	}

	logs, err := scope.RemoteClient().FetchLogs(scope.Context(), ref, location)
	if err != nil {
		return nil, err
	}

	// TODO (b5) - FetchLogs currently returns oplogs arranged in user > dataset > branch
	// hierarchy, and we need to descend to the branch oplog to get commit history
	// info. It might be nicer if FetchLogs instead returned the branch oplog, but
	// with .Parent() fields loaded & connected
	if len(logs.Logs) > 0 {
		logs = logs.Logs[0]
		if len(logs.Logs) > 0 {
			logs = logs.Logs[0]
		}
	}

	items := logbook.ConvertLogsToVersionInfos(logs, ref)
	log.Debugf("found %d items: %v", len(items), items)
	if len(items) == 0 {
		return nil, repo.ErrNoHistory
	}

	for i, item := range items {
		local, hasErr := scope.Filesystem().Has(scope.Context(), item.Path)
		if hasErr != nil {
			continue
		}
		items[i].Foreign = !local

		if local {
			if ds, err := dsfs.LoadDataset(scope.Context(), scope.Repo().Filesystem(), item.Path); err == nil {
				if ds.Commit != nil {
					items[i].CommitMessage = ds.Commit.Message
				}
			}
		}
	}

	return items, nil
}

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
