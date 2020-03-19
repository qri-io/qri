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
//  <dsref> = <humanFriendlyRef> [ <concreteRef> ] | <concreteRef>
//  <humanFriendlyRef> = <username> '/' <datasetname>
//  <concreteRef> = '@' [ <profileID> ] '/' <network> '/' <commitHash>
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
	alphaNumeric       = `[a-zA-Z]\w*`
	alphaNumericDsname = `[a-zA-Z]\w{0,143}`
	b58Id              = `Qm[0-9a-zA-Z]{0,44}`
)

var (
	dsNameCheck    = regexp.MustCompile(`^` + alphaNumericDsname + `$`)
	humanFriendly  = regexp.MustCompile(`^(` + alphaNumeric + `)\/(` + alphaNumericDsname + `)`)
	concreteRef    = regexp.MustCompile(`^@(` + b58Id + `)?\/(` + alphaNumeric + `)\/(` + b58Id + `)`)
	b58StrictCheck = regexp.MustCompile(`^Qm[1-9A-HJ-NP-Za-km-z]*$`)

	// ErrEmptyRef is an error for when a reference is empty
	ErrEmptyRef = fmt.Errorf("empty reference")
	// ErrParseError is an error returned when parsing fails
	ErrParseError = fmt.Errorf("could not parse ref")
	// ErrNotHumanFriendly is an error returned when a reference is not human-friendly
	ErrNotHumanFriendly = fmt.Errorf("ref can only have username/name")
	// ErrBadCaseName is the error when a bad case is used in the dataset name
	ErrBadCaseName = fmt.Errorf("dataset name may not contain any upper-case letters")
	// ErrBadCaseShouldRename is the error when a dataset should be renamed to not use upper case letters
	ErrBadCaseShouldRename = fmt.Errorf("dataset name should not contain any upper-case letters, rename it to only use lower-case letters, numbers, and underscores")
	// ErrDescribeValidName is an error describing a valid dataset name
	ErrDescribeValidName = fmt.Errorf("dataset name must start with a letter, and only contain letters, numbers, and underscore. Maximum length is 144 characters")
)

// Parse a reference from a string
func Parse(text string) (Ref, error) {
	var r Ref
	origLength := len(text)
	if origLength == 0 {
		return r, ErrEmptyRef
	}

	remain, partial, err := parseHumanFriendly(text)
	if err == nil {
		text = remain
		r.Username = partial.Username
		r.Name = partial.Name
	} else if err != ErrParseError {
		return r, err
	}

	remain, partial, err = parseConcreteRef(text)
	if err == nil {
		text = remain
		r.ProfileID = partial.ProfileID
		r.Path = partial.Path
	} else if err != ErrParseError {
		return r, err
	}

	if text != "" {
		pos := origLength - len(text)
		return r, fmt.Errorf("parsing ref, unexpected character at position %d: '%c'", pos, text[0])
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

	remain, partial, err := parseHumanFriendly(text)
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
		return r, fmt.Errorf("parsing ref, unexpected character at position %d: '%c'", pos, text[0])
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

// IsRefString returns whether the string parses as a valid reference
func IsRefString(text string) bool {
	_, err := Parse(text)
	return err == nil || err == ErrBadCaseName
}

// IsValidName returns whether the dataset name is valid
func IsValidName(text string) bool {
	return dsNameCheck.Match([]byte(text))
}

func parseHumanFriendly(text string) (string, Ref, error) {
	var r Ref
	matches := humanFriendly.FindStringSubmatch(text)
	if matches == nil {
		return text, r, ErrParseError
	}
	if len(matches) != 3 {
		return text, r, fmt.Errorf("unexpected number of regex matches %d", len(matches))
	}
	matchedLen := len(matches[0])
	r.Username = matches[1]
	r.Name = matches[2]
	return text[matchedLen:], r, nil
}

func parseConcreteRef(text string) (string, Ref, error) {
	var r Ref
	matches := concreteRef.FindStringSubmatch(text)
	if matches == nil {
		return text, r, ErrParseError
	}
	if len(matches) != 4 {
		return text, r, fmt.Errorf("unexpected number of regex matches %d", len(matches))
	}
	if matches[2] != "map" && matches[2] != "ipfs" {
		return text, r, fmt.Errorf("invalid network")
	}
	matchedLen := len(matches[0])
	if matches[1] != "" && b58StrictCheck.FindString(matches[1]) == "" {
		return text, r, fmt.Errorf("profileID contains invalid base58 characters")
	}
	r.ProfileID = matches[1]
	if matches[3] != "" && b58StrictCheck.FindString(matches[3]) == "" {
		return text, r, fmt.Errorf("path contains invalid base58 characters")
	}
	r.Path = fmt.Sprintf("/%s/%s", matches[2], matches[3])
	return text[matchedLen:], r, nil
}
