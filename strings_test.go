package fasttld

import (
	"reflect"
	"strings"
	"testing"
)

type punyCodeTest struct {
	url      string
	expected string
}

var punyCodeTests = []punyCodeTest{
	{"google.com", "google.com"},
	{"hello.世界.com", "hello.xn--rhqv96g.com"},
	{strings.Repeat("x", 65536) + "\uff00", ""}, // int32 overflow.
}

func TestPunyCode(t *testing.T) {
	for _, test := range punyCodeTests {
		converted := formatAsPunycode(test.url)
		if output := reflect.DeepEqual(converted, test.expected); !output {
			t.Errorf("Output %q not equal to expected %q", converted, test.expected)
		}
	}
}

type reverseTest struct {
	original []string
	expected []string
}

var reverseTests = []reverseTest{
	{[]string{}, []string{}},
	{[]string{"ab"}, []string{"ab"}},
	{[]string{"ab", "cd", "gh", "ij"}, []string{"ij", "gh", "cd", "ab"}},
	{[]string{"ab", "cd", "ef", "gh", "ij"}, []string{"ij", "gh", "ef", "cd", "ab"}},
}

func TestReverse(t *testing.T) {
	for _, test := range reverseTests {
		reverse(test.original)
		if output := reflect.DeepEqual(test.original, test.expected); !output {
			t.Errorf("Output %q not equal to expected %q", test.original, test.expected)
		}
	}
}

func TestRuneSliceAscending(t *testing.T) {
	slices := []runeSlice{whitespaceRuneSlice, labelSeparatorsRuneSlice, invalidHostNameCharsRuneSlice,
		validHostNameCharsRuneSlice}
	for sIdx, slice := range slices {
		if len(slice) == 0 {
			t.Errorf("Slice at index %d : is empty", sIdx)
		} else {
			val := int(slice[0])
			for idx, r := range slice {
				rVal := int(r)
				if idx != 0 && rVal <= val {
					t.Errorf("Slice at index %d :Element value at index %d less than or equal to element value at index %d", sIdx, idx, idx-1)
				}
				val = rVal
			}
		}
	}
}

type runeBinarySearchTest struct {
	target      rune
	sortedRunes runeSlice
	exists      bool
}

var runeBinarySearchTests = []runeBinarySearchTest{
	{'r', runeSlice{'a', 'b', 'k', '水'}, false},
	{'a', runeSlice{'a', 'b', 'k', '水'}, true},
	{'水', runeSlice{'a', 'b', 'k', '水'}, true},
	{'r', runeSlice{'a', 'b', '土', '木', '水', '火', '金'}, false},
	{'0', runeSlice{'a', 'b', '土', '木', '水', '火', '金'}, false},
	{'日', runeSlice{'a', 'b', '土', '木', '水', '火', '金'}, false},
	{'界', runeSlice{'a', 'b', '土', '木', '水', '火', '金'}, false},
	{'a', runeSlice{'a', 'b', '土', '木', '水', '火', '金'}, true},
	{'火', runeSlice{'a', 'b', '土', '木', '水', '火', '金'}, true},
	{'b', runeSlice{'a', 'b', '土', '木', '水', '火', '金'}, true},
	{'水', runeSlice{'a', 'b', '土', '木', '水', '火', '金'}, true},
}

func TestRuneBinarySearch(t *testing.T) {
	for _, test := range runeBinarySearchTests {
		if exists := runeBinarySearch(test.target, test.sortedRunes); exists != test.exists {
			t.Errorf("Output %t not equal to expected %t", exists, test.exists)
		}
	}
}

func TestFastTrim(t *testing.T) {
	ss := []string{".abc\u002e", "\u002eabc.", ".abc", "abc\u002e",
		"..abc\u002e", "\u002e.abc\u002e.",
	}
	expected := "abc"
	for _, s := range ss {
		if output := fastTrim(s, labelSeparatorsRuneSlice, trimBoth); output != expected {
			t.Errorf("Output %q not equal to expected %q", output, expected)
		}
	}
}
