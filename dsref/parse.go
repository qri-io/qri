package dsref

import (
	"fmt"
	"regexp"
	"unicode"
)

// These functions parse a string to create a dsref. We refer to a "human-friendly reference"
// as one with only a username and dataset name, such as "my_user/my_dataset". A "full reference"
// can also contain a "concrete reference", which includes an optional profileID plus a network
// and a "commit hash".
//
// Parse will parse a human-friendly reference, or only a concrete reference, or a full reference.
//
// ParseHumanFriendly will only successfully parse a human-friendly reference, and nothing else.
//
// The grammar is here:
//
//  <dsref> = <humanFriendlyPortion> [ <concreteRef> ] | <concreteRef>
//  <humanFriendlyPortion> = <validName> '/' <validName>
//  <concretePath> = '@' [ <profileID> ] '/' <network> '/' <commitHash>
//
// Some examples of valid references:
//     me/dataset
//     username/dataset
//     @/ipfs/QmSome1Commit2Hash3
//     @QmProfile4ID5/ipfs/QmSome1Commit2Hash3
//     username/dataset@QmProfile4ID5/ipfs/QmSome1Commit2Hash3
// An invalid reference:
//     /ipfs/QmSome1Commit2Hash3

const (
	alphaNumeric       = `[a-zA-Z][\w-]*`
	alphaNumericDsname = `[a-zA-Z][\w-]{0,143}`
	b58Id              = `Qm[0-9a-zA-Z]{0,44}`
)

var (
	validName      = regexp.MustCompile(`^` + alphaNumeric)
	dsNameCheck    = regexp.MustCompile(`^` + alphaNumericDsname + `$`)
	concretePath   = regexp.MustCompile(`^@(` + b58Id + `)?\/(` + alphaNumeric + `)\/(` + b58Id + `)`)
	b58StrictCheck = regexp.MustCompile(`^Qm[1-9A-HJ-NP-Za-km-z]*$`)

	// ErrEmptyRef is an error for when a reference is empty
	ErrEmptyRef = fmt.Errorf("empty reference")
	// ErrParseError is an error returned when parsing fails
	ErrParseError = fmt.Errorf("could not parse ref")
	// ErrUnexpectedChar is an error when a character is unexpected, topic string must be non-empty
	ErrUnexpectedChar = fmt.Errorf("unexpected character")
	// ErrNotHumanFriendly is an error returned when a reference is not human-friendly
	ErrNotHumanFriendly = fmt.Errorf("unexpected character '@', ref can only have username/name")
	// ErrBadCaseName is the error when a bad case is used in the dataset name
	ErrBadCaseName = fmt.Errorf("dataset name may not contain any upper-case letters")
	// ErrBadCaseUsername is for when a username contains upper-case letters
	ErrBadCaseUsername = fmt.Errorf("username may not contain any upper-case letters")
	// ErrBadCaseShouldRename is the error when a dataset should be renamed to not use upper case letters
	ErrBadCaseShouldRename = fmt.Errorf("dataset name should not contain any upper-case letters, rename it to only use lower-case letters, numbers, and underscores")
	// ErrDescribeValidName is an error describing a valid dataset name
	ErrDescribeValidName = fmt.Errorf("dataset name must start with a lower-case letter, and only contain lower-case letters, numbers, dashes, and underscore. Maximum length is 144 characters")
	// ErrDescribeValidUsername describes valid username
	ErrDescribeValidUsername = fmt.Errorf("username must start with a lower-case letter, and only contain lower-case letters, numbers, dashes, and underscores")
)

// Parse a reference from a string
func Parse(text string) (Ref, error) {
	var r Ref
	origLength := len(text)
	if origLength == 0 {
		return r, ErrEmptyRef
	}

	remain, partial, err := parseHumanFriendlyPortion(text)
	if err == nil {
		text = remain
		r.Username = partial.Username
		r.Name = partial.Name
	} else if err == ErrUnexpectedChar {
		// This error must only be returned when the topic string is non-empty, so it's safe to
		// index it at position 0.
		return r, NewParseError("%s at position %d: '%c'", err, len(text)-len(remain), remain[0])
	} else if err != ErrParseError {
		return r, err
	}

	remain, partial, err = parseConcretePath(text)
	if err == nil {
		text = remain
		r.ProfileID = partial.ProfileID
		r.Path = partial.Path
	} else if err != ErrParseError {
		return r, err
	}

	if text != "" {
		pos := origLength - len(text)
		return r, NewParseError("unexpected character at position %d: '%c'", pos, text[0])
	}

	// Dataset names are not supposed to contain upper-case characters. For now, return an error
	// but also the reference; callers should display a warning, but continue to work for now.
	for _, rune := range r.Name {
		if unicode.IsUpper(rune) {
			return r, ErrBadCaseName
		}
	}

	return r, nil
}

// ParseHumanFriendly parses a reference that only has a username and a dataset name
func ParseHumanFriendly(text string) (Ref, error) {
	var r Ref
	origLength := len(text)
	if origLength == 0 {
		return r, ErrEmptyRef
	}

	remain, partial, err := parseHumanFriendlyPortion(text)
	if err == nil {
		text = remain
		r.Username = partial.Username
		r.Name = partial.Name
	} else if err != ErrParseError {
		return r, err
	}

	if text != "" {
		if text[0] == '@' {
			return r, ErrNotHumanFriendly
		}
		pos := origLength - len(text)
		return r, NewParseError("unexpected character at position %d: '%c'", pos, text[0])
	}

	// Dataset names are not supposed to contain upper-case characters. For now, return an error
	// but also the reference; callers should display a warning, but continue to work for now.
	for _, rune := range r.Name {
		if unicode.IsUpper(rune) {
			return r, ErrBadCaseName
		}
	}

	return r, nil
}

// ParseError is an error for when a dataset reference fails to parse
type ParseError struct {
	Message string
}

// Error renders the ParseError as a string
func (e *ParseError) Error() string {
	return e.Message
}

// NewParseError returns a new ParseError, its parameters are a format string and arguments
func NewParseError(template string, args ...interface{}) error {
	return &ParseError{Message: fmt.Sprintf(template, args...)}
}

// MustParse parses a dsref from a string, or panics if it fails
func MustParse(text string) Ref {
	ref, err := Parse(text)
	if err != nil {
		panic(err)
	}
	return ref
}

// IsRefString returns whether the string parses as a valid reference
func IsRefString(text string) bool {
	_, err := Parse(text)
	return err == nil || err == ErrBadCaseName
}

// IsValidName returns whether the dataset name is valid
func IsValidName(text string) bool {
	return dsNameCheck.Match([]byte(text))
}

// EnsureValidName returns nil if the name is valid, and an error otherwise
func EnsureValidName(text string) error {
	if !dsNameCheck.Match([]byte(text)) {
		return ErrDescribeValidName
	}

	// Dataset names are not supposed to contain upper-case characters. For now, return an error
	// but also the reference; callers should display a warning, but continue to work for now.
	for _, r := range text {
		if unicode.IsUpper(r) {
			return ErrBadCaseName
		}
	}

	return nil
}

// EnsureValidUsername is the same as EnsureValidName but returns a different error
func EnsureValidUsername(text string) error {
	err := EnsureValidName(text)
	if err == ErrDescribeValidName {
		return ErrDescribeValidUsername
	}
	if err == ErrBadCaseName {
		return ErrBadCaseUsername
	}
	return err
}

// parse the front of a dataset reference, the human friendly portion
func parseHumanFriendlyPortion(text string) (string, Ref, error) {
	var r Ref
	// Parse as many alphaNumeric characters as possible for the username
	match := validName.FindString(text)
	if match == "" {
		return text, r, ErrParseError
	}
	r.Username = match
	text = text[len(match):]
	// Check if the remaining text is empty, or there's not a slash next
	if text == "" {
		return text, r, NewParseError("need username separated by '/' from dataset name")
	} else if text[0] != '/' {
		return text, r, ErrUnexpectedChar
	}
	text = text[1:]
	// Parse as many alphaNumeric characters as possible for the dataset name
	match = validName.FindString(text)
	if match == "" {
		return text, r, NewParseError("did not find valid dataset name")
	}
	r.Name = match
	text = text[len(match):]
	return text, r, nil
}

// parse the back of the dataset reference, the concrete path
func parseConcretePath(text string) (string, Ref, error) {
	var r Ref
	matches := concretePath.FindStringSubmatch(text)
	if matches == nil {
		return text, r, ErrParseError
	}
	if len(matches) != 4 {
		return text, r, NewParseError("unexpected number of regex matches %d", len(matches))
	}
	if matches[2] != "mem" && matches[2] != "ipfs" {
		return text, r, NewParseError("invalid network")
	}
	matchedLen := len(matches[0])
	if matches[1] != "" && b58StrictCheck.FindString(matches[1]) == "" {
		return text, r, NewParseError("profileID contains invalid base58 characters")
	}
	r.ProfileID = matches[1]
	if matches[3] != "" && b58StrictCheck.FindString(matches[3]) == "" {
		return text, r, NewParseError("path contains invalid base58 characters")
	}
	r.Path = fmt.Sprintf("/%s/%s", matches[2], matches[3])
	return text[matchedLen:], r, nil
}
