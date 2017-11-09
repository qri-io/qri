package dataset_sql

import (
	"bytes"
	"fmt"
	q "github.com/qri-io/dataset_sql/vt/proto/query"
)

// map bool to a unsigned 8 bit int
const QueryBoolType = q.Type_UINT8

func (node *AndExpr) Eval() (q.Type, []byte, error) {
	lt, lb, err := node.Left.Eval()
	if err != nil {
		return QueryBoolType, falseB, err
	}
	if lt != QueryBoolType {
		err = fmt.Errorf("non-boolean expression for left side of AND clause")
		return QueryBoolType, falseB, err
	}
	if !bytes.Equal(lb, trueB) {
		return QueryBoolType, falseB, nil
	}

	rt, rb, err := node.Right.Eval()
	if err != nil {
		return QueryBoolType, falseB, err
	}
	if rt != QueryBoolType {
		err = fmt.Errorf("non-boolean expression for right side of AND clause")
		return QueryBoolType, falseB, err
	}
	if !bytes.Equal(rb, trueB) {
		return QueryBoolType, falseB, nil
	}

	return QueryBoolType, trueB, nil
}

func (node *OrExpr) Eval() (q.Type, []byte, error) {
	lt, lb, err := node.Left.Eval()
	if err != nil {
		return QueryBoolType, falseB, err
	}
	if lt != QueryBoolType {
		err = fmt.Errorf("non-boolean expression for left side of AND clause: %s", String(node))
		return QueryBoolType, falseB, err
	}
	if bytes.Equal(lb, trueB) {
		return QueryBoolType, trueB, nil
	}

	rt, rb, err := node.Right.Eval()
	if err != nil {
		return QueryBoolType, falseB, err
	}
	if rt != QueryBoolType {
		err = fmt.Errorf("non-boolean expression for right side of AND clause: %s", String(node))
		return QueryBoolType, falseB, err
	}
	if bytes.Equal(rb, trueB) {
		return QueryBoolType, trueB, nil
	}

	return QueryBoolType, falseB, nil
}

func (node *NotExpr) Eval() (q.Type, []byte, error) {
	t, b, e := node.Expr.Eval()
	if t != QueryBoolType {
		e = fmt.Errorf("non-boolean expression for NOT expression: %s", String(node))
		return q.Type_NULL_TYPE, nil, e
	}
	if bytes.Equal(trueB, b) {
		return QueryBoolType, falseB, nil
	}
	// TODO - strange byte responses
	return QueryBoolType, trueB, nil
}

// TODO - finish
func (node *ParenExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval ParenExpr")
}

func (node *ComparisonExpr) Eval() (q.Type, []byte, error) {
	result, err := node.Compare()
	if err != nil {
		return q.Type_NULL_TYPE, nil, err
	}
	if result {
		return QueryBoolType, trueB, nil
	}
	return QueryBoolType, falseB, nil
}

// TODO - finish
func (node *RangeCond) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval RangeCond")
}

// TODO - finish
func (node *IsExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval IsExpr")
}

// TODO - finish
func (node *ExistsExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval ExistsExpr")
}

func (node *SQLVal) Eval() (q.Type, []byte, error) {
	var t q.Type
	switch node.Type {
	case StrVal:
		t = q.Type_TEXT
	case IntVal:
		t = q.Type_INT64
	case FloatVal:
		t = q.Type_FLOAT32
	case HexNum:
		t = q.Type_BINARY
	case HexVal:
		t = q.Type_BLOB
	case ValArg:
		// TODO - is this correct?
		t = q.Type_EXPRESSION
	}
	return t, node.Val, nil
}

func (node *NullVal) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, nil
}

func (node BoolVal) Eval() (q.Type, []byte, error) {
	if bool(node) == true {
		return QueryBoolType, trueB, nil
	}
	return QueryBoolType, falseB, nil
}

// Eval evaluates the node against a row of data
func (node *ColName) Eval() (q.Type, []byte, error) {
	// if !node.Metadata {
	// 	return q.Type_NULL_TYPE, nil, fmt.Errorf("col ref %s hasn't been populated with structural information", node.Name.String())
	// }
	return node.Metadata.QueryType, node.Value, nil
}

func (node ValTuple) Eval() (q.Type, []byte, error) {
	// TODO - huh?
	return q.Type_NULL_TYPE, nil, NotYetImplemented("val tuple Eval")
}

// TODO - finish
func (node *Subquery) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Subquery")
}

func (node ListArg) Eval() (q.Type, []byte, error) {
	// TODO - huh?
	return q.Type_NULL_TYPE, node, nil
}

// TODO - finish
func (node *BinaryExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval BinaryExpr")
}

func (node *UnaryExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval UnaryExpr")
}

// TODO - finish
func (node *IntervalExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval IntervalExpr")
}

// TODO - finish
func (node *CollateExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval CollateExpr")
}

// TODO - finish
func (node *FuncExpr) Eval() (q.Type, []byte, error) {
	return node.fn.Eval()
}

// TODO - finish
func (node *GroupConcatExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval GroupConcatExpr")
}

func (node *ValuesFuncExpr) Eval() (q.Type, []byte, error) {
	if node.Resolved == nil {
		return q.Type_NULL_TYPE, nil, fmt.Errorf("invalid values expression: %s", String(node))
	}
	return node.Resolved.Eval()
}

// TODO - finish
func (node *ConvertExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval ConvertExpr")
}

// TODO - finish
func (node *ConvertUsingExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval ConvertUsingExpr")
}

// TODO - finish
func (node *MatchExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval MatchExpr")
}

// TODO - finish
func (node *CaseExpr) Eval() (q.Type, []byte, error) {
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval finish")
}

func (node *Where) Eval() (q.Type, []byte, error) {
	if node == nil {
		return QueryBoolType, trueB, nil
	}
	return node.Expr.Eval()
}

func (node *AliasedExpr) Eval() (q.Type, []byte, error) {
	return node.Expr.Eval()
}

func (node TableName) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval TableName")
}

func (node OrderBy) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval OrderBy")
}

func (node GroupBy) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval GroupBy")
}

func (node TableExprs) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval TableExprs")
}

func (node *Set) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Set")
}

func (node Comments) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Comments")
}

func (node SelectExprs) Eval() (q.Type, []byte, error) {
	if len(node) == 1 {
		return node[0].Eval()
	}
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval SelectExprs")
}

func (node *Limit) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Limit")
}

func (node Columns) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Columns")
}

func (node OnDup) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval OnDup")
}

func (node TableNames) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval TableNames")
}

func (node UpdateExprs) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval UpdateExprs")
}

func (node TableIdent) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval TableIdent")
}

func (node ColIdent) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval ColIdent")
}

func (node IndexHints) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval IndexHints")
}

func (node Exprs) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Exprs")
}

func (node *ConvertType) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval ConvertType")
}

func (node *When) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval When")
}

func (node *Order) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Order")
}

func (node *UpdateExpr) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval UpdateExpr")
}

func (node *StarExpr) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval StarExpr")
}

func (node Nextval) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Nextval")
}

func (node *Select) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Select")
}

func (node *AliasedTableExpr) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval AliasedTableExpr")
}

func (node *ParenTableExpr) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval ParenTableExpr")
}

func (node *JoinTableExpr) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval JoinTableExpr")
}

func (node *Union) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Union")
}

func (node *ParenSelect) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval ParenSelect")
}

func (node *Insert) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Insert")
}

func (node *Delete) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Delete")
}

func (node *Update) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Update")
}

func (node *DDL) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval DDL")
}

func (node Values) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Values")
}

func (node *Show) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Show")
}

func (node *Use) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Use")
}

func (node *OtherRead) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval OtherRead")
}

func (node *OtherAdmin) Eval() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval OtherAdmin")
}
