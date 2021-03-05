package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/qri-io/qri/sql"
)

// SQLMethods encapsulates business logic for the qri search command
// TODO (b5): switch to using an Instance instead of separate fields
type SQLMethods struct {
	inst *Instance
}

// NewSQLMethods creates SQLMethods from a qri Instance
func NewSQLMethods(inst *Instance) *SQLMethods {
	return &SQLMethods{inst: inst}
}

// CoreRequestsName implements the requests interface
func (m SQLMethods) CoreRequestsName() string { return "sql" }

// SQLQueryParams defines paremeters for the exec Method
// ExecParams provides parameters to the execute command
type SQLQueryParams struct {
	Query        string
	OutputFormat string
	ResolverMode string
}

// SetNonZeroDefaults sets format to "json" if it's value is an empty string
func (p *SQLQueryParams) SetNonZeroDefaults() {
	if p.OutputFormat == "" {
		p.OutputFormat = "json"
	}
}

// Exec runs an SQL query
func (m *SQLMethods) Exec(ctx context.Context, p *SQLQueryParams) ([]byte, error) {
	if m.inst.http != nil {
		if p.OutputFormat == "json" {
			res := []map[string]interface{}{}
			err := m.inst.http.Call(ctx, AESQL, p, &res)
			if err != nil {
				return nil, err
			}
			bres, err := json.Marshal(res)
			if err != nil {
				return nil, err
			}
			return bres, nil
		}
		var bres bytes.Buffer
		err := m.inst.http.CallRaw(ctx, AESQL, p, &bres)
		if err != nil {
			return nil, err
		}
		return bres.Bytes(), nil
	}

	if p == nil {
		return nil, fmt.Errorf("error: search params cannot be nil")
	}

	resolver, err := m.inst.resolverForMode(p.ResolverMode)
	if err != nil {
		return nil, err
	}
	// create a loader sql will use to load & fetch datasets
	// pass in the configured peername, allowing the "me" alias in reference strings
	loadDataset := NewParseResolveLoadFunc(m.inst.cfg.Profile.Peername, resolver, m.inst)
	svc := sql.New(m.inst.repo, loadDataset)

	buf := &bytes.Buffer{}
	if err := svc.Exec(ctx, buf, p.OutputFormat, p.Query); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
