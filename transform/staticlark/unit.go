package staticlark

import (
	"fmt"
	"strconv"
	"strings"

	"go.starlark.net/syntax"
)

// unit of code, roughly equivalent to a single line
// can be an assignment, function call, or control structure such as `if` or `for`
// represented as a lisp-like AST
type unit struct {
	atom  string
	tail  []*unit
	todo  bool
	where syntax.Position
}

// String converts a unit into a prefix notation string
func (u *unit) String() string {
	if len(u.tail) == 0 {
		if u.todo {
			return fmt.Sprintf("TODO:%s", u.atom)
		}
		return u.atom
	}
	res := make([]string, len(u.tail))
	for i, node := range u.tail {
		res[i] = node.String()
	}
	if u.todo {
		return fmt.Sprintf("TODO:[%s %s]", u.atom, strings.Join(res, " "))
	}
	return fmt.Sprintf("[%s %s]", u.atom, strings.Join(res, " "))
}

// Push adds to the tail of the unit
func (u *unit) Push(text string) {
	u.tail = append(u.tail, &unit{atom: text})
}

// return all of the sources of data, as lexical tokens, that affect
// the calculation of this unit's value
func (u *unit) DataSources() []string {
	if len(u.tail) == 0 {
		if _, err := strconv.Atoi(u.atom); err == nil {
			// if a number, return no identifiers
			return []string{}
		}
		return []string{u.atom}
	}
	res := []string{}
	for _, t := range u.tail {
		res = append(res, t.DataSources()...)
	}
	return res
}

func toUnitTODO(text string) *unit {
	return &unit{atom: text, todo: true}
}
