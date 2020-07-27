package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	qerr "github.com/qri-io/qri/errors"
)

func TestDiffComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_diff_complete")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args []string
		err  string
	}{
		{[]string{}, ""},
		{[]string{"one arg"}, ""},
		{[]string{"one arg", "two args"}, ""},
	}

	for i, c := range cases {
		opt := &DiffOptions{
			IOStreams: run.Streams,
		}

		opt.Complete(f, c.args)

		if c.err != run.ErrStream.String() {
			t.Errorf("case %d, error mismatch. Expected: %q, Got: %q", i, c.err, run.ErrStream.String())
			run.IOReset()
			continue
		}

		if opt.DatasetMethods == nil {
			t.Errorf("case %d, opt.DatasetMethods not set.", i)
			run.IOReset()
			continue
		}
		run.IOReset()
	}
}

func TestDiffRun(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_dag_info")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	good := []struct {
		description string
		opt         *DiffOptions
		stdout      string
	}{
		{"diff two dataset metas",
			&DiffOptions{
				Refs:     NewListOfRefSelects([]string{"me/movies", "me/cities"}),
				Selector: "meta",
			},
			`0 elements. 1 insert. 1 delete.

 qri: "md:0"
-title: "example movie data"
+title: "example city data"
`,
		},
		{"diff json output",
			&DiffOptions{
				Refs:     NewListOfRefSelects([]string{"me/movies", "me/cities"}),
				Selector: "meta",
				Format:   "json",
			},
			`{"stat":{"leftNodes":3,"rightNodes":3,"leftWeight":45,"rightWeight":43,"inserts":1,"deletes":1},"diff":[[" ","qri","md:0"],["-","title","example movie data"],["+","title","example city data"]]}
`,
		},
	}

	for _, c := range good {
		t.Run(c.description, func(t *testing.T) {
			dsm, err := f.DatasetMethods()
			if err != nil {
				t.Fatalf("case %s, error creating dataset request: %s", c.description, err)
			}

			opt := c.opt
			opt.IOStreams = run.Streams
			opt.DatasetMethods = dsm

			if err = opt.Run(); err != nil {
				t.Fatalf("case %s unexpected error: %s", c.description, err)
			}

			if diff := cmp.Diff(run.OutStream.String(), c.stdout); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}

			run.IOReset()
		})
	}

	bad := []struct {
		opt *DiffOptions
		err string
	}{
		{
			&DiffOptions{},
			"nothing to diff",
		},
	}

	for _, c := range bad {
		dsm, err := f.DatasetMethods()
		if err != nil {
			t.Errorf("case %s, error creating dataset request: %s", c.err, err)
			continue
		}

		opt := c.opt
		opt.Refs = NewListOfRefSelects([]string{})
		opt.IOStreams = run.Streams
		opt.DatasetMethods = dsm

		err = opt.Run()

		if err == nil {
			t.Errorf("expected: %q, got no error", c.err)
			run.IOReset()
			continue
		}

		var qerror qerr.Error
		if errors.As(err, &qerror) {
			if qerror.Message() != c.err {
				t.Errorf("qri error mismatch. expected:\n%q\n,got:\n%q", c.err, qerror.Message())
			}
		} else if c.err != err.Error() {
			t.Errorf("error mismatch. expected: %q, got: %q", c.err, err.Error())
		}
		run.IOReset()
	}
}

// Test that we can compare bodies of different dataset revisions.
func TestDiffPrevRevision(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_diff_revisions")
	defer run.Delete()

	// Save three versions, then diff the last two
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	output := run.MustExec(t, "qri diff body me/test_movies")

	expect := `+30 elements. 10 inserts. 0 deletes.

 0: ["Avatar ",178]
 1: ["Pirates of the Caribbean: At World's End ",169]
 2: ["Spectre ",148]
 3: ["The Dark Knight Rises ",164]
 4: ["Star Wars: Episode VII - The Force Awakens             ",""]
 5: ["John Carter ",132]
 6: ["Spider-Man 3 ",156]
 7: ["Tangled ",100]
 8: ["Avengers: Age of Ultron ",141]
 9: ["Harry Potter and the Half-Blood Prince ",153]
 10: ["Batman v Superman: Dawn of Justice ",183]
 11: ["Superman Returns ",169]
 12: ["Quantum of Solace ",106]
 13: ["Pirates of the Caribbean: Dead Man's Chest ",151]
 14: ["The Lone Ranger ",150]
 15: ["Man of Steel ",143]
 16: ["The Chronicles of Narnia: Prince Caspian ",150]
 17: ["The Avengers ",173]
+18: ["Dragonfly ",104]
+19: ["The Black Dahlia ",121]
+20: ["Flyboys ",140]
+21: ["The Last Castle ",131]
+22: ["Supernova ",91]
+23: ["Winter's Tale ",118]
+24: ["The Mortal Instruments: City of Bones ",130]
+25: ["Meet Dave ",90]
+26: ["Dark Water ",103]
+27: ["Edtv ",122]
`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// Test that diff works using the name of a component file to mean a selector for that component
func TestDiffKnownFilenameComponent(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_diff_revisions")
	defer run.Delete()

	// Save two versions with a change to the structure
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv --file=testdata/movies/structure_override.json me/test_movies")
	run.MustExec(t, "qri save --file=testdata/movies/structure_rename.json me/test_movies")

	// Diff the structure, using the name of the component file
	output := run.MustExec(t, "qri diff structure.json me/test_movies")

	expect := `0 elements. 1 insert. 1 delete.

 checksum: "QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf"
 depth: 2
 entries: 8
 errCount: 1
 format: "csv"
 formatConfig: {"headerRow":true,"lazyQuotes":false}
 length: 224
 qri: "st:0"
 schema: 
   items: 
     items: 
       0: 
        -title: "name"
        +title: "title"
         type: "string"
       1: {"title":"duration","type":"integer"}
   type: "array"
`

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}
