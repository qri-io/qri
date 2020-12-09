package lib

import (
	"context"

	"github.com/qri-io/qri/changes"
	"github.com/qri-io/qri/dsref"
)

// ChangeReportParams defines parameters for diffing two sources
type ChangeReportParams struct {
	LeftRefstr  string `json:"left"`
	RightRefstr string `json:"right"`
}

// ChangeReport is a simple utility type declaration
type ChangeReport = changes.ChangeReportResponse

// ChangeReport resolves the requested datasets and tries to generate a change report
func (m *DatasetMethods) ChangeReport(p *ChangeReportParams, res *ChangeReport) error {
	ctx := context.TODO()
	reportSource := ""

	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.ChangeReport", p, res))
	}

	right, _, err := m.inst.ParseAndResolveRef(ctx, p.RightRefstr, reportSource)
	if err != nil {
		return err
	}

	var left dsref.Ref
	if p.LeftRefstr != "" {
		if left, _, err = m.inst.ParseAndResolveRef(ctx, p.LeftRefstr, reportSource); err != nil {
			return err
		}
	} else {
		left = dsref.Ref{Username: right.Username, Name: right.Name}
	}

	report, err := changes.New(m.inst, m.inst.stats).Report(ctx, left, right, reportSource)
	if err != nil {
		return err
	}

	*res = *report
	return nil
}
