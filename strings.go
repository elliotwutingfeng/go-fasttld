// Package fasttld is a high performance top level domains (TLD)
// extraction module implemented with compressed tries.
//
// This module is a port of the Python fasttld module,
// with additional modifications to support extraction
// of subcomponents from full URLs, IPv4 addresses, and IPv6 addresses.
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

const whitespace string = " \n\t\r\uFEFF\u200b\u200c\u200d"

// For replacing internationalised label separators when converting URL to punycode.
var standardLabelSeparatorReplacer = strings.NewReplacer(makeNewReplacerParams(labelSeparators, ".")...)

const endOfHostDelimiters string = "/:?&#"

var endOfHostDelimitersSet asciiSet = makeASCIISet(endOfHostDelimiters)

// For extracting URL scheme.
var schemeRegex = regexp.MustCompile("^([A-Za-z0-9+-.]+:)?//")

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

// indexAny returns the index of the first instance of any Unicode code point
// from asciiSet in s, or -1 if no Unicode code point from asciiSet is present in s.
//
// Similar to strings.IndexAny but takes in an asciiSet instead of a string
// and skips input validation.
func indexAny(s string, as asciiSet) int {
	for i := 0; i < len(s); i++ {
		if as.contains(s[i]) {
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
		r, size := utf8.DecodeLastRuneInString(s[:i])
		i -= size
		if strings.IndexRune(chars, r) >= 0 {
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
	// size of delimiter is 3
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
	var params = make([]string, 8)
	for _, r := range toBeReplaced {
		params = append(params, string(r))
		params = append(params, toReplaceWith)
	}
	return params
}
