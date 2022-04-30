// go-fasttld is a high performance top level domains (TLD)
// extraction module implemented with compressed tries.
//
// This module is a port of the Python fasttld module,
// with additional modifications to support extraction
// of subcomponents from full URLs and IPv4 addresses.
package fasttld

import (
	"os"
	"reflect"
	"strings"
	"testing"

	tlde "github.com/M507/tlde/src"
	joeguotldextract "github.com/joeguo/tldextract"
	tld "github.com/jpillora/go-tld"
	mjd2021usatldextract "github.com/mjd2021usa/tldextract"
	"github.com/spf13/afero"
)

func getTestPSLFilePath() string {
	var sb strings.Builder
	sb.WriteString(getCurrentFilePath())
	sb.WriteString(string(os.PathSeparator))
	sb.WriteString("test")
	sb.WriteString(string(os.PathSeparator))
	sb.WriteString(defaultPSLFileName)
	return sb.String()
}

func TestNestedDict(t *testing.T) {
	originalDict := &trie{matches: map[string]*trie{}}
	keysSequence := []([]string){{"a"}, {"a", "d"}, {"a", "b"}, {"a", "b", "c"}, {"c"}, {"c", "b"}, {"d", "f"}}
	for _, keys := range keysSequence {
		nestedDict(originalDict, keys)
	}
	// check each nested value
	//Top level c
	if len(originalDict.matches["c"].matches) != 1 {
		t.Errorf("Top level c must have Matches map of length 1")
	}
	if _, ok := originalDict.matches["c"].matches["b"]; !ok {
		t.Errorf("Top level c must have b in Matches map")
	}
	if !originalDict.matches["c"].end {
		t.Errorf("Top level c must have End = true")
	}
	// Top level a
	if len(originalDict.matches["a"].matches) != 2 {
		t.Errorf("Top level a must have Matches map of length 2")
	}
	// a -> d
	if _, ok := originalDict.matches["a"].matches["d"]; !ok {
		t.Errorf("Top level a must have d in Matches map")
	}
	if len(originalDict.matches["a"].matches["d"].matches) != 0 {
		t.Errorf("a -> d must have empty Matches map")
	}
	// a -> b
	if _, ok := originalDict.matches["a"].matches["b"]; !ok {
		t.Errorf("Top level a must have b in Matches map")
	}
	if !originalDict.matches["a"].matches["b"].end {
		t.Errorf("a -> b must have End = true")
	}
	if len(originalDict.matches["a"].matches["b"].matches) != 1 {
		t.Errorf("a -> b must have Matches map of length 1")
	}
	// a -> b -> c
	if _, ok := originalDict.matches["a"].matches["b"].matches["c"]; !ok {
		t.Errorf("a -> b must have c in Matches map")
	}
	if len(originalDict.matches["a"].matches["b"].matches["c"].matches) != 0 {
		t.Errorf("a -> b -> c must have empty Matches map")
	}
	if !originalDict.matches["a"].end {
		t.Errorf("Top level a must have End = true")
	}
	// d -> f
	if originalDict.matches["d"].end {
		t.Errorf("Top level d must have End = false")
	}
	if originalDict.matches["d"].matches["f"].end {
		t.Errorf("d -> f must have End = false")
	}
	if len(originalDict.matches["d"].matches["f"].matches) != 0 {
		t.Errorf("d -> f must have empty Matches map")
	}
}

func TestTrie(t *testing.T) {
	trie, err := trieConstruct(false, "test/mini_public_suffix_list.dat")
	if err != nil {
		t.Errorf("trieConstruct failed | %q", err)
	}
	if lenTrieMatches := len(trie.matches); lenTrieMatches != 2 {
		t.Errorf("Expected top level Trie Matches map length of 2. Got %d.", lenTrieMatches)
	}
	for _, tld := range []string{"ac", "ck"} {
		if _, ok := trie.matches[tld]; !ok {
			t.Errorf("Top level %q must exist", tld)
		}
	}
	if !trie.matches["ac"].end {
		t.Errorf("Top level ac must have End = true")
	}
	if trie.matches["ck"].end {
		t.Errorf("Top level ck must have End = false")
	}
	if len(trie.matches["ck"].matches) != 2 {
		t.Errorf("Top level ck must have Matches map of length 2")
	}
	if _, ok := trie.matches["ck"].matches["*"]; !ok {
		t.Errorf("Top level ck must have * in Matches map")
	}
	if len(trie.matches["ck"].matches["*"].matches) != 0 {
		t.Errorf("ck -> * must have empty Matches map")
	}
	if _, ok := trie.matches["ck"].matches["!www"]; !ok {
		t.Errorf("Top level ck must have !www in Matches map")
	}
	if len(trie.matches["ck"].matches["!www"].matches) != 0 {
		t.Errorf("ck -> !www must have empty Matches map")
	}
	for _, tld := range []string{"com", "edu", "gov", "net", "mil", "org"} {
		if _, ok := trie.matches["ac"].matches[tld]; !ok {
			t.Errorf("Top level ac must have %q in Matches map", tld)
		}
		if len(trie.matches["ac"].matches[tld].matches) != 0 {
			t.Errorf("ac -> %q must have empty Matches map", tld)
		}
	}
}

type reverseTest struct {
	original []string
	expected []string
}

var reverseTests = []reverseTest{
	{[]string{"ab", "cd", "ef", "gh", "ij"}, []string{"ij", "gh", "ef", "cd", "ab"}},
	{[]string{"ab", "cd", "gh", "ij"}, []string{"ij", "gh", "cd", "ab"}},
}

func TestReverse(t *testing.T) {
	for _, test := range reverseTests {
		reverse(test.original)
		if output := reflect.DeepEqual(test.original, test.expected); !output {
			t.Errorf("Output %q not equal to expected %q", test.original, test.expected)
		}
	}
}

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

type newTest struct {
	cacheFilePath        string
	includePrivateSuffix bool
	expected             int
}

var newTests = []newTest{
	{cacheFilePath: "test/public_suffix_list.dat", includePrivateSuffix: false, expected: 1656},
	{cacheFilePath: "test/public_suffix_list.dat", includePrivateSuffix: true, expected: 1674},
}

func TestNew(t *testing.T) {
	for _, test := range newTests {
		cacheFilePath := test.cacheFilePath
		if cacheFilePath == "" {
			cacheFilePath = getTestPSLFilePath()
		}
		extractor, _ := New(SuffixListParams{
			CacheFilePath:        cacheFilePath,
			IncludePrivateSuffix: test.includePrivateSuffix,
		})
		if numTopLevelKeys := len(extractor.TldTrie.matches); numTopLevelKeys != test.expected {
			t.Errorf("Expected number of top level keys to be %d. Got %d.", test.expected, numTopLevelKeys)
		}
	}
}

type extractTest struct {
	includePrivateSuffix bool
	urlParams            URLParams
	expected             *ExtractResult
	description          string
}

var extraExtractTests = []extractTest{
	{urlParams: URLParams{URL: "maps.google.com.sg",
		IgnoreSubDomains: true},
		expected: &ExtractResult{
			Domain: "google", Suffix: "com.sg",
			RegisteredDomain: "google.com.sg",
		}, description: "Ignore SubDomains",
	},

	{urlParams: URLParams{URL: "https://brb.i.am.going.to.be.a.fk"},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb.i.am.going.to", Domain: "be", Suffix: "a.fk",
			RegisteredDomain: "be.a.fk",
		}, description: "Asterisk",
	},

	{urlParams: URLParams{URL: "https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk"},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb\u002ei\u3002am\uff0egoing\uff61to", Domain: "be", Suffix: "a\uff61fk",
			RegisteredDomain: "be\u3002a\uff61fk",
		}, description: "internationalised domain name dots",
	},

	{urlParams: URLParams{URL: "a\uff61fk"},
		expected: &ExtractResult{Suffix: "a\uff61fk"}, description: "internationalised domain name dots | Suffix only",
	},

	{includePrivateSuffix: true,
		urlParams: URLParams{URL: "https://brb.i.am.going.to.be.blogspot.com:5000/a/b/c/d.txt?id=42",
			IgnoreSubDomains: false, ConvertURLToPunyCode: false},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb.i.am.going.to", Domain: "be", Suffix: "blogspot.com",
			RegisteredDomain: "be.blogspot.com", Port: "5000", Path: "a/b/c/d.txt?id=42",
		}, description: "Include Private Suffix"},
	{includePrivateSuffix: true,
		urlParams: URLParams{URL: "global.prod.fastly.net",
			IgnoreSubDomains: false, ConvertURLToPunyCode: false},
		expected: &ExtractResult{
			Suffix: "global.prod.fastly.net",
		}, description: "Include Private Suffix | Suffix only"},

	{includePrivateSuffix: true,
		urlParams: URLParams{URL: "maps.google.com.sg:5000",
			IgnoreSubDomains: false, ConvertURLToPunyCode: false},
		expected: &ExtractResult{
			SubDomain: "maps", Domain: "google", Suffix: "com.sg",
			RegisteredDomain: "google.com.sg", Port: "5000",
		}, description: "Port number"},

	{includePrivateSuffix: true,
		urlParams: URLParams{URL: "maps.google.com.sg:8589934592/this/path/will/not/be/parsed",
			IgnoreSubDomains: false, ConvertURLToPunyCode: false},
		expected: &ExtractResult{
			SubDomain: "maps", Domain: "google", Suffix: "com.sg",
			RegisteredDomain: "google.com.sg",
		}, description: "Invalid Port number"},

	{includePrivateSuffix: true,
		urlParams: URLParams{URL: " https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk/a/b/c. \uff61",
			IgnoreSubDomains: false, ConvertURLToPunyCode: false},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb\u002ei\u3002am\uff0egoing\uff61to", Domain: "be", Suffix: "a\uff61fk",
			RegisteredDomain: "be\u3002a\uff61fk", Path: "a/b/c",
		}, description: "Surrounded by extra whitespace and periods"},

	{includePrivateSuffix: true,
		urlParams: URLParams{URL: " https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk/a/b/c. \uff61",
			IgnoreSubDomains: false, ConvertURLToPunyCode: true},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb.i.am.going.to", Domain: "be", Suffix: "a.fk",
			RegisteredDomain: "be.a.fk", Path: "a/b/c",
		}, description: "Surrounded by extra whitespace and periods | PunyCode"},
}

// test cases ported from https://github.com/mjd2021usa/tldextract
var tldExtractGoTests = []extractTest{
	{urlParams: URLParams{}, expected: &ExtractResult{}, description: "empty string"},
	{urlParams: URLParams{URL: "users@myhost.com"}, expected: &ExtractResult{UserInfo: "users", Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com"}, description: "user@ address"},
	{urlParams: URLParams{URL: "mailto:users@myhost.com"}, expected: &ExtractResult{UserInfo: "mailto:users", Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com"}, description: "email address"},
	{urlParams: URLParams{URL: "myhost.com:999"}, expected: &ExtractResult{Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com", Port: "999"}, description: "host:port"},
	{urlParams: URLParams{URL: "myhost.com"}, expected: &ExtractResult{Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com"}, description: "basic host"},
	{urlParams: URLParams{URL: "255.255.myhost.com"}, expected: &ExtractResult{SubDomain: "255.255", Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com"}, description: "basic host with numeric subdomains"},
	{urlParams: URLParams{URL: "https://user:pass@foo.myhost.com:999/some/path?param1=value1&param2=value2"}, expected: &ExtractResult{Scheme: "https://", UserInfo: "user:pass", SubDomain: "foo", Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com", Port: "999", Path: "some/path?param1=value1&param2=value2"}, description: "Full URL with subdomain"},
	{urlParams: URLParams{URL: "http://www.duckduckgo.com"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "www", Domain: "duckduckgo", Suffix: "com", RegisteredDomain: "duckduckgo.com"}, description: "Full URL with subdomain"},
	{urlParams: URLParams{URL: "http://duckduckgo.co.uk/path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, expected: &ExtractResult{Scheme: "http://", Domain: "duckduckgo", Suffix: "co.uk", RegisteredDomain: "duckduckgo.co.uk", Path: "path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, description: "Full HTTP URL with no subdomain"},
	{urlParams: URLParams{URL: "http://big.long.sub.domain.duckduckgo.co.uk/path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "big.long.sub.domain", Domain: "duckduckgo", Suffix: "co.uk", RegisteredDomain: "duckduckgo.co.uk", Path: "path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, description: "Full HTTP URL with subdomain"},
	{urlParams: URLParams{URL: "https://duckduckgo.co.uk/path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, expected: &ExtractResult{Scheme: "https://", Domain: "duckduckgo", Suffix: "co.uk", RegisteredDomain: "duckduckgo.co.uk", Path: "path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, description: "Full HTTPS URL with no subdomain"},
	{urlParams: URLParams{URL: "ftp://peterparker:multipass@mail.duckduckgo.co.uk:666/path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, expected: &ExtractResult{Scheme: "ftp://", UserInfo: "peterparker:multipass", SubDomain: "mail", Domain: "duckduckgo", Suffix: "co.uk", RegisteredDomain: "duckduckgo.co.uk", Port: "666", Path: "path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, description: "Full ftp URL with subdomain"},
	{urlParams: URLParams{URL: "git+ssh://www.github.com/"}, expected: &ExtractResult{Scheme: "git+ssh://", SubDomain: "www", Domain: "github", Suffix: "com", RegisteredDomain: "github.com"}, description: "Full git+ssh URL with subdomain"},
	{urlParams: URLParams{URL: "ssh://server.domain.com/"}, expected: &ExtractResult{Scheme: "ssh://", SubDomain: "server", Domain: "domain", Suffix: "com", RegisteredDomain: "domain.com"}, description: "Full ssh URL with subdomain"},
	{urlParams: URLParams{URL: "//server.domain.com/path"}, expected: &ExtractResult{Scheme: "//", SubDomain: "server", Domain: "domain", Suffix: "com", RegisteredDomain: "domain.com", Path: "path"}, description: "Missing protocol URL with subdomain"},
	{urlParams: URLParams{URL: "server.domain.com/path"}, expected: &ExtractResult{SubDomain: "server", Domain: "domain", Suffix: "com", RegisteredDomain: "domain.com", Path: "path"}, description: "Full ssh URL with subdomain"},
	{urlParams: URLParams{URL: "10.10.10.10"}, expected: &ExtractResult{Domain: "10.10.10.10", RegisteredDomain: "10.10.10.10"}, description: "Basic IPv4 Address"},
	{urlParams: URLParams{URL: "http://10.10.10.10:5000"}, expected: &ExtractResult{Scheme: "http://", Domain: "10.10.10.10", RegisteredDomain: "10.10.10.10", Port: "5000"}, description: "Basic IPv4 Address URL"},
	{urlParams: URLParams{URL: "http://10.10.10.256"}, expected: &ExtractResult{Scheme: "http://"}, description: "Basic IPv4 Address URL with bad IP"},
	{urlParams: URLParams{URL: "http://godaddy.godaddy"}, expected: &ExtractResult{Scheme: "http://", Domain: "godaddy", Suffix: "godaddy", RegisteredDomain: "godaddy.godaddy"}, description: "Basic URL"},
	{urlParams: URLParams{URL: "http://godaddy.godaddy.godaddy"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "godaddy", Domain: "godaddy", Suffix: "godaddy", RegisteredDomain: "godaddy.godaddy"}, description: "Basic URL with subdomain"},
	{urlParams: URLParams{URL: "http://godaddy.godaddy.co.uk"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "godaddy", Domain: "godaddy", Suffix: "co.uk", RegisteredDomain: "godaddy.co.uk"}, description: "Basic URL with subdomain"},
	{urlParams: URLParams{URL: "http://godaddy"}, expected: &ExtractResult{Scheme: "http://", Suffix: "godaddy"}, description: "Basic URL with TLD only"},
	{urlParams: URLParams{URL: "http://godaddy.cannon-fodder"}, expected: &ExtractResult{Scheme: "http://"}, description: "Basic URL with bad TLD"},
	{urlParams: URLParams{URL: "http://godaddy.godaddy.cannon-fodder"}, expected: &ExtractResult{Scheme: "http://"}, description: "Basic URL with subdomainand bad TLD"},

	{urlParams: URLParams{URL: "http://domainer.个人.hk", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "domainer", Suffix: "xn--ciqpn.hk", RegisteredDomain: "domainer.xn--ciqpn.hk"}, description: "Basic URL with mixed international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://domainer.公司.香港", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "domainer", Suffix: "xn--55qx5d.xn--j6w193g", RegisteredDomain: "domainer.xn--55qx5d.xn--j6w193g"}, description: "Basic URL with full international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://domainer.个人.hk"}, expected: &ExtractResult{Scheme: "http://", Domain: "domainer", Suffix: "个人.hk", RegisteredDomain: "domainer.个人.hk"}, description: "Basic URL with mixed international TLD (result in unicode)"},
	{urlParams: URLParams{URL: "http://domainer.公司.香港"}, expected: &ExtractResult{Scheme: "http://", Domain: "domainer", Suffix: "公司.香港", RegisteredDomain: "domainer.公司.香港"}, description: "Basic URL with full international TLD (result in unicode)"},

	{urlParams: URLParams{URL: "http://domainer.xn--ciqpn.hk", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "domainer", Suffix: "xn--ciqpn.hk", RegisteredDomain: "domainer.xn--ciqpn.hk"}, description: "Basic URL with mixed punycode international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://domainer.xn--55qx5d.xn--j6w193g", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "domainer", Suffix: "xn--55qx5d.xn--j6w193g", RegisteredDomain: "domainer.xn--55qx5d.xn--j6w193g"}, description: "Basic URL with full punycode international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://domainer.xn--ciqpn.hk"}, expected: &ExtractResult{Scheme: "http://", Domain: "domainer", Suffix: "xn--ciqpn.hk", RegisteredDomain: "domainer.xn--ciqpn.hk"}, description: "Basic URL with mixed punycode international TLD (result in unicode)"},
	{urlParams: URLParams{URL: "http://domainer.xn--55qx5d.xn--j6w193g"}, expected: &ExtractResult{Scheme: "http://", Domain: "domainer", Suffix: "xn--55qx5d.xn--j6w193g", RegisteredDomain: "domainer.xn--55qx5d.xn--j6w193g"}, description: "Basic URL with full punycode international TLD (result in unicode)"},

	{urlParams: URLParams{URL: "https://example.ai/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "ai", RegisteredDomain: "example.ai", Path: "en"}, description: "Domain only + ai"},
	{urlParams: URLParams{URL: "https://example.co/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "co", RegisteredDomain: "example.co", Path: "en"}, description: "Domain only + co"},
	{urlParams: URLParams{URL: "https://example.sg/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "sg", RegisteredDomain: "example.sg", Path: "en"}, description: "Domain only + sg"},
	{urlParams: URLParams{URL: "https://example.tv/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "tv", RegisteredDomain: "example.tv", Path: "en"}, description: "Domain only + tv"},

	{urlParams: URLParams{URL: "https://asdf.wwe.ck"}, expected: &ExtractResult{Scheme: "https://", Domain: "asdf", Suffix: "wwe.ck", RegisteredDomain: "asdf.wwe.ck"}, description: "Wildcard rule | *.ck"},
	{urlParams: URLParams{URL: "https://asdf.www.ck"}, expected: &ExtractResult{Scheme: "https://", SubDomain: "asdf", Domain: "www", Suffix: "ck", RegisteredDomain: "www.ck"}, description: "Wildcard exception rule | !www.ck"},

	// {urlParams: URLParams{URL: "git+ssh://www.!github.com/"}, expected: &ExtractResult{}, description: "Full git+ssh URL with bad domain"},
}

func TestExtract(t *testing.T) {
	extractorWithPrivateSuffix, _ := New(SuffixListParams{
		CacheFilePath:        getTestPSLFilePath(),
		IncludePrivateSuffix: true,
	})
	extractorWithoutPrivateSuffix, _ := New(SuffixListParams{
		CacheFilePath:        getTestPSLFilePath(),
		IncludePrivateSuffix: false,
	})
	for _, testCollection := range []([]extractTest){extraExtractTests, tldExtractGoTests} {
		for _, test := range testCollection {
			var extractor *FastTLD
			if test.includePrivateSuffix {
				extractor = extractorWithPrivateSuffix
			} else {
				extractor = extractorWithoutPrivateSuffix
			}
			res := extractor.Extract(test.urlParams)

			if output := reflect.DeepEqual(res,
				test.expected); !output {
				t.Errorf("Output %q not equal to expected %q | %q",
					res, test.expected, test.description)
			}
		}
	}
}

func TestFileLastModifiedHours(t *testing.T) {
	filesystem := new(afero.MemMapFs)
	file, _ := afero.TempFile(filesystem, "", "ioutil-test")
	fileinfo, _ := filesystem.Stat(file.Name())
	if hours := fileLastModifiedHours(fileinfo); int(hours) != 0 {
		t.Errorf("Expected hours elapsed since last modification to be 0 immediately after file creation. %f", hours)
	}
	defer file.Close()
}

var benchmarkURLs = []string{
	"https://news.google.com", "https://iupac.org/iupac-announces-the-2021-top-ten-emerging-technologies-in-chemistry/",
	"https://www.google.com/maps/dir/Parliament+Place,+Parliament+House+Of+Singapore,+" +
		"Singapore/Parliament+St,+London,+UK/@25.2440033,33.6721455,4z/data=!3m1!4b1!4m14!4m13!1m5!1m1!1s0x31d" +
		"a19a0abd4d71d:0xeda26636dc4ea1dc!2m2!1d103.8504863!2d1.2891543!1m5!1m1!1s0x487604c5aaa7da5b:0xf13a2" +
		"197d7e7dd26!2m2!1d-0.1260826!2d51.5017061!3e4",
}

var benchmarkURL = benchmarkURLs[0]

// this module
func BenchmarkGoFastTld(b *testing.B) {
	extractorWithoutPrivateSuffix, _ := New(SuffixListParams{
		CacheFilePath:        getTestPSLFilePath(),
		IncludePrivateSuffix: false,
	})
	extractor := extractorWithoutPrivateSuffix

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.Extract(URLParams{
			URL: benchmarkURL})
	}
}

// github.com/jpillora/go-tld
func BenchmarkJPilloraGoTld(b *testing.B) {
	// this module also provides the PORT and PATH subcomponents
	// it cannot handle "+://google.com" and IP addresses
	// it cannot handle urls without scheme component

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tld.Parse(benchmarkURL)
	}
}

// github.com/joeguo/tldextract
func BenchmarkJoeGuoTldExtract(b *testing.B) {
	cache := "/tmp/tld.cache"
	extract, _ := joeguotldextract.New(cache, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extract.Extract(benchmarkURL)
	}
}

// github.com/mjd2021usa/tldextract
func BenchmarkMjd2021USATldExtract(b *testing.B) {
	cache := "/tmp/tld.cache"
	extract, _ := mjd2021usatldextract.New(cache, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extract.Extract(benchmarkURL)
	}
}

// github.com/M507/tlde
func BenchmarkM507Tlde(b *testing.B) {
	// Appears to be the same as github.com/joeguo/tldextract
	cache := "/tmp/tld.cache"
	extract, _ := tlde.New(cache, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extract.Extract(benchmarkURL)
	}
}

/*
// github.com/weppos/publicsuffix-go
func BenchmarkPublicSuffixGo(b *testing.B) {
	// this module cannot handle full URLs with scheme (i.e. https:// ftp:// etc.)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		publicsuffix.Parse(benchmarkURL)
	}
}
*/

/*
// github.com/forease/gotld
func BenchmarkGoTldForeEase(b *testing.B) {
	// does not extract subdomain properly, cannot handle ip addresses
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gotld.GetSubdomain(benchmarkURL, 2048)
	}
}
*/
