package dataset_sql

import (
	"bytes"
	q "github.com/qri-io/dataset_sql/vt/proto/query"
	"testing"
)

type evalTestCase struct {
	exp Expr
	t   q.Type
	res []byte
	err error
}

func runEvalCases(t *testing.T, cases []evalTestCase) {
	for i, c := range cases {
		typ, res, err := c.exp.Eval()
		if c.err != err {
			t.Errorf("case %d error mistmatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		if typ != c.t {
			t.Errorf("case %d type mismatch. expected: %d, got: %d", i, c.t, typ)
		}
		if !bytes.Equal(c.res, res) {
			t.Errorf("case %d res mismatch. expected: %s, got: %s", i, string(c.res), string(res))
		}
	}
}

func TestEvalAndExpr(t *testing.T) {
	cases := []evalTestCase{
		{&AndExpr{Left: BoolVal(true), Right: BoolVal(false)}, QueryBoolType, falseB, nil},
		{&AndExpr{Left: BoolVal(true), Right: BoolVal(true)}, QueryBoolType, trueB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalOrExpr(t *testing.T) {
	cases := []evalTestCase{
		{&OrExpr{Left: BoolVal(true), Right: BoolVal(false)}, QueryBoolType, trueB, nil},
		{&OrExpr{Left: BoolVal(false), Right: BoolVal(true)}, QueryBoolType, trueB, nil},
		{&OrExpr{Left: BoolVal(true), Right: BoolVal(true)}, QueryBoolType, trueB, nil},
		{&OrExpr{Left: BoolVal(false), Right: BoolVal(false)}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalNotExpr(t *testing.T) {
	cases := []evalTestCase{
		{&NotExpr{Expr: BoolVal(false)}, QueryBoolType, trueB, nil},
		{&NotExpr{Expr: BoolVal(true)}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalParenExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&ParenExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalComparisonExpr(t *testing.T) {
	cases := []evalTestCase{
		{&ComparisonExpr{Operator: EqualStr, Left: BoolVal(true), Right: BoolVal(true)}, QueryBoolType, trueB, nil},
		{&ComparisonExpr{Operator: EqualStr, Left: BoolVal(true), Right: BoolVal(false)}, QueryBoolType, falseB, nil},
		{&ComparisonExpr{Operator: LikeStr, Left: &SQLVal{Type: StrVal, Val: []byte("apples")}, Right: &SQLVal{Type: StrVal, Val: []byte("apples")}}, QueryBoolType, trueB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalRangeCond(t *testing.T) {
	cases := []evalTestCase{
	//{&RangeCond{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalIsExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&IsExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalExistsExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&ExistsExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalSQLVal(t *testing.T) {
	cases := []evalTestCase{
	//{&SQLVal{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalNullVal(t *testing.T) {
	cases := []evalTestCase{
	//{&NullVal{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalBoolVal(t *testing.T) {
	cases := []evalTestCase{
	//{&BoolVal{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalColName(t *testing.T) {
	cases := []evalTestCase{
	//{&ColName{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalValTuple(t *testing.T) {
	cases := []evalTestCase{
	//{&ValTuple{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalSubquery(t *testing.T) {
	cases := []evalTestCase{
	//{&Subquery{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalListArg(t *testing.T) {
	cases := []evalTestCase{
	//{&ListArg{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalBinaryExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&BinaryExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalUnaryExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&UnaryExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalIntervalExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&IntervalExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalCollateExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&CollateExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalFuncExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&FuncExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalGroupConcatExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&GroupConcatExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalValuesFuncExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&ValuesFuncExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalConvertExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&ConvertExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalConvertUsingExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&ConvertUsingExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalMatchExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&MatchExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalCaseExpr(t *testing.T) {
	cases := []evalTestCase{
	//{&CaseExpr{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
func TestEvalWhere(t *testing.T) {
	cases := []evalTestCase{
	//{&Where{}, QueryBoolType, falseB, nil},
	}
	runEvalCases(t, cases)
}
