package cmd

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/errors"
)

func TestValidateComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer_validate_complete", "qri_test_validate_complete")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args           []string
		bodyFilepath   string
		schemaFilepath string
		expect         string
		format         string
		err            string
	}{
		{[]string{}, "filepath", "schemafilepath", "", "table", ""},
		{[]string{"test/foo"}, "", "", "", "", `"" is not a valid output format. Please use one of: "table", "json", "csv"`},
		{[]string{"test/foo"}, "", "", "test/foo", "table", ""},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			defer run.IOReset()

			opt := &ValidateOptions{
				IOStreams:      run.Streams,
				BodyFilepath:   c.bodyFilepath,
				SchemaFilepath: c.schemaFilepath,
				Format:         c.format,
			}
			if err := opt.Complete(f, c.args); err != nil {
				if c.err != err.Error() {
					t.Fatalf("unexpected error. %q != %q", c.err, err.Error())
				}
			}

			if c.expect != opt.Refs.Ref() {
				t.Errorf("case %d, opt.Refs not set correctly. Expected: %q, Got: %q", i, c.expect, opt.Refs.Ref())
			}

			if opt.inst == nil {
				t.Fatalf("case %d, opt.inst not set.", i)
			}
		})
	}

}

func TestValidateRun(t *testing.T) {
	run := NewTestRunner(t, "test_peer_validate_run", "qri_test_validate_run")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	inst, err := f.Instance()
	if err != nil {
		t.Fatalf("error creating instance: %s", err)
	}

	bad := []struct {
		ref           string
		bodyPath      string
		schemaPath    string
		structurePath string
		format        string

		err     string
		message string
	}{
		{"", "", "", "", "table",
			"bad arguments provided", "please provide a dataset name, or a supply the --body and --schema or --structure flags"},
		{"peer/bad_dataset", "", "", "", "table",
			"reference not found", ""},
		{"", "bad/filepath", "", "table", "testdata/days_of_week_schema.json",
			"error opening body file \"bad/filepath\": path not found", ""},
		{"", "testdata/days_of_week.csv", "bad/schema_filepath", "", "",
			"error opening schema file: bad/schema_filepath", ""},
	}

	for _, c := range bad {
		t.Run(c.err, func(t *testing.T) {
			defer run.IOReset()

			opt := &ValidateOptions{
				IOStreams:         run.Streams,
				Refs:              NewExplicitRefSelect(c.ref),
				BodyFilepath:      c.bodyPath,
				SchemaFilepath:    c.schemaPath,
				StructureFilepath: c.structurePath,
				Format:            c.format,
				inst:              inst,
			}

			err := opt.Run()
			if err == nil {
				t.Errorf("expected error, got none")
				return
			}
			if c.err != err.Error() {
				t.Errorf("error mismatch.\nwant: %q\ngot:  %q", c.err, err.Error())
			}

			if libErr, ok := err.(errors.Error); ok {
				if libErr.Message() != c.message {
					t.Errorf("mismatched user-friendly message. Expected: '%s', Got: '%s'", c.message, libErr.Message())
					return
				}
			}
		})
	}

	good := []struct {
		ref           string
		bodyPath      string
		schemaPath    string
		structurePath string
		format        string

		output string
	}{
		{"peer/movies", "", "", "", "table",
			`#  ROW   COL  VALUE  ERROR                              
0  4     1           type should be integer, got string  
1  199   1           type should be integer, got string  
2  206   1           type should be integer, got string  
3  1510  1           type should be integer, got string  
`},
		{"peer/movies", "", "", "", "csv", `#,row,col,value,error
0,4,1,,"type should be integer, got string"
1,199,1,,"type should be integer, got string"
2,206,1,,"type should be integer, got string"
3,1510,1,,"type should be integer, got string"
`},
		{"peer/movies", "", "", "", "json",
			`[{"propertyPath":"/4/1","invalidValue":"","message":"type should be integer, got string"},{"propertyPath":"/199/1","invalidValue":"","message":"type should be integer, got string"},{"propertyPath":"/206/1","invalidValue":"","message":"type should be integer, got string"},{"propertyPath":"/1510/1","invalidValue":"","message":"type should be integer, got string"}]
`},
		{"", "testdata/days_of_week.csv", "testdata/days_of_week_schema.json", "", "table",
			"✔ All good!\n"},
		{"", "testdata/days_of_week.csv", "testdata/days_of_week_schema.json", "", "json",
			"[]\n"},
	}

	for i, c := range good {
		t.Run(fmt.Sprintf("good_%d", i), func(t *testing.T) {
			defer run.IOReset()

			opt := &ValidateOptions{
				IOStreams:         run.Streams,
				Refs:              NewExplicitRefSelect(c.ref),
				BodyFilepath:      c.bodyPath,
				SchemaFilepath:    c.schemaPath,
				StructureFilepath: c.structurePath,
				Format:            c.format,
				inst:              inst,
			}

			if err := opt.Run(); err != nil {
				t.Errorf("unexpected error: %s", err)
			}

			if diff := cmp.Diff(c.output, run.OutStream.String()); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateCommandlineFlags(t *testing.T) {
	run := NewTestRunner(t, "test_peer_validate_commandline_flags", "qri_test_validate_commandline_flags")
	defer run.Delete()

	output := run.MustExec(t, "qri validate --body=testdata/movies/body_ten.csv --structure=testdata/movies/structure_override.json")
	expectContain := `/4/1         type should be integer, got string`

	if !strings.Contains(output, expectContain) {
		t.Errorf("expected output to contain %q, got %q", expectContain, output)
	}

	output = run.MustExec(t, "qri validate --body=testdata/movies/body_ten.csv --schema=testdata/movies/schema_only.json")
	expectContain = `/0/1  duration  type should be integer, got string  
1  /5/1            type should be integer, got string`

	if !strings.Contains(output, expectContain) {
		t.Errorf("expected output to contain %q, got %q", expectContain, output)
	}

	// Fail because both --structure and --schema are given
	err := run.ExecCommand("qri validate --body=testdata/movies/body_ten.csv --structure=testdata/movies/structure_override.json --schema=testdata/movies/schema_only.json")
	if err == nil {
		t.Fatal("expected error, did not get one")
	}
	expect := "bad arguments provided"
	if expect != err.Error() {
		t.Errorf("expected %q, got %q", expect, err.Error())
	}
}
