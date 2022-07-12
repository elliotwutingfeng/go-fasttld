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

func TestFastTrim(t *testing.T) {
	ss := []string{".abc\u002e", "\u002eabc.", ".abc", "abc\u002e",
		"..abc\u002e", "\u002e.abc\u002e.",
	}
	expected := "abc"
	for _, s := range ss {
		if output := fastTrim(s, labelSeparatorsRuneSet, trimBoth); output != expected {
			t.Errorf("Output %q not equal to expected %q", output, expected)
		}
	}
	if output := fastTrim(".", labelSeparatorsRuneSet, trimBoth); output != "" {
		t.Errorf("Output %q not equal to expected %q", output, "")
	}
	if output := fastTrim(".", labelSeparatorsRuneSet, trimLeft); output != "" {
		t.Errorf("Output %q not equal to expected %q", output, "")
	}
	if output := fastTrim(".", labelSeparatorsRuneSet, trimRight); output != "" {
		t.Errorf("Output %q not equal to expected %q", output, "")
	}
}
