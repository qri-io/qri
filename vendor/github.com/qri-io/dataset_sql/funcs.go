package dataset_sql

import (
	"fmt"

	"github.com/qri-io/dataset/datatypes"
	q "github.com/qri-io/dataset_sql/vt/proto/query"
)

type AggFunc interface {
	Eval() (q.Type, []byte, error)
	Value() []byte
}

// AggregateFuncs extracts a slice of any aggregate functions from an AST, while also writing
// pointers to newly-generated funcs to the AST
func AggregateFuncs(root Statement) (funcs []AggFunc, err error) {
	err = root.WalkSubtree(func(node SQLNode) (bool, error) {
		switch t := node.(type) {
		case *FuncExpr:
			fn, err := t.Function()
			if err != nil {
				return false, err
			}
			funcs = append(funcs, fn)
		}
		return true, nil
	})
	return
}

// Function gives the backing function to perform
func (node *FuncExpr) Function() (fn AggFunc, err error) {
	fn, err = node.newAggFunc(node.Name.Lowered())
	if err != nil {
		return
	}
	node.fn = fn
	return fn, nil
}

func (node *FuncExpr) Datatype() datatypes.Type {
	switch node.Name.Lowered() {
	case "sum", "avg", "count", "max", "min":
		return datatypes.Float
	}
	return datatypes.Any
}

type numericAggFunc interface {
	Eval(val float32)
	Value() float32
}

// TODO
// type AggFuncBitAnd struct{ Exprs SelecExprs value float32 }
// type AggFuncBitOr struct{ Exprs SelecExprs value float32 }
// type AggFuncBitXor struct{ Exprs SelecExprs value float32 }
// type AggFuncGroupConcat struct{ Exprs SelecExprs value float32 }
// type AggFuncStd struct{ Exprs SelecExprs value float32 }
// type AggFuncStddevPop struct{ Exprs SelecExprs value float32 }
// type AggFuncStddevSamp struct{ Exprs SelecExprs value float32 }
// type AggFuncStddev struct{ Exprs SelecExprs value float32 }
// type AggFuncVarPop struct{ Value float32 }
// type AggFuncVarSamp struct{ Value float32 }
// type AggFuncVariance struct{ Value float32 }

func (node *FuncExpr) newAggFunc(name string) (AggFunc, error) {
	// if !datatypes.EachNumeric(node.Exprs.FieldTypes(from)) {
	// 	return nil, fmt.Errorf("sum only works with numeric fields")
	// }

	if len(node.Exprs) != 1 {
		return nil, fmt.Errorf("too many arguments for aggregate function: %s", name)
	}

	var fn numericAggFunc
	switch name {
	case "sum":
		fn = &sumFunc{}
	case "avg":
		fn = &avgFunc{}
	case "count":
		fn = &countFunc{}
	case "max":
		fn = &maxFunc{}
	case "min":
		fn = &minFunc{}
	default:
		return nil, fmt.Errorf("unrecognized aggregate function: %s", node.Name)
	}

	return &aggFunc{Name: name, Exprs: node.Exprs, fn: fn}, nil
}

type aggFunc struct {
	Name  string
	Exprs SelectExprs
	fn    numericAggFunc
}

func (af *aggFunc) Datatype() datatypes.Type {
	return datatypes.Float
}

func (af *aggFunc) Eval() (q.Type, []byte, error) {
	ts, vs, err := af.Exprs.Values()
	if err != nil {
		return q.Type_NULL_TYPE, nil, err
	}

	var v float32
	for i, val := range vs {
		if len(val) == 0 {
			if af.Name != "sum" {
				af.fn.Eval(0)
			}
			continue
		}

		switch ts[i] {
		case q.Type_INT64:
			value, err := datatypes.ParseInteger(val)
			if err != nil {
				return q.Type_NULL_TYPE, nil, fmt.Errorf("invalid integer: '%s'", string(val))
			}
			v = float32(value)
		case q.Type_FLOAT32:
			value, err := datatypes.ParseFloat(val)
			if err != nil {
				return q.Type_NULL_TYPE, nil, fmt.Errorf("invalid float: '%s'", string(val))
			}
			v = float32(value)
		}
		af.fn.Eval(v)
	}

	return q.Type_FLOAT32, nil, nil
}

func (af *aggFunc) Value() []byte {
	// fmt.Println(af.Name, af.fn.Value())
	val, err := datatypes.Float.ValueToBytes(af.fn.Value())
	if err != nil {
		return nil
	}
	return val
}

// func readInt(data []byte) (int64, error) {
// 	return binary.ReadVarint(bytes.NewBuffer(data))
// }

// func readFloat32(data []byte) (float32, error) {
// 	f64, err := strconv.ParseFloat(string(data), 32)
// 	return float32(f64), err
// }

type avgFunc struct {
	count int
	total float32
}

func (a *avgFunc) Eval(val float32) {
	a.count++
	a.total += val
}
func (a avgFunc) Value() float32 { return a.total / float32(a.count) }

type sumFunc struct{ total float32 }

func (a *sumFunc) Eval(val float32) { a.total += val }
func (a sumFunc) Value() float32    { return a.total }

type countFunc struct{ count int }

func (a *countFunc) Eval(val float32) { a.count++ }
func (a countFunc) Value() float32    { return float32(a.count) }

type maxFunc struct{ max float32 }

func (a *maxFunc) Eval(val float32) {
	if val > a.max {
		a.max = val
	}
}
func (a maxFunc) Value() float32 { return a.max }

type minFunc struct{ min float32 }

func (a *minFunc) Eval(val float32) {
	if val < a.min {
		a.min = val
	}
}
func (a minFunc) Value() float32 { return a.min }

func (nodes SelectExprs) Values() (types []q.Type, vals [][]byte, err error) {
	for _, se := range nodes {
		switch node := se.(type) {
		case *StarExpr:
			ts, vs, e := node.Values()
			if e != nil {
				err = e
				return
			}
			types = append(types, ts...)
			vals = append(vals, vs...)
		case *AliasedExpr:
			t, v, e := node.Expr.Eval()
			if e != nil {
				err = e
				return
			}
			types = append(types, t)
			vals = append(vals, v)
		case Nextval:
			t, v, e := node.Value()
			if e != nil {
				err = e
				return
			}
			types = append(types, t)
			vals = append(vals, v)
		}
	}
	return
}

func (node *StarExpr) Values() ([]q.Type, [][]byte, error) {
	return []q.Type{q.Type_NULL_TYPE}, nil, NotYetImplemented("star expession values")
}

func (node *Nextval) Value() (q.Type, []byte, error) {
	// TODO
	return q.Type_NULL_TYPE, nil, NotYetImplemented("eval Nextval")
}
