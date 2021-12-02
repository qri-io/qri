package staticlark

import (
	"fmt"
	"strings"
	"unicode"

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
	// atom only
	if u.tail == nil {
		if u.todo {
			return fmt.Sprintf("TODO:%s", u.atom)
		}
		return u.atom
	}
	// function with no args
	if len(u.tail) == 0 {
		if u.todo {
			return fmt.Sprintf("TODO:[%s]", u.atom)
		}
		return fmt.Sprintf("[%s]", u.atom)
	}
	// function with args
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
	startIndex := 1
	if u.atom == "set!" {
		startIndex = 2
	}
	if len(u.tail) < startIndex {
		return []string{}
	}
	res := []string{}
	for i, t := range u.tail {
		if i < (startIndex - 1) {
			continue
		}
		res = append(res, t.getSources()...)
	}
	return res
}

func (u *unit) getSources() []string {
	if len(u.tail) == 0 {
		if isIdent(u.atom) {
			return []string{u.atom}
		}
		return []string{}
	}
	res := []string{}
	for _, t := range u.tail {
		res = append(res, t.getSources()...)
	}
	return res
}

type invoke struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

func (u *unit) Invocations() []invoke {
	if u.atom == "set!" {
		return u.tail[1].Invocations()
	}
	if u.atom == "return" {
		return u.invokeFromList()
	}
	if isIdent(u.atom) && u.tail != nil {
		name := u.atom
		args := u.getSources()
		inv := invoke{Name: name, Args: args}
		return append([]invoke{inv}, u.invokeFromList()...)
	}
	return u.invokeFromList()
}

func (u *unit) invokeFromList() []invoke {
	if u.tail == nil {
		return []invoke{}
	}
	var res []invoke
	for _, t := range u.tail {
		res = append(res, t.Invocations()...)
	}
	if res == nil {
		return []invoke{}
	}
	return res
}

func (u *unit) AssignsTo() string {
	if u.atom == "set!" {
		return u.tail[0].atom
	}
	return ""
}

func (u *unit) IsReturn() bool {
	return u.atom == "return"
}

func toUnitTODO(text string) *unit {
	return &unit{atom: text, todo: true}
}

func isIdent(text string) bool {
	return unicode.IsLetter([]rune(text)[0])
}
