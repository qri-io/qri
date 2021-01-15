package cmd

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTransformApply(t *testing.T) {
	run := NewTestRunner(t, "test_peer_transform_apply", "qri_test_transform_apply")
	defer run.Delete()

	// Apply a transform which makes a body
	output := run.MustExec(t, "qri apply --file testdata/movies/tf_one_movie.star")
	expectContains := ` "body": [
  [
   "Spectre",
   148
  ]
 ],`
	if !strings.Contains(output, expectContains) {
		t.Errorf("contents mismatch, want: %s, got: %s", expectContains, output)
	}

	// Save a first version with a normal body
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv me/movies")

	// Apply a transform which sets a meta on the existing dataset
	output = run.MustExec(t, "qri apply --file testdata/movies/tf_set_meta.star me/movies")
	expectContains = `"title": "Did Set Title"`
	if !strings.Contains(output, expectContains) {
		t.Errorf("contents mismatch, want: %s, got: %s", expectContains, output)
	}
}

func TestApplyRefNotFound(t *testing.T) {
	run := NewTestRunner(t, "test_peer_transform_apply", "qri_test_transform_apply")
	defer run.Delete()

	// Error to apply a transform using a dataset ref that doesn't exist.
	err := run.ExecCommand("qri apply --file testdata/movies/tf_one_movie.star me/not_found")
	if err == nil {
		t.Errorf("error expected, did not get one")
	}
	expectErr := `reference not found`
	if diff := cmp.Diff(expectErr, err.Error()); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestApplyMetaOnly(t *testing.T) {
	run := NewTestRunner(t, "test_peer_apply_meta_only", "qri_test_apply_meta_only")
	defer run.Delete()

	// Apply a transform which sets a meta to the existing dataset
	output := run.MustExec(t, "qri apply --file testdata/movies/tf_set_meta.star")
	expectContains := `"title": "Did Set Title"`
	if !strings.Contains(output, expectContains) {
		t.Errorf("contents mismatch, want: %s, got: %s", expectContains, output)
	}
}

func TestApplyModifyBody(t *testing.T) {
	run := NewTestRunner(t, "test_peer_apply_mod_body", "qri_test_apply_mod_body")
	defer run.Delete()

	// Save two versions, the second of which uses get_body in a transformation
	run.MustExec(t, "qri save --body=testdata/movies/body_two.json me/test_ds")
	output := run.MustExec(t, "qri apply --file=testdata/movies/tf_add_one.star me/test_ds")
	expectContains := `"body": [
  [
   "Avatar",
   179
  ],
  [
   "Pirates of the Caribbean: At World's End",
   170
  ]
 ],`
	if !strings.Contains(output, expectContains) {
		t.Errorf("contents mismatch, want: %s, got: %s", expectContains, output)
	}
}
