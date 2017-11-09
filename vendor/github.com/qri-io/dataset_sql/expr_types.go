package dataset_sql

import (
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
)

func (nodes SelectExprs) FieldTypes(from map[string]*dataset.Structure) (types []datatypes.Type) {
	for _, se := range nodes {
		switch node := se.(type) {
		case *StarExpr:
			types = append(types, node.FieldTypes(from)...)
		case *AliasedExpr:
			types = append(types, node.FieldType(from))
		case Nextval:
			types = append(types, node.FieldType(from))
		default:
		}
	}
	return
}

func (node *StarExpr) FieldTypes(from map[string]*dataset.Structure) (types []datatypes.Type) {
	// TODO - test this
	if tbl := from[node.TableName.String()]; tbl != nil {
		types = make([]datatypes.Type, len(tbl.Schema.Fields))
		for i, f := range tbl.Schema.Fields {
			types[i] = f.Type
		}
	}
	return
}

func (node Nextval) FieldType(from map[string]*dataset.Structure) datatypes.Type {
	// TODO
	return datatypes.Any
}

// FieldType returns a string representation of the type of field
// where datatype is one of: "", "string", "integer", "float", "boolean", "date"
// TODO - this may need rethinking.
func (node *AliasedExpr) FieldType(from map[string]*dataset.Structure) datatypes.Type {
	switch n := node.Expr.(type) {
	case *ColName:
		colName := node.Expr.(*ColName)
		for _, resourceData := range from {
			for _, f := range resourceData.Schema.Fields {
				// fmt.Println(name.Name.String(), f.Name)
				if colName.Name.String() == f.Name {
					return f.Type
				}
			}
		}
		return datatypes.Unknown
	// case BoolExpr:
	//  return datatypes.Boolean
	case *NullVal:
		return datatypes.Any
	case *SQLVal:
		switch n.Type {
		case StrVal:
			return datatypes.String
		case FloatVal:
			return datatypes.Float
		case IntVal:
			return datatypes.Integer
		case HexNum:
			// TODO - this is wrong
			return datatypes.String
		case HexVal:
			return datatypes.String
		case ValArg:
			// TODO - this is probably wrong
			return datatypes.Any
		}
	}

	return datatypes.Unknown
}
