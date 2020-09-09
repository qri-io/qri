// Package preprocess is a SQL query preprocessor. It extracts a mapping of
// table references, converting illegal table names to legal SQL in the process.
//
// As an example the statemnent:
// SELECT * FROM illegal_sql/name
//
// will be converted to:
// SELECT * FROM illegal_sql_name as ds
// and return a map keying "illegal_name": "illegal/name"
//
// preprocess exists to cover the ways in which Qri deviates from the SQL spec,
// while reducing the burdern of maintaining a complete parser. This process
// intentionally avoids creating any AST. Think if it as glorified regex
package preprocess

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Query pulls dataset references from an SQL query, replacing dataset ref
// strings with legal SQL table names, and a mapping between the two
func Query(query string) (string, map[string]string, error) {
	p := newProcessor(strings.NewReader(query))
	err := p.Process()

	return p.processed.String(), p.mapping, err
}

// newProcessor allocates a processor from an io.Reader
func newProcessor(r io.Reader) *processor {
	return &processor{
		r:       bufio.NewReader(r),
		mapping: map[string]string{},
		aliases: map[string]struct{}{},
	}
}

// processor tokenizes an input stream
// TODO(b5): set position properly for errors
type processor struct {
	r         *bufio.Reader
	processed strings.Builder
	mapping   map[string]string

	autoAliased int
	aliases     map[string]struct{}

	// scanning state
	parseBuf  *token // parser token buffer, set by calling unscan
	scanBuf   *token // lexer token buffer populated by scanning
	text      strings.Builder
	nesting   int
	line, col int
	err       error
}

func (p *processor) Process() error {
	if err := p.processReferences(); err != nil {
		return err
	}

	str := p.processed.String()
	p.processed = strings.Builder{}
	p.r = bufio.NewReader(strings.NewReader(str))

	return p.processSelects()
}

func (p *processor) processReferences() error {
	nesting := p.nesting

	for {
		tok := p.scan()

		switch tok.Type {
		case fromTok, joinTok:
			p.processed.WriteString(tok.Text)
			if err := p.processTableRefs(); err != nil {
				return err
			}
		case leftParenTok:
			p.processed.WriteString(tok.Text)
			p.nesting++
			if err := p.processReferences(); err != nil {
				return err
			}
		case rightParenTok:
			p.processed.WriteString(tok.Text)
			p.nesting--
			if p.nesting == nesting {
				return nil
			}
		case eofTok:
			return nil
		default:
			p.processed.WriteString(tok.Text)
		}
	}
}

func (p *processor) processTableRefs() error {
	hasReadName := false
	nesting := p.nesting

	for {
		tok := p.scan()

		switch tok.Type {
		// check for table name references
		case textTok:
			if err := p.processTableRef(tok.Text); err != nil {
				return err
			}

		default:
			// default to writing any non-empty token text
			if tok.Text != "" {
				p.processed.WriteString(tok.Text)
			}

			switch tok.Type {
			case commaTok:
				if !hasReadName {
					return fmt.Errorf("encountered '%s' before table name", tok.Text)
				}
				hasReadName = false

			case leftParenTok:
				p.nesting++
				if err := p.processReferences(); err != nil {
					return err
				}
			case rightParenTok:
				p.nesting--
				if p.nesting == nesting {
					return nil
				}
				// pass an empty string to skip mapping, but still check for table alias
				p.processTableRef("")

			case asTok:
			case whereTok, groupTok, havingTok, limitTok, orderTok, unionTok:
				return nil
			case eofTok:
				return nil
			default:
			}
		}
	}
}

// processTableRef adds a reference to the map, writes a legal name, and
// adds a table alias if one does not exist
func (p *processor) processTableRef(text string) error {
	if text != "" {
		name := toLegalName(text)
		p.mapping[name] = text
		p.processed.WriteString(name)
	}

	alias := ""

	for {
		tok := p.scan()

		switch tok.Type {
		case whitespaceTok:
			p.processed.WriteString(tok.Text)
		case asTok:
			p.processed.WriteString(tok.Text)
		case textTok:
			alias = tok.Text
			if _, ok := p.aliases[alias]; ok {
				return fmt.Errorf("duplicate reference alias '%s'", alias)
			}
			p.aliases[alias] = struct{}{}
			p.processed.WriteString(tok.Text)
		default:
			if alias == "" {
				alias = p.newTableName()
				p.processed.WriteString(" ")
				p.processed.WriteString(alias)
			}
			switch tok.Type {
			case commaTok:
				p.processed.WriteString(tok.Text)
			default:
				p.unscan(tok)
			}
			return nil
		}
	}
}

func (p *processor) processSelects() error {
	for {
		tok := p.scan()
		switch tok.Type {
		case selectTok:
			p.processed.WriteString(tok.Text)
			if err := p.processSelectCols(); err != nil {
				return err
			}
		case eofTok:
			return nil
		default:
			p.processed.WriteString(tok.Text)
		}
	}
}

func (p *processor) processSelectCols() error {
	for {
		tok := p.scan()

		if tok.IsKeyword() {
			p.unscan(tok)
			return nil
		}

		switch tok.Type {
		case textTok:
			if err := p.processColumnReference(tok); err != nil {
				return err
			}
		case eofTok:
			return nil
		default:
			p.processed.WriteString(tok.Text)
		}
	}
}

func (p *processor) processColumnReference(tok token) error {

	if tok.IsNumeric() {
		p.processed.WriteString(tok.Text)
		return nil
	}

	colName := tok.Text
	switch strings.Count(colName, ".") {
	case 0:
		if len(p.aliases) != 1 {
			return fmt.Errorf("ambiguous column: %q, in statement that references multiple datasets. use aliases to distinguish datasets", colName)
		}
		alias := ""
		for col := range p.aliases {
			alias = col
		}

		p.processed.WriteString(fmt.Sprintf("%s.%s", alias, strings.Trim(colName, "'\"")))
	case 1:
		// column name contains an alias reference
		p.processed.WriteString(colName)
	default:
		return fmt.Errorf("illegal column reference: %q", colName)
	}
	return nil
}

func (p *processor) newTableName() string {
	for {
		p.autoAliased++
		alias := fmt.Sprintf("ds%d", p.autoAliased)
		if p.autoAliased == 1 {
			alias = "ds"
		}

		if _, ok := p.aliases[alias]; ok {
			continue
		}

		p.aliases[alias] = struct{}{}
		return alias
	}
}

func toLegalName(refStr string) string {
	refStr = strings.Replace(refStr, "@", "_at_", 1)
	refStr = strings.ReplaceAll(refStr, "/", "_")
	return strings.ReplaceAll(refStr, "-", "_")
}

// scan reads one token from the input stream
func (p *processor) scan() token {
	if p.parseBuf != nil {
		t := *p.parseBuf
		p.parseBuf = nil
		return t
	}

	if p.scanBuf != nil {
		t := *p.scanBuf
		p.scanBuf = nil
		return t
	}

	inText := false
	p.text.Reset()
	p.col = 0

	for {
		ch := p.read()
		p.col++

		switch ch {
		case '\r':
			// ignore line feeds
			continue
		case '\n':
			p.line++
			p.col = 0
			t := token{Type: whitespaceTok, Text: "\n"}
			if inText {
				p.scanBuf = &t
				return p.newTextualTok()
			}
			return t

		case '(':
			t := token{Type: leftParenTok, Text: "("}
			if inText {
				p.scanBuf = &t
				return p.newTextualTok()
			}
			return t
		case ')':
			t := token{Type: rightParenTok, Text: ")"}
			if inText {
				p.scanBuf = &t
				return p.newTextualTok()
			}
			return t
		case ',':
			t := token{Type: commaTok, Text: ","}
			if inText {
				p.scanBuf = &t
				return p.newTextualTok()
			}
			return t
		case '*':
			return token{Type: starTok, Text: "*"}
		case '\t':
			return token{Type: whitespaceTok, Text: "\t"}
		case ' ':
			t := token{Type: whitespaceTok, Text: " "}
			if inText {
				p.scanBuf = &t
				return p.newTextualTok()
			}
			return t

		case eof:
			t := token{Type: eofTok}
			if inText {
				p.scanBuf = &t
				return p.newTextualTok()
			}
			return t

		default:
			p.text.WriteRune(ch)
			inText = true
		}
	}
}

func (p *processor) unscan(t token) {
	p.parseBuf = &t
}

// read reads the next rune from the buffered reader.
// Returns the rune(0) if an error occurs (or io.EOF is returned).
func (p *processor) read() rune {
	ch, _, err := p.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

// newTok creates a new token from current processor state
func (p *processor) newTextualTok() token {
	// p.buf = buf
	t := token{
		Type: textTok,
		Text: strings.TrimSpace(p.text.String()),
		Pos:  position{Line: p.line, Col: p.col},
	}

	// identify keywords
	switch strings.ToLower(t.Text) {
	case selectTok.String():
		t.Type = selectTok
	case asTok.String():
		t.Type = asTok
	case byTok.String():
		t.Type = byTok
	case fromTok.String():
		t.Type = fromTok
	case groupTok.String():
		t.Type = groupTok
	case havingTok.String():
		t.Type = havingTok
	case joinTok.String():
		t.Type = joinTok
	case unionTok.String():
		t.Type = unionTok
	case limitTok.String():
		t.Type = limitTok
	case orderTok.String():
		t.Type = orderTok
	case whereTok.String():
		t.Type = whereTok
	}

	return t
}

// eof represents a marker rune for the end of the reader.
var eof = rune(0)

// token is a recognized token from the outlineline lexicon
type token struct {
	Type tokenType
	Pos  position
	Text string
}

func (t token) IsKeyword() bool {
	return t.Type > keywordBegin && t.Type < keywordEnd
}

func (t token) IsLiteral() bool {
	return t.Type > literalBegin && t.Type < literalEnd
}

func (t token) IsNumeric() bool {
	_, err := strconv.ParseFloat(t.Text, 64)
	return err == nil
}

// String implements the stringer interface for token
func (t token) String() string {
	return t.Text
}

type position struct {
	Line, Col int
}

// tokenType enumerates the different types of tokens
type tokenType int

const (
	// IllegalTok is the default for unrecognized tokens
	IllegalTok tokenType = iota
	eofTok

	literalBegin  // marks the beginning of literal tokens in the token enumeration
	indentTok     // tab character "\t" or two consecutive spaces"  "
	newlineTok    // line break
	textTok       // a token for arbitrary text
	commaTok      // a "," character
	leftParenTok  // "("
	rightParenTok // ")"
	starTok       // "*"
	whitespaceTok // spaces, tabs, newlines
	quoteTok      // a single quote: '
	literalEnd    // marks the end of literal tokens in the token enumeration

	keywordBegin // keywordBegin marks the start of SQL keyword tokens in the token enumeration
	selectTok
	asTok
	byTok
	fromTok
	groupTok
	havingTok
	joinTok
	limitTok
	orderTok
	unionTok
	whereTok
	keywordEnd // keywordEnd marks the end of keyword tokens in the token enumeration
)

func (t tokenType) String() string {
	switch t {
	case commaTok:
		return "comma"
	case textTok:
		return "text"
	case whitespaceTok:
		return "WS"

	case starTok:
		return "*"
	case selectTok:
		return "select"
	case asTok:
		return "as"
	case byTok:
		return "by"
	case fromTok:
		return "from"
	case groupTok:
		return "group"
	case havingTok:
		return "having"
	case joinTok:
		return "join"
	case limitTok:
		return "limit"
	case orderTok:
		return "order"
	case unionTok:
		return "union"
	case whereTok:
		return "where"
	case eofTok:
		return "EOF"

	default:
		return "unknown"
	}
}
