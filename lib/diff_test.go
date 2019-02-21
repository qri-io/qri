package lib

func TestDatasetRequestsDiff(t *testing.T) {
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	req := NewDatasetRequests(node, nil)

	// File 1
	fp1, err := dstest.BodyFilepath("testdata/jobs_by_automation")
	if err != nil {
		t.Errorf("getting data filepath: %s", err.Error())
		return
	}

	dsRef1 := repo.DatasetRef{}
	initParams := &SaveParams{
		Dataset: &dataset.Dataset{
			Name:     "jobs_ranked_by_automation_prob",
			BodyPath: fp1,
		},
	}

	err = req.Save(initParams, &dsRef1)
	if err != nil {
		t.Errorf("couldn't init file 1: %s", err.Error())
		return
	}

	// File 2
	fp2, err := dstest.BodyFilepath("testdata/jobs_by_automation_2")
	if err != nil {
		t.Errorf("getting data filepath: %s", err.Error())
		return
	}
	dsRef2 := repo.DatasetRef{}
	initParams = &SaveParams{
		Dataset: &dataset.Dataset{
			Name:     "jobs_ranked_by_automation_prob",
			BodyPath: fp2,
		},
	}
	err = req.Save(initParams, &dsRef2)
	if err != nil {
		t.Errorf("couldn't load second file: %s", err.Error())
		return
	}

	//test cases
	cases := []struct {
		Left, Right   repo.DatasetRef
		All           bool
		Components    map[string]bool
		displayFormat string
		expected      string
		err           string
	}{
		{dsRef1, dsRef2, false, map[string]bool{"structure": true}, "listKeys", "Structure: 3 changes\n\t- modified checksum\n\t- modified length\n\t- modified schema", ""},
		{dsRef1, dsRef2, true, nil, "listKeys", "Structure: 3 changes\n\t- modified checksum\n\t- modified length\n\t- modified schema", ""},
	}
	// execute
	for i, c := range cases {
		p := &DiffParams{
			Left:           c.Left,
			Right:          c.Right,
			DiffAll:        c.All,
			DiffComponents: c.Components,
		}
		res := map[string]*dsdiff.SubDiff{}
		err := req.Diff(p, &res)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected '%s', got '%s'", i, c.err, err.Error())
		}

		if c.err != "" {
			continue
		}

		stringDiffs, err := dsdiff.MapDiffsToString(res, c.displayFormat)
		if err != nil {
			t.Errorf("case %d error mapping to string: %s", i, err.Error())
		}
		if stringDiffs != c.expected {
			t.Errorf("case %d response mistmatch: expected '%s', got '%s'", i, c.expected, stringDiffs)
		}
	}
}