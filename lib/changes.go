package lib

import (
	"context"

	"github.com/qri-io/qri/changes"
)

// ChangeReportParams defines parameters for diffing two sources
type ChangeReportParams struct {
	LeftRef  string `schema:"leftRef" json:"leftRef"`
	RightRef string `schema:"rightRef" json:"rightRef"`
}

// ChangeReport is a simple utility type declaration
type ChangeReport = changes.ChangeReportResponse

// Changes resolves the requested datasets and tries to generate a change report
func (m DiffMethods) Changes(ctx context.Context, p *ChangeReportParams) (*ChangeReport, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "changes"), p)
	if res, ok := got.(*ChangeReport); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Changes generates report of changes between two datasets
func (diffImpl) Changes(scope scope, p *ChangeReportParams) (*ChangeReport, error) {
	svc := changes.New(scope.Loader(), scope.Stats())
	return svc.Report(scope.Context(), p.LeftRef, p.RightRef)
}
