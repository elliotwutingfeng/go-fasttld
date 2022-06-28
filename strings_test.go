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
	{"https://google.com", "https://google.com"},
	{"https://hello.世界.com", "https://hello.xn--rhqv96g.com"},
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
