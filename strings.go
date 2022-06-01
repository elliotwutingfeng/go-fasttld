package fasttld

import (
	"log"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// Obtained from IETF RFC 3490
const labelSeparators string = "\u002e\u3002\uff0e\uff61"

const whitespace string = " \t\n\v\f\r\uFEFF\u200b\u200c\u200d\u00a0\u1680\u0085\u00a0"

// For replacing internationalised label separators when converting URL to punycode.
var standardLabelSeparatorReplacer = strings.NewReplacer(makeNewReplacerParams(labelSeparators, ".")...)

const endOfHostWithPortDelimiters string = `/\?#`

var endOfHostWithPortDelimitersSet asciiSet = makeASCIISet(endOfHostWithPortDelimiters)

const endOfHostDelimiters string = endOfHostWithPortDelimiters + ":"

var endOfHostDelimitersSet asciiSet = makeASCIISet(endOfHostDelimiters)

// Characters that cannot appear in UserInfo
const invalidUserInfoChars string = endOfHostWithPortDelimiters + "[]"

var invalidUserInfoCharsSet asciiSet = makeASCIISet(invalidUserInfoChars)

// For extracting URL scheme.
var schemeRegex = regexp.MustCompile(`(?i)^([a-z][a-z0-9+-.]*:)?[\\/]{2,}`)

// asciiSet is a 32-byte value, where each bit represents the presence of a
// given ASCII character in the set. The 128-bits of the lower 16 bytes,
// starting with the least-significant bit of the lowest word to the
// most-significant bit of the highest word, map to the full range of all
// 128 ASCII characters. The 128-bits of the upper 16 bytes will be zeroed,
// ensuring that any non-ASCII character will be reported as not in the set.
// This allocates a total of 32 bytes even though the upper half
// is unused to avoid bounds checks in asciiSet.contains.
type asciiSet [8]uint32

// makeASCIISet creates a set of ASCII characters.
//
// Similar to strings.makeASCIISet but skips input validation.
func makeASCIISet(chars string) (as asciiSet) {
	// all characters in chars are expected to be valid ASCII characters
	for i := 0; i < len(chars); i++ {
		c := chars[i]
		as[c/32] |= 1 << (c % 32)
	}
	return as
}

// contains reports whether c is inside the set.
//
// same as strings.contains.
func (as *asciiSet) contains(c byte) bool {
	return (as[c/32] & (1 << (c % 32))) != 0
}

// indexAnyASCII returns the index of the first instance of any Unicode code point
// from asciiSet in s, or -1 if no Unicode code point from asciiSet is present in s.
//
// Similar to strings.IndexAny but takes in an asciiSet instead of a string
// and skips input validation.
func indexAnyASCII(s string, as asciiSet) int {
	for i := 0; i < len(s); i++ {
		if as.contains(s[i]) {
			return i
		}
	}
	return -1
}

// indexAny returns the index of the first instance of any Unicode code point
// from chars in s, or -1 if no Unicode code point from chars is present in s.
//
// Similar to strings.IndexAny but does not attempt to make an asciiSet
// and skips input validation.
func indexAny(s, chars string) int {
	for i, c := range s {
		if strings.IndexRune(chars, c) != -1 {
			return i
		}
	}
	return -1
}

// lastIndexAny returns the index of the last instance of any Unicode code
// point from chars in s, or -1 if no Unicode code point from chars is
// present in s.
//
// Similar to strings.LastIndexAny but skips input validation.
func lastIndexAny(s string, chars string) int {
	for i := len(s); i > 0; {
		r, size := utf8.DecodeLastRuneInString(s[0:i])
		i -= size
		if strings.IndexRune(chars, r) != -1 {
			return i
		}
	}
	return -1
}

// reverse reverses a slice of strings in-place.
func reverse(input []string) {
	for i, j := 0, len(input)-1; i < j; i, j = i+1, j-1 {
		input[i], input[j] = input[j], input[i]
	}
}

// sepSize returns byte length of an sep rune, given the rune's first byte.
func sepSize(r byte) int {
	// r is the first byte of any of the runes in labelSeparators
	if r == 46 {
		// First byte of '.' is 46
		// size of '.' is 1
		return 1
	}
	// First byte of any label separator other than '.' is not 46
	// size of separator is 3
	return 3
}

// formatAsPunycode formats s as punycode.
func formatAsPunycode(s string) string {
	asPunyCode, err := idna.ToASCII(s)
	if err != nil {
		log.Println(strings.SplitAfterN(err.Error(), "idna: invalid label", 2)[0])
		return ""
	}
	return asPunyCode
}

// makeNewReplacerParams generates parameters for
// the strings.NewReplacer function
// where all runes in toBeReplaced are to be
// replaced by toReplaceWith
func makeNewReplacerParams(toBeReplaced string, toReplaceWith string) []string {
	var params = make([]string, len(toBeReplaced))
	for _, r := range toBeReplaced {
		params = append(params, string(r), toReplaceWith)
	}
	return params
}

// indexByteExceptAfter returns the index of the first instance of byte b,
// otherwise -1 if any byte in notAfterCharsSet is found first or if b is not present in s.
func indexByteExceptAfter(s string, b byte, notAfterCharsSet asciiSet) int {
	if firstNotAfterCharIdx := indexAnyASCII(s, notAfterCharsSet); firstNotAfterCharIdx != -1 {
		return strings.IndexByte(s[0:firstNotAfterCharIdx], b)
	}
	return strings.IndexByte(s, b)
}
