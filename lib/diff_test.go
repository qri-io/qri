package lib

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	reporef "github.com/qri-io/qri/repo/ref"
)

func TestDatasetRequestsDiff(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	req := NewDatasetMethods(tr.Instance)
	jobsOnePath := tr.writeFile(t, "jobs_by_automation_1.csv", jobsByAutomationData1)
	jobsTwoPath := tr.writeFile(t, "jobs_by_automation_2.csv", jobsByAutomationData2)

	djsOnePath := tr.writeFile(t, "djs_1.json", `{ "dj dj booth": { "rating": 1, "uses_soundcloud": true } }`)
	djsTwoPath := tr.writeFile(t, "djs_2.json", `{ "DJ dj booth": { "rating": 1, "uses_soundcloud": true } }`)

	dsRef1 := reporef.DatasetRef{}
	initParams := &SaveParams{
		Ref:      "me/jobs_ranked_by_automation_prob",
		BodyPath: jobsOnePath,
	}
	if err := req.Save(initParams, &dsRef1); err != nil {
		t.Fatalf("couldn't save: %s", err.Error())
	}

	dsRef2 := reporef.DatasetRef{}
	initParams = &SaveParams{
		Ref:      "me/jobs_ranked_by_automation_prob",
		BodyPath: jobsTwoPath,
	}
	if err := req.Save(initParams, &dsRef2); err != nil {
		t.Fatalf("couldn't save second revision: %s", err.Error())
	}

	good := []struct {
		description string
		Left, Right string
		Selector    string
		Stat        *DiffStat
		DeltaLen    int
	}{
		{"two fully qualified references",
			dsRef1.String(), dsRef2.String(),
			"",
			&DiffStat{Left: 41, Right: 42, LeftWeight: 2728, RightWeight: 2686, Inserts: 8, Updates: 0, Deletes: 8},
			9,
		},
		{"fill left path from history",
			dsRef2.AliasString(), dsRef2.AliasString(),
			"",
			&DiffStat{Left: 41, Right: 42, LeftWeight: 2728, RightWeight: 2686, Inserts: 8, Updates: 0, Deletes: 8},
			9,
		},
		{"two local file paths",
			"testdata/jobs_by_automation/body.csv", "testdata/jobs_by_automation_2/body.csv",
			"",
			&DiffStat{Left: 151, Right: 151, LeftWeight: 3757, RightWeight: 3757, Inserts: 1, Updates: 0, Deletes: 1},
			30,
		},
		{"diff local csv & json file",
			"testdata/now_tf/input.dataset.json", "testdata/jobs_by_automation/body.csv",
			"",
			&DiffStat{Left: 10, Right: 151, LeftWeight: 162, RightWeight: 3757, Inserts: 1, Updates: 0, Deletes: 1},
			2,
		},
		{"case-sensitive key change",
			djsOnePath, djsTwoPath,
			"",
			&DiffStat{Left: 4, Right: 4, LeftWeight: 18, RightWeight: 18, Inserts: 1, Updates: 0, Deletes: 1},
			2,
		},
	}

	// execute
	for i, c := range good {
		t.Run(c.description, func(t *testing.T) {
			p := &DiffParams{
				LeftPath:  c.Left,
				RightPath: c.Right,
				Selector:  c.Selector,
			}
			// If test has same two paths, assume we want the previous version compared against head.
			if p.LeftPath == p.RightPath {
				p.IsLeftAsPrevious = true
			}
			res := &DiffResponse{}
			err := req.Diff(p, res)
			if err != nil {
				t.Errorf("%d: \"%s\" error: %s", i, c.description, err.Error())
				return
			}

			if diff := cmp.Diff(c.Stat, res.Stat); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}

			// dlt, _ := json.MarshalIndent(res.Diff, "", "  ")
			// t.Logf("%s", dlt)

			if len(res.Diff) != c.DeltaLen {
				t.Errorf("%d: \"%s\" delta length mismatch. want: %d got: %d", i, c.description, c.DeltaLen, len(res.Diff))
			}
		})
	}
}

const jobsByAutomationData1 = `
rank,probability_of_automation,soc_code,job_title
702,"0.99","41-9041","Telemarketers"
701,"0.99","23-2093","Title Examiners, Abstractors, and Searchers"
700,"0.99","51-6051","Sewers, Hand"
699,"0.99","15-2091","Mathematical Technicians"
698,"0.99","13-2053","Insurance Underwriters"
697,"0.99","49-9064","Watch Repairers"
696,"0.99","43-5011","Cargo and Freight Agents"
695,"0.99","13-2082","Tax Preparers"
694,"0.99","51-9151","Photographic Process Workers and Processing Machine Operators"
693,"0.99","43-4141","New Accounts Clerks"
692,"0.99","25-4031","Library Technicians"
691,"0.99","43-9021","Data Entry Keyers"
690,"0.98","51-2093","Timing Device Assemblers and Adjusters"
689,"0.98","43-9041","Insurance Claims and Policy Processing Clerks"
688,"0.98","43-4011","Brokerage Clerks"
687,"0.98","43-4151","Order Clerks"
686,"0.98","13-2072","Loan Officers"
685,"0.98","13-1032","Insurance Appraisers, Auto Damage"
684,"0.98","27-2023","Umpires, Referees, and Other Sports Officials"
683,"0.98","43-3071","Tellers"
682,"0.98","51-9194","Etchers and Engravers"
681,"0.98","51-9111","Packaging and Filling Machine Operators and Tenders"
680,"0.98","43-3061","Procurement Clerks"
679,"0.98","43-5071","Shipping, Receiving, and Traffic Clerks"
678,"0.98","51-4035","Milling and Planing Machine Setters, Operators, and Tenders, Metal and Plastic"
677,"0.98","13-2041","Credit Analysts"
676,"0.98","41-2022","Parts Salespersons"
675,"0.98","13-1031","Claims Adjusters, Examiners, and Investigators"
674,"0.98","53-3031","Driver/Sales Workers"
673,"0.98","27-4013","Radio Operators"
`

const jobsByAutomationData2 = `
rank,probability_of_automation,industry_code,job_name
702,"0.99","41-9041","Telemarketers"
701,"0.99","23-2093","Title Examiners, Abstractors, and Searchers"
700,"0.99","51-6051","Sewers, Hand"
699,"0.99","15-2091","Mathematical Technicians"
698,"0.88","13-2053","Insurance Underwriters"
697,"0.99","49-9064","Watch Repairers"
696,"0.99","43-5011","Cargo and Freight Agents"
695,"0.99","13-2082","Tax Preparers"
694,"0.99","51-9151","Photographic Process Workers and Processing Machine Operators"
693,"0.99","43-4141","New Accounts Clerks"
692,"0.99","25-4031","Library Technicians"
691,"0.99","43-9021","Data Entry Keyers"
690,"0.98","51-2093","Timing Device Assemblers and Adjusters"
689,"0.98","43-9041","Insurance Claims and Policy Processing Clerks"
688,"0.98","43-4011","Brokerage Clerks"
687,"0.98","43-4151","Order Clerks"
686,"0.98","13-2072","Loan Officers"
685,"0.98","13-1032","Insurance Appraisers, Auto Damage"
684,"0.98","27-2023","Umpires, Referees, and Other Sports Officials"
683,"0.98","43-3071","Tellers"
682,"0.98","51-9194","Etchers and Engravers"
681,"0.98","51-9111","Packaging and Filling Machine Operators and Tenders"
680,"0.98","43-3061","Procurement Clerks"
679,"0.98","43-5071","Shipping, Receiving, and Traffic Clerks"
678,"0.98","51-4035","Milling and Planing Machine Setters, Operators, and Tenders, Metal and Plastic"
677,"0.98","13-2041","Credit Analysts"
676,"0.98","41-2022","Parts Salespersons"
675,"0.98","13-1031","Claims Adjusters, Examiners, and Investigators"
674,"0.98","53-3031","Driver/Sales Workers"
673,"0.98","27-4013","Radio Operators"
`
