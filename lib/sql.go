package lib

import (
	"bytes"
	"context"

	"github.com/qri-io/qri/sql"
)

// SQLMethods groups together methods for SQL
type SQLMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m SQLMethods) Name() string { return "sql" }

// Attributes defines attributes for each method
func (m SQLMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"exec": {endpoint: AESQL, httpVerb: "POST"},
	}
}

// SQLQueryParams defines paremeters for running a SQL query
type SQLQueryParams struct {
	Query  string
	Format string
}

// SetNonZeroDefaults sets format to "json" if it's value is an empty string
func (p *SQLQueryParams) SetNonZeroDefaults() {
	if p.Format == "" {
		p.Format = "json"
	}
}

// Exec runs an SQL query
func (m SQLMethods) Exec(ctx context.Context, p *SQLQueryParams) ([]byte, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "exec"), p)
	if res, ok := got.([]byte); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Implementations for SQL methods follow

// sqlImpl holds the method implementations for SQL
type sqlImpl struct{}

// Exec runs an SQL query
func (sqlImpl) Exec(scope scope, p *SQLQueryParams) ([]byte, error) {
	svc := sql.New(scope.Repo(), scope.Loader())
	buf := &bytes.Buffer{}
	if err := svc.Exec(scope.Context(), buf, p.Format, p.Query); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
