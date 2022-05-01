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

var schemeTests = []extractTest{
	{urlParams: URLParams{URL: "https://username:password@foo.example.com:999/some/path?param1=value1&param2=葡萄"},
		expected: &ExtractResult{
			Scheme: "https://", UserInfo: "username:password", SubDomain: "foo",
			Domain: "example", Suffix: "com", RegisteredDomain: "example.com",
			Port: "999", Path: "some/path?param1=value1&param2=葡萄"}, description: "Full https URL with SubDomain"},
	{urlParams: URLParams{URL: "http://www.example.com"},
		expected: &ExtractResult{
			Scheme: "http://", SubDomain: "www",
			Domain: "example", Suffix: "com", RegisteredDomain: "example.com"},
		description: "Full http URL with SubDomain no path"},
	{urlParams: URLParams{
		URL: "http://example.co.uk/path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "co.uk",
			RegisteredDomain: "example.co.uk",
			Path:             "path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		description: "Full http URL with no SubDomain"},
	{urlParams: URLParams{
		URL: "http://big.long.sub.domain.example.co.uk/path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		expected: &ExtractResult{Scheme: "http://", SubDomain: "big.long.sub.domain",
			Domain: "example", Suffix: "co.uk", RegisteredDomain: "example.co.uk",
			Path: "path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		description: "Full http URL with SubDomain"},
	{urlParams: URLParams{
		URL: "ftp://username名字:password@mail.example.co.uk:666/path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		expected: &ExtractResult{Scheme: "ftp://", UserInfo: "username名字:password", SubDomain: "mail",
			Domain: "example", Suffix: "co.uk", RegisteredDomain: "example.co.uk", Port: "666",
			Path: "path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		description: "Full ftp URL with SubDomain"},
	{urlParams: URLParams{URL: "git+ssh://www.example.com/"},
		expected: &ExtractResult{Scheme: "git+ssh://", SubDomain: "www",
			Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Full git+ssh URL with SubDomain"},
	{urlParams: URLParams{URL: "ssh://server.example.com/"},
		expected: &ExtractResult{Scheme: "ssh://", SubDomain: "server",
			Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Full ssh URL with SubDomain"},
}
var noSchemeTests = []extractTest{
	{urlParams: URLParams{URL: "users@example.com"}, expected: &ExtractResult{UserInfo: "users", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "UserInfo + Domain | No Scheme"},
	{urlParams: URLParams{URL: "mailto:users@example.com"}, expected: &ExtractResult{UserInfo: "mailto:users", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Mailto | No Scheme"},
	{urlParams: URLParams{URL: "example.com:999"}, expected: &ExtractResult{Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Port: "999"}, description: "Domain + Port | No Scheme"},
	{urlParams: URLParams{URL: "example.com"}, expected: &ExtractResult{Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Domain | No Scheme"},
	{urlParams: URLParams{URL: "255.255.example.com"}, expected: &ExtractResult{SubDomain: "255.255", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Numeric SubDomain + Domain | No Scheme"},
	{urlParams: URLParams{URL: "server.example.com/path"}, expected: &ExtractResult{SubDomain: "server", Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Path: "path"}, description: "SubDomain, Domain and Path | No Scheme"},
}
var ipv4Tests = []extractTest{
	{urlParams: URLParams{URL: "127.0.0.1"},
		expected: &ExtractResult{Domain: "127.0.0.1",
			RegisteredDomain: "127.0.0.1"}, description: "Basic IPv4 Address"},
	{urlParams: URLParams{URL: "http://127.0.0.1:5000"},
		expected: &ExtractResult{
			Scheme: "http://", Domain: "127.0.0.1", RegisteredDomain: "127.0.0.1", Port: "5000"},
		description: "Basic IPv4 Address with Scheme"},
}
var ignoreSubDomainsTests = []extractTest{
	{urlParams: URLParams{URL: "maps.google.com.sg",
		IgnoreSubDomains: true},
		expected: &ExtractResult{
			Domain: "google", Suffix: "com.sg",
			RegisteredDomain: "google.com.sg",
		}, description: "Ignore SubDomain",
	},
}
var privateSuffixTests = []extractTest{
	{includePrivateSuffix: true,
		urlParams: URLParams{URL: "https://brb.i.am.going.to.be.blogspot.com:5000/a/b/c/d.txt?id=42"},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb.i.am.going.to", Domain: "be", Suffix: "blogspot.com",
			RegisteredDomain: "be.blogspot.com", Port: "5000", Path: "a/b/c/d.txt?id=42",
		}, description: "Include Private Suffix"},
	{includePrivateSuffix: true,
		urlParams: URLParams{URL: "global.prod.fastly.net"},
		expected: &ExtractResult{
			Suffix: "global.prod.fastly.net",
		}, description: "Include Private Suffix | Suffix only"},
}
var periodsAndWhiteSpacesTests = []extractTest{
	{urlParams: URLParams{URL: "https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk"},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb\u002ei\u3002am\uff0egoing\uff61to", Domain: "be", Suffix: "a\uff61fk",
			RegisteredDomain: "be\u3002a\uff61fk",
		}, description: "Internationalised period delimiters",
	},
	{urlParams: URLParams{URL: "a\uff61fk"},
		expected: &ExtractResult{Suffix: "a\uff61fk"}, description: "Internationalised period delimiters | Suffix only",
	},
	{urlParams: URLParams{URL: " https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk/a/b/c. \uff61"},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb\u002ei\u3002am\uff0egoing\uff61to", Domain: "be", Suffix: "a\uff61fk",
			RegisteredDomain: "be\u3002a\uff61fk", Path: "a/b/c",
		}, description: "Surrounded by extra whitespace and periods"},

	{urlParams: URLParams{URL: " https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk/a/B/c. \uff61",
		ConvertURLToPunyCode: true},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb.i.am.going.to", Domain: "be", Suffix: "a.fk",
			RegisteredDomain: "be.a.fk", Path: "a/B/c",
		}, description: "Surrounded by extra whitespace and period delimiters | PunyCode"},
}
var invalidTests = []extractTest{
	{urlParams: URLParams{}, expected: &ExtractResult{}, description: "empty string"},
	{urlParams: URLParams{URL: "maps.google.com.sg:8589934592/this/path/will/not/be/parsed"},
		expected: &ExtractResult{
			SubDomain: "maps", Domain: "google", Suffix: "com.sg",
			RegisteredDomain: "google.com.sg",
		}, description: "Invalid Port number"},
	{urlParams: URLParams{URL: "//server.example.com/path"}, expected: &ExtractResult{Scheme: "//", SubDomain: "server", Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Path: "path"}, description: "Missing protocol URL with subdomain"},
	{urlParams: URLParams{URL: "http://temasek"}, expected: &ExtractResult{Scheme: "http://", Suffix: "temasek"}, description: "Basic URL with TLD only"},
	{urlParams: URLParams{URL: "http://temasek.this-tld-cannot-be-real"}, expected: &ExtractResult{Scheme: "http://"}, description: "Basic URL with bad TLD"},
	{urlParams: URLParams{URL: "http://temasek.temasek.this-tld-cannot-be-real"}, expected: &ExtractResult{Scheme: "http://"}, description: "Basic URL with subdomain and bad TLD"},
	{urlParams: URLParams{URL: "http://127.0.0.256"}, expected: &ExtractResult{Scheme: "http://"}, description: "Basic IPv4 Address URL with bad IP"},
	{urlParams: URLParams{URL: "http://a:b@xn--tub-1m9d15sfkkhsifsbqygyujjrw60.com"},
		expected: &ExtractResult{Scheme: "http://", UserInfo: "a:b"}, description: "Invalid punycode Domain"},
	// {urlParams: URLParams{URL: "git+ssh://www.!example.com/"}, expected: &ExtractResult{}, description: "Full git+ssh URL with bad Domain"},
}
var internationalTLDTests = []extractTest{
	{urlParams: URLParams{URL: "http://example.敎育.hk/地图/A/b/C?编号=42", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--lcvr32d.hk", RegisteredDomain: "example.xn--lcvr32d.hk", Path: "地图/A/b/C?编号=42"}, description: "Basic URL with mixed international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://example.обр.срб/地图/A/b/C?编号=42", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--90azh.xn--90a3ac", RegisteredDomain: "example.xn--90azh.xn--90a3ac", Path: "地图/A/b/C?编号=42"}, description: "Basic URL with full international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://example.敎育.hk/地图/A/b/C?编号=42"}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "敎育.hk", RegisteredDomain: "example.敎育.hk", Path: "地图/A/b/C?编号=42"}, description: "Basic URL with mixed international TLD (result in unicode)"},
	{urlParams: URLParams{URL: "http://example.обр.срб/地图/A/b/C?编号=42"}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "обр.срб", RegisteredDomain: "example.обр.срб", Path: "地图/A/b/C?编号=42"}, description: "Basic URL with full international TLD (result in unicode)"},
	{urlParams: URLParams{URL: "http://example.xn--ciqpn.hk/地图/A/b/C?编号=42", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--ciqpn.hk", RegisteredDomain: "example.xn--ciqpn.hk", Path: "地图/A/b/C?编号=42"}, description: "Basic URL with mixed punycode international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://example.xn--90azh.xn--90a3ac/地图/A/b/C?编号=42", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--90azh.xn--90a3ac", RegisteredDomain: "example.xn--90azh.xn--90a3ac", Path: "地图/A/b/C?编号=42"}, description: "Basic URL with full punycode international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://example.xn--ciqpn.hk"}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--ciqpn.hk", RegisteredDomain: "example.xn--ciqpn.hk"}, description: "Basic URL with mixed punycode international TLD (result in unicode)"},
	{urlParams: URLParams{URL: "http://example.xn--90azh.xn--90a3ac"}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--90azh.xn--90a3ac", RegisteredDomain: "example.xn--90azh.xn--90a3ac"}, description: "Basic URL with full punycode international TLD (result in unicode)"},
}
var domainOnlySingleTLDTests = []extractTest{
	{urlParams: URLParams{URL: "https://example.ai/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "ai", RegisteredDomain: "example.ai", Path: "en"}, description: "Domain only + ai"},
	{urlParams: URLParams{URL: "https://example.co/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "co", RegisteredDomain: "example.co", Path: "en"}, description: "Domain only + co"},
	{urlParams: URLParams{URL: "https://example.sg/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "sg", RegisteredDomain: "example.sg", Path: "en"}, description: "Domain only + sg"},
	{urlParams: URLParams{URL: "https://example.tv/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "tv", RegisteredDomain: "example.tv", Path: "en"}, description: "Domain only + tv"},
}
var wildcardTests = []extractTest{
	{urlParams: URLParams{URL: "https://asdf.wwe.ck"},
		expected: &ExtractResult{
			Scheme: "https://", Domain: "asdf", Suffix: "wwe.ck",
			RegisteredDomain: "asdf.wwe.ck"},
		description: "Wildcard rule | *.ck"},
	{urlParams: URLParams{URL: "https://asdf.www.ck"},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "asdf", Domain: "www", Suffix: "ck",
			RegisteredDomain: "www.ck"},
		description: "Wildcard exception rule | !www.ck"},
	{urlParams: URLParams{URL: "https://brb.i.am.going.to.be.a.fk"},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb.i.am.going.to", Domain: "be", Suffix: "a.fk",
			RegisteredDomain: "be.a.fk",
		}, description: "Wildcard rule | *.fk",
	},
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
	for _, testCollection := range []([]extractTest){
		schemeTests,
		noSchemeTests,
		ipv4Tests,
		ignoreSubDomainsTests,
		privateSuffixTests,
		periodsAndWhiteSpacesTests,
		invalidTests,
		internationalTLDTests,
		domainOnlySingleTLDTests,
		wildcardTests,
	} {
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
