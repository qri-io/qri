package cmd

import (
	"testing"
	// "github.com/google/go-cmp/cmp"
)

func TestSQLRun(t *testing.T) {
	run := NewTestRunner(t, "test_peer_sql_command", "qri_test_sql_command")
	defer run.Delete()

	run.MustExec(t, "qri save me/one_ds --body testdata/movies/body_ten.csv")

	run.MustExecuteQuotedCommand(t, `qri sql "SELECT * FROM me/one_ds as one LIMIT 1" "--format" "csv"`)

	// TODO (b5) - this test uses a spinner, output cleanup will be needed before we can test results
	// var selectAllResultsTable = `
	// `
	// if diff := cmp.Diff(selectAllResultsTable, got); diff != "" {
	// 	t.Errorf("result mismatch. (-want +got): %s\n", diff)
	// }
}
