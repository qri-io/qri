package dsref

import (
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// NameMaxLength is the maximum length of a name that will be generated
var NameMaxLength = 44

// GenerateName converts the input into a valid dataset string, which starts with a lower-case
// letter, and only has letters, digits, dashes and underscores.
func GenerateName(input, prefix string) string {
	// Normalize unicode by trying to convert all unicode characters to ascii.
	// https://stackoverflow.com/questions/26722450/remove-diacritics-using-go
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isNonspacingMark), norm.NFC)
	input, _, _ = transform.String(t, input)
	// Trim space from the left and right
	input = strings.TrimSpace(input)
	// Run the state machine that converts the string to a valid dataset name
	name := convertWordsStateMachine(input)
	// If the dataset does not start with a lower-case letter, add the prefix. It is the
	// responsibility of the caller to provide a valid prefix.
	first := []rune(name)[0]
	if !unicode.IsLower(first) {
		name = prefix + name
	}
	return name
}

func isNonspacingMark(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}

// State is used to handle the state machine that processes strings
type State int

const (
	// StateNone is for being outside of a word, looking at spaces
	StateNone State = 0
	// StateLowerWord is for looking at a word made-up of lower-case letters
	StateLowerWord State = 1
	// StateFirstUpper is for looking at the first upper-case letter of a word
	StateFirstUpper State = 2
	// StateCapsWord is for looking at a word made up of all upper-case letters
	StateCapsWord State = 3
	// StateNumber is for looking at a sequence of digits
	StateNumber State = 4
	// StatePunc is for looking at punctuation characters
	StatePunc State = 5
	// StateExit is set by flush in order to exit the top-level loop immediately
	StateExit State = 6
)

// Process a string, finding each word and outputting them to a new string. Words may be separated
// by spaces, or punctuation, or could be camel cased. In any case, they should be separated
// by dashes or underscores in the output string. Other punctuation characters will be replaced
// by dashes.
func convertWordsStateMachine(input string) string {
	state := StateNone
	// result is the accumlated string based upon the state machine's position
	result := strings.Builder{}
	// next is one or more dashes (obtained from spaces or punctuation) or underscores, followed
	// by characters in the curent word. Once a state change finishes that word, it will be
	// added to the result by calling `flush`.
	next := strings.Builder{}

	// flushWord will take the word in `next` and append it to the result. It handles any per-word
	// tasks, like lower-casing words, separating words with underscores, and making sure the
	// result does not exceed the maximum length.
	flushWord := func(nextState State, nextRune rune) {
		word := strings.ToLower(next.String())

		// If nothing to flush, exit early
		if word == "" {
			return
		}

		// Put an underscore between words
		if result.Len() > 0 {
			prev := []rune(result.String())
			// Check if the previous word ended with, and the next word starts with, an alphanum.
			if isAlphanum(prev[len(prev)-1]) && isAlphanum([]rune(word)[0]) {
				word = "_" + word
			}
		}

		// Check length of result
		if result.Len()+len(word) > NameMaxLength {
			if result.Len() == 0 {
				result.WriteString(word[:NameMaxLength])
			}
			state = StateExit
			return
		}

		// Add the word to the result
		result.WriteString(word)
		next.Reset()

		// Assign next state and rune
		state = nextState
		next.WriteRune(nextRune)
	}

	//
	// The state machine below is used to convert an arbitrary string into a string that is
	// always a valid dataset name. The main two reasons for using a state machine instead of
	// another approach is these two cases:
	//   AnnualTPSReport -> annual_tps_report
	//   category: climate -> category-climate
	// These both require looking at multiple characters in a row in order to decide how to
	// split words and replace punctuation. The state machine accomplishes this with the
	// two particular states StateCapsWord and StatePunc, respectively.
	//
	// The basic form of the state machine that accomplishes these cases is this:
	//
	// +-----------+            +-----------------+            +---------------+
	// | StateNone | - upper -> | StateFirstUpper | - upper -> | StateCapsWord |
	// +-----------+            +-----------------+            +---------------+
	//    |  |                     |                              |
	//    |  |                    lower (CamelCase)              lower
	//    |  |                     |                              |
	//    |  |                     v                              v
	//    |  |               +------------+             `split prev, goto StateLower`
	//    | lower ---------> | StateLower |
	//    |                  +------------+
	//    |
	//    |                  +-----------+
	//   punc -------------> | StatePunc | - space -> `ignore space, combine with punc`
	//                       +-----------+

	for _, r := range input {
		if r > 127 {
			// Ignore non-ascii code points
			continue
		}

		switch state {
		case StateExit:
			break

		case StateNone:
			if r == ' ' {
				next.WriteRune('_')
			} else if unicode.IsLower(r) {
				state = StateLowerWord
				next.WriteRune(r)
			} else if unicode.IsUpper(r) {
				state = StateFirstUpper
				next.WriteRune(r)
			} else if unicode.IsDigit(r) {
				state = StateNumber
				next.WriteRune(r)
			} else if isPunc(r) {
				state = StatePunc
				next.WriteRune(r)
			} else if r == '_' || r == '-' {
				next.WriteRune('-')
			}

		case StateLowerWord:
			if r == ' ' {
				flushWord(StateNone, '_')
			} else if unicode.IsLower(r) {
				next.WriteRune(r)
			} else if unicode.IsUpper(r) {
				// Was looking at a word of lower-case characters, and now encountered a
				// upper-case letter, which means the previous word is done
				flushWord(StateFirstUpper, r)
			} else if unicode.IsDigit(r) {
				flushWord(StateNumber, r)
			} else if isPunc(r) {
				flushWord(StatePunc, '-')
			} else if r == '_' || r == '-' {
				flushWord(StateNone, r)
			}

		case StateFirstUpper:
			if r == ' ' {
				flushWord(StateNone, '_')
			} else if unicode.IsLower(r) {
				state = StateLowerWord
				next.WriteRune(r)
			} else if unicode.IsUpper(r) {
				state = StateCapsWord
				next.WriteRune(r)
			} else if unicode.IsDigit(r) {
				flushWord(StateNumber, r)
			} else if isPunc(r) {
				flushWord(StatePunc, '-')
			} else if r == '_' || r == '-' {
				flushWord(StateNone, r)
			}

		case StateCapsWord:
			if r == ' ' {
				flushWord(StateNone, '_')
			} else if unicode.IsLower(r) {
				// Just encounterd a series of upper-case letters (2 or more) and now see a
				// lower-case letter. Split off the previous upper-case letters before the final
				// one, and turn that into a word, then keep that final upper-case letter as the
				// start of the next word
				//
				// For example, if this is the string:
				//   NBCTelevisionNetwork
				// We would encounter this situation when the cursor gets here
				//   NBCTelevisionNetwork
				//       ^
				// Which would be split into this:
				// 'nbc' <- previous word
				// 'te'  <- next
				got := []rune(next.String())
				prevWord := got[:len(got)-1]
				lastLetter := got[len(got)-1]
				// Pull off the previous word
				next.Reset()
				next.WriteString(string(prevWord))
				// Flush that word, start the next, which now has two characters
				flushWord(StateLowerWord, lastLetter)
				next.WriteRune(r)
			} else if unicode.IsUpper(r) {
				next.WriteRune(r)
			} else if unicode.IsDigit(r) {
				flushWord(StateNumber, r)
			} else if isPunc(r) {
				flushWord(StatePunc, '-')
			} else if r == '_' || r == '-' {
				flushWord(StateNone, r)
			}

		case StateNumber:
			if r == ' ' {
				flushWord(StateNone, '_')
			} else if unicode.IsLower(r) {
				flushWord(StateLowerWord, r)
			} else if unicode.IsUpper(r) {
				// Was looking at a number, and now encountered a upper-case letter, which
				// means the previous word is done
				flushWord(StateFirstUpper, r)
			} else if unicode.IsDigit(r) {
				next.WriteRune(r)
			} else if isPunc(r) {
				flushWord(StatePunc, '-')
			} else if r == '_' || r == '-' {
				flushWord(StateNone, r)
			}

		case StatePunc:
			if r == ' ' {
				// Punctuation ignores spaces after it
				continue
			} else if unicode.IsLower(r) {
				flushWord(StateLowerWord, r)
			} else if unicode.IsUpper(r) {
				// Was looking at punctuation, and now encountered a upper-case letter, which
				// means the previous word is done
				flushWord(StateFirstUpper, r)
			} else if unicode.IsDigit(r) {
				flushWord(StateNumber, r)
			} else if isPunc(r) {
				next.WriteRune('-')
			} else if r == '_' || r == '-' {
				flushWord(StateNone, r)
			}
		}
	}
	// Input is finished, flush the last word
	flushWord(StateNone, rune(0))
	return result.String()
}

// PuncCharacters is the list of punctuation characters that get converted to dashes
const PuncCharacters = "`~!@#$%^&*()=+[{]}\\|;:'\",<.>/?"

func isPunc(r rune) bool {
	return strings.IndexRune(PuncCharacters, r) != -1
}

func isAlphanum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
