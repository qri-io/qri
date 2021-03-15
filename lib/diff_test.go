package lib

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/dsref"
)

func TestDatasetMethodsDiff(t *testing.T) {
	tr := newTestRunner(t)
	defer tr.Delete()

	req := tr.Instance.Dataset()
	jobsOnePath := tr.MustWriteTmpFile(t, "jobs_by_automation_1.csv", jobsByAutomationData1)
	jobsTwoPath := tr.MustWriteTmpFile(t, "jobs_by_automation_2.csv", jobsByAutomationData2)

	djsOnePath := tr.MustWriteTmpFile(t, "djs_1.json", `{ "dj dj booth": { "rating": 1, "uses_soundcloud": true } }`)
	djsTwoPath := tr.MustWriteTmpFile(t, "djs_2.json", `{ "DJ dj booth": { "rating": 1, "uses_soundcloud": true } }`)

	initParams := &SaveParams{
		Ref:      "me/jobs_ranked_by_automation_prob",
		BodyPath: jobsOnePath,
	}
	ds1, err := req.Save(tr.Ctx, initParams)
	if err != nil {
		t.Fatalf("couldn't save: %s", err.Error())
	}

	initParams = &SaveParams{
		Ref:      "me/jobs_ranked_by_automation_prob",
		BodyPath: jobsTwoPath,
	}
	ds2, err := req.Save(tr.Ctx, initParams)
	if err != nil {
		t.Fatalf("couldn't save second revision: %s", err.Error())
	}

	dsRef1 := dsref.ConvertDatasetToVersionInfo(ds1).SimpleRef()
	dsRef2 := dsref.ConvertDatasetToVersionInfo(ds2).SimpleRef()

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
			&DiffStat{Left: 205, Right: 208, LeftWeight: 4920, RightWeight: 4913, Inserts: 18, Updates: 0, Deletes: 13},
			9,
		},
		{"fill left path from history",
			dsRef2.Alias(), dsRef2.Alias(),
			"",
			&DiffStat{Left: 205, Right: 208, LeftWeight: 4920, RightWeight: 4913, Inserts: 18, Updates: 0, Deletes: 13},
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
				LeftSide:  c.Left,
				RightSide: c.Right,
				Selector:  c.Selector,
			}
			// If test has same two paths, we want the previous version compared to head
			if p.LeftSide == p.RightSide {
				p.UseLeftPrevVersion = true
				p.RightSide = ""
			}
			res, err := req.Diff(tr.Ctx, p)
			if err != nil {
				t.Errorf("%d: \"%s\" error: %s", i, c.description, err.Error())
				return
			}

			if diff := cmp.Diff(c.Stat, res.Stat); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}

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

// Test that we can compare bodies of different dataset revisions.
func TestDiffPrevRevision(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	// Save three versions, then diff the last head against its previous version
	run.MustSaveFromBody(t, "test_cities", "testdata/cities_2/body.csv")
	run.MustSaveFromBody(t, "test_cities", "testdata/cities_2/body_more.csv")
	run.MustSaveFromBody(t, "test_cities", "testdata/cities_2/body_even_more.csv")

	output, err := run.Diff("me/test_cities", "", "body")
	if err != nil {
		t.Fatal(err)
	}

	// TODO(dustmop): Come up with a better way to represent this diff, that still looks nice when
	// compared with cmp.Diff.
	expect := `{"stat":{"leftNodes":36,"rightNodes":46,"leftWeight":510,"rightWeight":637,"inserts":4,"deletes":2},"diff":[[" ",0,["toronto",50000000,55.5,false]],[" ",1,["new york",8500000,44.4,true]],[" ",2,["los angeles",3990000,42.7,true]],["-",3,["chicago",300000,44.4,true]],["+",3,["dallas",1340000,30,true]],[" ",4,["chatham",35000,65.25,true]],[" ",5,null,[[" ",0,"mexico city"],["-",1,70000000],["+",1,80000000],[" ",2,28.6],[" ",3,false]]],[" ",6,["raleigh",250000,50.65,true]],["+",7,["paris",2100000,41.1,false]],["+",8,["london",8900000,36.5,false]]]}`

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// Test that we can compare two different datasets
func TestDiff(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	// Save a dataset with one version
	run.MustSaveFromBody(t, "test_cities", "testdata/cities_2/body.csv")
	// Save a different dataset with one version
	run.MustSaveFromBody(t, "test_more", "testdata/cities_2/body_more.csv")

	// Diff the heads
	output, err := run.Diff("me/test_cities", "me/test_more", "")
	if err != nil {
		t.Fatal(err)
	}

	// TODO(dustmop): Come up with a better way to represent this diff, that still looks nice when
	// compared with cmp.Diff.
	// TODO(dustmop): Would be better if Diff only returned the changes, instead of things that
	// stay the same, since the delta in this case is pretty small.
	expect := `{"stat":{"leftNodes":102,"rightNodes":112,"leftWeight":3814,"rightWeight":4170,"inserts":51,"deletes":25},"diff":[["-","bodyPath","/mem/Qmc7AoCfFVW5xe8qhyjNYewSgBHFubp6yLM3mfBzQp7iTr"],["+","bodyPath","/mem/QmYuVj1JvALB9Au5DNcVxGLMcWCDBUfbKCN3QbpvissSC4"],[" ","commit",null,[[" ","author",{"id":"QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"}],["-","message","created dataset from body.csv"],["+","message","created dataset from body_more.csv"],["-","path","/mem/QmPuJT13PENgG6yQQsC5SxRickT8CfBGdP27Q1DFzYbR1U"],["+","path","/mem/QmQm7BGbSbjXSXJopZG23j1ruAYHNPMGk5rsbsvaJ2AKH3"],[" ","qri","cm:0"],["-","signature","aO6TIie3VfmC8fvzZL4BulTUYd98nEMzvXrsw8Ght3Tt4vDEaZ58v1iogI1BHVX4lVLlVIJn2F5JK1vZEGB/Du8Y9qvZi1VPvZIT4kRBG4KcGezNHQmiQidk9GgnuqgVVW+wPgDyAwrP+FD2B/3U+UXTeqUKiTeaYEiS+8odqnBWkA0lNjkIROVz3Q7bOmKUZePzvCDONv4zugpLgKSPNuEbAv8qf2rVZLzrOiA7C1z1dh1kgrfv+RNgrNwWL6PmdfebE2ND5kr9X6qesZamUXyrK97Zb/KDZwEOsDPUQlQYdXlzvi0yx+o2V7vnwbkUN6PO9ytmOsh3egL7TTJH1A=="],["+","signature","M1RDiczU7mze3siJoi5BCi73oV7c0bXisPXYlZD11fvXeCCiu+NYvUMcDIOB5u/k/CNn4zi51NX503mxy05wS6I7D5FDl6GME9qyiky8KY5vDobgFBq93ht+p+/arTo2G0a0FdWgaQf9c0YWid1xTbBlZ2ED558AZFBnH7QHhtZRc5YDRPVAEMP/0R9pgDsM4lfo52cOfIxmg7cRkcjfvJ6pkTRPMDNu/M1ZvveF0fq5sMMJ+8joA0Kh9IkLU2CiwxfkFKcQi2fDcixjpFoikyRqQbLYATMRDrB+s18eSMMO5AKMp41WonBGBaAv+j0RGEONtJBaN7pmQU5od3Y9Tw=="],["-","timestamp","2001-01-01T01:01:01.000000001Z"],["+","timestamp","2001-01-01T01:02:01.000000001Z"],["-","title","created dataset from body.csv"],["+","title","created dataset from body_more.csv"]]],["-","path","/mem/QmQqnpYhA7UfcyCLkupkd35VLY8AL9XSQEtY5QhhLvV5jj"],["+","path","/mem/QmS6G5QpWtdHGeQMT3Nn2pqmdvQYJPSN8L7DyxVcmqu342"],[" ","qri","ds:0"],[" ","stats",null,[["-","path","/mem/QmVvv9vBHLsYbDYzG1G931Pi58gTPdVE4hcUsxb6rAU8S9"],["+","path","/mem/QmcLRdnni9jpvhACHN9LmUsDuT2iiqpEhQ1ZqxdLK5ELr6"],[" ","qri","sa:0"],[" ","stats",null,[[" ",0,null,[["-","count",5],["+","count",7],[" ","frequencies",null,[[" ","chatham",1],[" ","chicago",1],["+","los angeles",1],["+","mexico city",1],[" ","new york",1],[" ","raleigh",1],[" ","toronto",1]]],["-","maxLength",8],["+","maxLength",11],[" ","minLength",7],[" ","type","string"],["-","unique",5],["+","unique",7]]],[" ",1,null,[["-","count",5],["+","count",7],[" ","histogram",null,[[" ","bins",null,[["+",0,35000],[" ",1,250000],["+",2,300000],["+",3,3990000],[" ",4,8500000],["+",5,50000000],["+",6,70000000],["+",7,70000001]]],[" ","frequencies",null,[["+",0,1],["+",1,1],["+",2,1],["+",3,1],["+",4,1],["+",5,1],["+",6,1]]]]],["-","max",50000000],["+","max",70000000],["-","mean",11817000],["+","mean",19010714.285714287],["-","median",300000],["+","median",3990000],[" ","min",35000],[" ","type","numeric"]]],[" ",2,null,[["-","count",5],["+","count",7],[" ","histogram",null,[[" ","bins",null,[["+",0,28.6],["+",1,42.7],["+",2,44.4],["+",3,50.65],[" ",4,55.5],["+",5,65.25],[" ",6,66.25]]],[" ","frequencies",null,[["+",0,1],["+",1,1],["+",2,2],["+",3,1],["+",4,1],["+",5,1]]]]],[" ","max",65.25],["-","mean",52.04],["+","mean",47.357142857142854],[" ","median",50.65],["-","min",44.4],["+","min",28.6],[" ","type","numeric"]]],[" ",3,null,[["-","count",5],["+","count",7],["-","falseCount",1],["+","falseCount",2],["-","trueCount",4],["+","trueCount",5],[" ","type","boolean"]]]]]]],[" ","structure",null,[["-","checksum","/mem/Qmc7AoCfFVW5xe8qhyjNYewSgBHFubp6yLM3mfBzQp7iTr"],["+","checksum","/mem/QmYuVj1JvALB9Au5DNcVxGLMcWCDBUfbKCN3QbpvissSC4"],[" ","depth",2],["-","entries",5],["+","entries",7],[" ","format","csv"],[" ","formatConfig",{"headerRow":true,"lazyQuotes":true}],["-","length",155],["+","length",217],["-","path","/mem/QmX3HjmvFGYXavQiPqpJvAZZ14J1DNPjCCGEzEy9NgZq2J"],["+","path","/mem/QmNpWHSFo8xwNQyamsaYAqUKu7Nzd3J1a4MukyNVf4J1xt"],[" ","qri","st:0"],[" ","schema",{"items":{"items":[{"title":"city","type":"string"},{"title":"pop","type":"integer"},{"title":"avg_age","type":"number"},{"title":"in_usa","type":"boolean"}],"type":"array"},"type":"array"}]]]]}`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Diff the bodies
	output, err = run.Diff("me/test_cities", "me/test_more", "body")
	if err != nil {
		t.Fatal(err)
	}

	expect = `{"stat":{"leftNodes":26,"rightNodes":36,"leftWeight":344,"rightWeight":510,"inserts":2},"diff":[[" ",0,["toronto",50000000,55.5,false]],[" ",1,["new york",8500000,44.4,true]],["+",2,["los angeles",3990000,42.7,true]],[" ",3,["chicago",300000,44.4,true]],[" ",4,["chatham",35000,65.25,true]],["+",5,["mexico city",70000000,28.6,false]],[" ",6,["raleigh",250000,50.65,true]]]}`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// Test that diffing a dataset with only one version produces an error
func TestDiffOnlyOneRevision(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	run.MustSaveFromBody(t, "test_cities", "testdata/cities_2/body.csv")
	_, err := run.Diff("me/test_cities", "", "body")
	if err == nil {
		t.Fatal("expected error, did not get one")
	}
	expect := `dataset has only one version, nothing to diff against`
	if err.Error() != expect {
		t.Errorf("expected error: %q, got: %q", expect, err)
	}
}

// Test that we can compare csv files
func TestDiffLocalCsvFiles(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	output, err := run.Diff("testdata/cities_2/body.csv", "testdata/cities_2/body_more.csv", "")
	if err != nil {
		t.Fatal(err)
	}
	expect := `{"stat":{"leftNodes":26,"rightNodes":36,"leftWeight":344,"rightWeight":510,"inserts":2},"schemaStat":{"leftNodes":5,"rightNodes":5,"leftWeight":41,"rightWeight":41},"schema":[[" ",0,"city"],[" ",1,"pop"],[" ",2,"avg_age"],[" ",3,"in_usa"]],"diff":[[" ",0,["toronto",50000000,55.5,false]],[" ",1,["new york",8500000,44.4,true]],["+",2,["los angeles",3990000,42.7,true]],[" ",3,["chicago",300000,44.4,true]],[" ",4,["chatham",35000,65.25,true]],["+",5,["mexico city",70000000,28.6,false]],[" ",6,["raleigh",250000,50.65,true]]]}`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// Test that we can compare json files
func TestDiffLocalJsonFiles(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	output, err := run.Diff("../cmd/testdata/movies/body_two.json", "../cmd/testdata/movies/body_four.json", "")
	if err != nil {
		t.Fatal(err)
	}
	expect := `{"stat":{"leftNodes":7,"rightNodes":13,"leftWeight":161,"rightWeight":267,"inserts":2},"schemaStat":{"leftNodes":2,"rightNodes":2,"leftWeight":11,"rightWeight":11},"schema":[[" ","type","array"]],"diff":[[" ",0,["Avatar",178]],[" ",1,["Pirates of the Caribbean: At World's End",169]],["+",2,["Spectre",148]],["+",3,["The Dark Knight Rises",164]]]}`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

func TestDiffErrors(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	// Save a dataset with one version
	run.MustSaveFromBody(t, "test_cities", "testdata/cities_2/body.csv")

	// Save a different dataset with one version
	run.MustSaveFromBody(t, "test_more", "testdata/cities_2/body_more.csv")

	// Error to compare a dataset ref to a file
	_, err := run.Diff("me/test_cities", "testdata/cities_2/body_even_more.csv", "")
	expectErr := `cannot compare a file to dataset, must compare similar things`
	if diff := cmp.Diff(expectErr, errorMessage(err)); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Error to only set left-side
	_, err = run.DiffWithParams(&DiffParams{
		LeftSide: "me/test_cities",
	})
	expectErr = `invalid parameters to diff`
	if diff := cmp.Diff(expectErr, errorMessage(err)); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Error to set left-side with both WorkingDir and UseLeftPrevVersion
	_, err = run.DiffWithParams(&DiffParams{
		LeftSide:           "me/test_cities",
		WorkingDir:         "workdir",
		UseLeftPrevVersion: true,
	})
	expectErr = `cannot use both previous version and working directory`
	if diff := cmp.Diff(expectErr, errorMessage(err)); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Error to set left-side and right-side with WorkingDir
	_, err = run.DiffWithParams(&DiffParams{
		LeftSide:   "me/test_cities",
		RightSide:  "me/test_more",
		WorkingDir: "workdir",
	})
	expectErr = `cannot use working directory when comparing two sources`
	if diff := cmp.Diff(expectErr, errorMessage(err)); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Error to set left-side and right-side with UseLeftPrevVersion
	_, err = run.DiffWithParams(&DiffParams{
		LeftSide:           "me/test_cities",
		RightSide:          "me/test_more",
		UseLeftPrevVersion: true,
	})
	expectErr = `cannot use previous version when comparing two sources`
	if diff := cmp.Diff(expectErr, errorMessage(err)); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Error to use a selector for a field that doesn't exist
	_, err = run.Diff("me/test_cities", "me/test_more", "meta")
	expectErr = `component "meta" not found`
	if diff := cmp.Diff(expectErr, errorMessage(err)); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// TODO(dustmop): Test comparing a dataset in FSI, with a modification in the working directory
// TODO(dustmop): Test comparing a dataset in FSI, using selector

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
