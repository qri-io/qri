package dataset_sql

import (
	// "fmt"
	// "github.com/qri-io/dataset"
	// "strconv"
	// "strings"
	st "github.com/qri-io/dataset_sql/sqltypes"
	"testing"
)

func TestCompareLike(t *testing.T) {
	apples, _ := st.BuildValue("apples")
	app, _ := st.BuildValue("app%")
	pples, _ := st.BuildValue("%apples")
	appl, _ := st.BuildValue("%appl%")
	underscores, _ := st.BuildValue("______")
	APPLES, _ := st.BuildValue("APPLES")

	cases := []struct {
		str, expr st.Value
		expect    bool
		err       error
	}{
		{apples, apples, true, nil},
		{apples, app, true, nil},
		{apples, pples, true, nil},
		{apples, appl, true, nil},
		{apples, underscores, true, nil},
		{apples, APPLES, true, nil},
	}

	for i, c := range cases {
		got, err := CompareLike(c.str, c.expr)
		if c.err != err {
			t.Errorf("case %d error mistmatch. expected: %s, got: %s", i, c.err, err)
			continue
		}

		if c.expect != got {
			t.Errorf("case %d mismatch. %t != %t", i, c.expect, got)
		}
	}
}

// type compare struct {
// 	stmt     string
// 	response bool
// 	err      error
// }

// func runComparisons(t *testing.T, cases []compare) {
// 	for i, c := range cases {
// 		stmt, err := Parse(fmt.Sprintf("select %s as result", c.stmt))
// 		if err != nil {
// 			t.Errorf("case %d unexpected parse error: %s", i, err.Error())
// 			continue
// 		}
// 		_, res, err := stmt.Exec(nil, nil, &ExecOpt{Format: dataset.CsvDataFormat})
// 		if c.err != err {
// 			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
// 			continue
// 		}
// 		if c.err == nil {
// 			response, err := strconv.ParseBool(strings.TrimSpace(string(res)))
// 			if err != nil {
// 				t.Errorf("case %d unexpected boolean parsing error: %s", i, err.Error())
// 				continue
// 			}
// 			if response != c.response {
// 				t.Errorf("case %d comparison mismatch. expected: %t, got: %t", i, c.response, response)
// 			}
// 		}
// 	}
// }

// func TestCompareStrVal(t *testing.T) {
// 	runComparisons(t, []compare{
// 		{"'a' = 'a'", true, nil},
// 		{"'a' = 'b'", false, nil},
// 	})
// }

// func TestCompareNumVal(t *testing.T) {
// 	runComparisons(t, []compare{
// 		{"1 = 1", true, nil},
// 		{"1 > 1", false, nil},
// 		{"1 < 1", false, nil},
// 		{"1 < 0", false, nil},
// 		{"1 > 0", true, nil},
// 		{"1.1 > 1.0", true, nil},
// 		{"1.1 > 1", true, nil},
// 	})
// }

// func TestCompareValArg(t *testing.T) {
// 	runComparisons(t, []compare{})
// }

// func TestCompareNullVal(t *testing.T) {
// 	runComparisons(t, []compare{})
// }

// func TestCompareBoolVal(t *testing.T) {
// 	runComparisons(t, []compare{
// 		{"true", true, nil},
// 		{"false", false, nil},
// 		{"true = true", true, nil},
// 		{"false = false", true, nil},
// 		{"false = false", true, nil},
// 	})
// }

// func TestCompareColName(t *testing.T) {
// 	runComparisons(t, []compare{})
// }

// func TestCompareValTuple(t *testing.T) {
// 	runComparisons(t, []compare{})
// }

// func TestCompareSubquery(t *testing.T) {
// 	runComparisons(t, []compare{})
// }

// func TestCompareListArg(t *testing.T) {
// 	runComparisons(t, []compare{})
// }

// func TestCompareBinaryExpr(t *testing.T) {
// 	runComparisons(t, []compare{})
// }

// func TestCompareUnaryExpr(t *testing.T) {
// 	runComparisons(t, []compare{})
// }

// func TestCompareIntervalExpr(t *testing.T) {
// 	runComparisons(t, []compare{})
// }

// func TestCompareFuncExpr(t *testing.T) {
// 	runComparisons(t, []compare{})
// }

// func TestCompareCaseExpr(t *testing.T) {
// 	runComparisons(t, []compare{})
// }
