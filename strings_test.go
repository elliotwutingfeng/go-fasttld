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
