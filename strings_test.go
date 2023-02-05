package fasttld

import (
	"reflect"
	"strings"
	"testing"

	"github.com/karlseguin/intset"
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

func TestFastTrim(t *testing.T) {
	const charsToTrim string = ".@新"
	var charsToTrimRuneSet *intset.Rune = makeRuneSet(charsToTrim)

	ss := []string{".abc.", ".abc", "abc.", "..abc.", ".abc..", "..abc..",
		"@abc@", "@abc", "abc@", "@@abc@", "@abc@@", "@@abc@@",
		"新abc新", "新abc", "abc新", "新新abc新", "新abc新新", "新新abc新新",
		"新@abc新.", "新.abc", "abc@新", "新新.abc新", "新abc新@新", "新新.abc.新新",
		".", "..",
		".@", "@.",
		".@新", "新@.",
		" ", " .@ ", ". .@ ", " .@ 新",
		"abc"}

	for _, s := range ss {
		expectedTrimBoth := strings.Trim(s, charsToTrim)
		if output := fastTrim(s, charsToTrimRuneSet, trimBoth); output != expectedTrimBoth {
			t.Errorf("Output %q not equal to expected %q", output, expectedTrimBoth)
		}
		expectedTrimLeft := strings.TrimLeft(s, charsToTrim)
		if output := fastTrim(s, charsToTrimRuneSet, trimLeft); output != expectedTrimLeft {
			t.Errorf("Output %q not equal to expected %q", output, expectedTrimLeft)
		}
		expectedTrimRight := strings.TrimRight(s, charsToTrim)
		if output := fastTrim(s, charsToTrimRuneSet, trimRight); output != expectedTrimRight {
			t.Errorf("Output %q not equal to expected %q", output, expectedTrimRight)
		}
	}
}
