package base

// func TestPrepareDatasetSave(t *testing.T) {
// 	r := newTestRepo(t)
// 	ref := addCitiesDataset(t, r)

// 	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}

// 	dsp := tc.Input.Encode()
// 	dsp.Meta.Title = "updated"
// 	dsp.Name = ref.Name
// 	dsp.Peername = ref.Peername

// 	_, _, _, err = PrepareDatasetSave(r, dsp)
// 	if err != nil {
// 		t.Error(err.Error())
// 	}
// }
