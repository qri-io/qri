package dataset_sql

import (
	"fmt"
	"github.com/ipfs/go-datastore"

	"github.com/qri-io/dataset"
	q "github.com/qri-io/dataset_sql/vt/proto/query"
)

func Prepare(ds *dataset.Dataset, opts *ExecOpt) (Statement, map[string]datastore.Key, error) {
	concreteStmt, err := Parse(ds.QueryString)
	if err != nil {
		return nil, nil, err
	}

	err = RemoveUnusedReferences(concreteStmt, ds)
	if err != nil {
		return nil, nil, err
	}

	strs := map[string]*dataset.Structure{}
	for name, ds := range ds.Resources {
		strs[name] = ds.Structure
	}

	ds.Structure, err = ResultStructure(concreteStmt, strs, opts)
	if err != nil {
		return nil, nil, err
	}

	queryString, stmt, remap, err := Format(ds)
	if err != nil {
		return nil, nil, err
	}

	// TODO - turn this on once we have client-side formatting
	// ds.QueryString = queryString

	paths := map[string]datastore.Key{}
	// collect table references
	for mapped, ref := range remap {
		// for i, adr := range stmt.References() {
		if ds.Resources[ref] == nil {
			return nil, nil, fmt.Errorf("couldn't find resource for table name: %s", ref)
		}
		paths[mapped] = ds.Resources[ref].Data
		ds.Query.Structures[mapped] = ds.Resources[ref].Structure.Abstract()
	}

	ds.Query.Syntax = "sql"
	ds.Query.Statement = queryString
	ds.Query.Structure, err = ResultStructure(stmt, ds.Query.Structures, opts)
	if err != nil {
		return nil, nil, err
	}

	err = PrepareStatement(stmt, ds.Query.Structures)
	return stmt, paths, err
}

// PrepareStatement sets up a statement for exectution. It modifies the passed-in statement
// making optimizations, associating type information from resources, etc.
func PrepareStatement(stmt Statement, resources map[string]*dataset.Structure) (err error) {
	err = fitASTResources(stmt, resources)
	if err != nil {
		return
	}
	return populateTableInfo(stmt, resources)
}

// fitASTResources removes star expressions, replacing them with concrete colIdent
// pointers extracted from resources. It's important that no extraneous tables
// are in the resources map
func fitASTResources(ast SQLNode, resources map[string]*dataset.Structure) error {
	var visit func(node SQLNode) func(node SQLNode) (bool, error)
	visit = func(parent SQLNode) func(node SQLNode) (bool, error) {
		return func(child SQLNode) (bool, error) {
			if child == nil {
				return false, nil
			}

			switch node := child.(type) {
			case *StarExpr:
				if node != nil {
					se, err := starExprSelectExprs(node, resources)
					if err != nil {
						return false, err
					}
					return false, replaceSelectExprs(parent, node, se)
				}
			}
			return true, nil
		}
	}

	return ast.WalkSubtree(visit(ast))
}

func starExprSelectExprs(star *StarExpr, resources map[string]*dataset.Structure) (SelectExprs, error) {
	name := star.TableName.String()
	for tableName, resourceData := range resources {
		// we add fields if the names match, or if no name is specified
		if name == "" || tableName == name {
			se := make(SelectExprs, len(resourceData.Schema.Fields))
			for i, f := range resourceData.Schema.Fields {
				col := &ColName{
					Name:      NewColIdent(f.Name),
					Qualifier: TableName{Name: NewTableIdent(tableName)},
					Metadata: StructureRef{
						TableName: tableName,
						Field:     f,
						ColIndex:  i,
					},
				}
				se[i] = &AliasedExpr{As: NewColIdent(f.Name), Expr: col}
			}
			return se, nil
		}
	}
	return nil, fmt.Errorf("couldn't find table for star expression: '%s'", name)
}

func replaceSelectExprs(parent, prev SQLNode, se SelectExprs) error {
	switch node := parent.(type) {
	case *Select:
		for i, exp := range node.SelectExprs {
			if exp == prev {
				node.SelectExprs = spliceSelectExprs(node.SelectExprs, se, i)
				return nil
			}
		}
	}
	return fmt.Errorf("couldn't find selectExprs for parent")
}

func spliceSelectExprs(a, b SelectExprs, pos int) SelectExprs {
	return append(a[:pos], append(b, a[pos+1:]...)...)
}

// StructureRef is placed on ColName SQLNodes to
// connect typing & data lookup information
type StructureRef struct {
	TableName string
	ColIndex  int
	Field     *dataset.Field
	QueryType q.Type
}

// PopulateAST adds type information & data lookup locations to an AST
// for a given resource.
// TODO - column ambiguity check
func populateTableInfo(tree SQLNode, resources map[string]*dataset.Structure) error {
	return tree.WalkSubtree(func(node SQLNode) (bool, error) {
		if col, ok := node.(*ColName); ok && node != nil {
			if col.Qualifier.TableName() != "" && resources[col.Qualifier.TableName()] != nil {
				for i, f := range resources[col.Qualifier.TableName()].Schema.Fields {
					if col.Name.String() == f.Name {
						qt := queryDatatypeForDataType(f.Type)
						if qt == q.Type_NULL_TYPE {
							return false, fmt.Errorf("unsupported datatype for colname evaluation: %s", f.Type.String())
						}
						col.Metadata = StructureRef{
							Field:     f,
							TableName: col.Qualifier.TableName(),
							ColIndex:  i,
							QueryType: qt,
						}
						return true, nil
					}
				}
				return false, fmt.Errorf("couldn't find field named '%s' in dataset '%s'", col.Name.String(), col.Qualifier.TableName())
			} else {
				for tableName, st := range resources {
					for i, f := range st.Schema.Fields {
						if col.Name.String() == f.Name {
							col.Qualifier = TableName{Name: NewTableIdent(tableName)}
							qt := queryDatatypeForDataType(f.Type)
							if qt == q.Type_NULL_TYPE {
								return false, fmt.Errorf("unsupported datatype for colname evaluation: %s", f.Type.String())
							}
							col.Metadata = StructureRef{
								Field:     f,
								TableName: tableName,
								QueryType: qt,
								ColIndex:  i,
							}
							return true, nil
						}
					}
				}
				return false, fmt.Errorf("couldn't find field named '%s' in any of the specified datasets", col.Name.String())
			}
		}
		return true, nil
	})
}
