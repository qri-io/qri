package lib

import (
	"bytes"
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
	inst *Instance
}

// CoreRequestsName implements the Requets interface
func (m LogMethods) CoreRequestsName() string { return "log" }

// NewLogMethods creates a LogMethods pointer from either a repo
// or an rpc.Client
func NewLogMethods(inst *Instance) *LogMethods {
	return &LogMethods{
		inst: inst,
	}
}

// LogParams defines parameters for the Log method
type LogParams struct {
	ListParams
	// Reference to data to fetch history for
	Ref    string
	Pull   bool
	Source string
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *LogParams) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &LogParams{}
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

// Log returns the history of changes for a given dataset
func (m *LogMethods) Log(ctx context.Context, params *LogParams) ([]dsref.VersionInfo, error) {
	res := []dsref.VersionInfo{}
	if m.inst.http != nil {
		err := m.inst.http.Call(ctx, AEHistory, params, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

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

	ref, source, err := m.inst.ParseAndResolveRef(ctx, params.Ref, params.Source)
	if err != nil {
		return nil, err
	}

	if source == "" {
		// local resolution
		return base.DatasetLog(ctx, m.inst.repo, ref, params.Limit, params.Offset, true)
	}

	logs, err := m.inst.remoteClient.FetchLogs(ctx, ref, source)
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
		local, hasErr := m.inst.qfs.Has(ctx, item.Path)
		if hasErr != nil {
			continue
		}
		items[i].Foreign = !local

		if local {
			if ds, err := dsfs.LoadDataset(ctx, m.inst.repo.Filesystem(), item.Path); err == nil {
				if ds.Commit != nil {
					items[i].CommitMessage = ds.Commit.Message
				}
			}
		}
	}

	return items, nil
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

// Logbook lists log entries for actions taken on a given dataset
func (m *LogMethods) Logbook(ctx context.Context, p *RefListParams) ([]LogEntry, error) {
	res := []LogEntry{}
	var err error
	if m.inst.http != nil {
		err = m.inst.http.Call(ctx, AELogbook, p, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	ref, _, err := m.inst.ParseAndResolveRef(ctx, p.Ref, "local")
	if err != nil {
		return nil, err
	}

	book := m.inst.node.Repo.Logbook()
	res, err = book.LogEntries(ctx, ref, p.Offset, p.Limit)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// PlainLogsParams enapsulates parameters for the PlainLogs methods
type PlainLogsParams struct {
	// no options yet
}

// PlainLogs is an alias for a human representation of a plain-old-data logbook
type PlainLogs = []logbook.PlainLog

// PlainLogs encodes the full logbook as human-oriented json
func (m *LogMethods) PlainLogs(ctx context.Context, p *PlainLogsParams) (*PlainLogs, error) {
	res := &PlainLogs{}
	var err error
	if m.inst.http != nil {
		err = m.inst.http.Call(ctx, AELogs, p, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}
	*res, err = m.inst.repo.Logbook().PlainLogs(ctx)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// LogbookSummary returns a string overview of the logbook
func (m *LogMethods) LogbookSummary(ctx context.Context, p *struct{}) (*string, error) {
	res := ""
	if m.inst.http != nil {
		var bres bytes.Buffer
		err := m.inst.http.CallRaw(ctx, AELogbookSummary, p, &bres)
		if err != nil {
			return nil, err
		}
		res = bres.String()
		return &res, nil
	}
	res = m.inst.repo.Logbook().SummaryString(ctx)
	return &res, nil
}
