package lib

import (
	"context"
	"fmt"
	"net/http"

	"github.com/qri-io/qri/api/util"
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
		"history":        {AEHistory, "POST"},
		"log":            {AELog, "POST"},
		"rawlogbook":     {denyRPC, ""},
		"logbooksummary": {denyRPC, ""},
	}
}

// HistoryParams defines parameters for the Log method
type HistoryParams struct {
	ListParams
	// Reference to data to fetch history for
	Ref    string
	Pull   bool
	Source string
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *HistoryParams) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &HistoryParams{}
	}

	lp := &ListParams{}
	if err := lp.UnmarshalFromRequest(r); err != nil {
		return err
	}

	p.ListParams = *lp

	params := *p
	if params.Ref == "" {
		params.Ref = r.FormValue("refstr")
	}

	ref, err := dsref.Parse(params.Ref)
	if err != nil {
		return err
	}
	lp.Peername = ref.Username

	local := r.FormValue("local") == "true"
	remoteName := r.FormValue("remote")
	params.Pull = r.FormValue("pull") == "true" || params.Pull

	if params.Source == "" {
		if local && (remoteName != "" || params.Pull) {
			return fmt.Errorf("cannot use the 'local' param with either the 'remote' or 'pull' params")
		} else if local {
			remoteName = "local"
		}
		params.Source = remoteName
	}

	*p = params
	return nil
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

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *RefListParams) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &RefListParams{}
	}

	params := *p
	if params.Ref == "" {
		params.Ref = r.FormValue("refstr")
	}

	_, err := dsref.Parse(params.Ref)
	if err != nil {
		return err
	}

	if i := util.ReqParamInt(r, "offset", 0); i != 0 {
		params.Offset = i
	}
	if i := util.ReqParamInt(r, "limit", 0); i != 0 {
		params.Limit = i
	}

	*p = params
	return nil
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

	if params.Pull {
		switch params.Source {
		case "":
			params.Source = "network"
		case "local":
			return nil, fmt.Errorf("cannot pull with only local source")
		}
	}

	ref, source, err := scope.ParseAndResolveRef(scope.Context(), params.Ref, params.Source)
	if err != nil {
		return nil, err
	}

	if source == "" {
		// local resolution
		return base.DatasetLog(scope.Context(), scope.Repo(), ref, params.Limit, params.Offset, true)
	}

	logs, err := scope.RemoteClient().FetchLogs(scope.Context(), ref, source)
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

	ref, _, err := scope.ParseAndResolveRef(scope.Context(), p.Ref, "local")
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
