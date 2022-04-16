package fasttld

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/joeguo/tldextract"
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

type nestedDictTest struct {
	originalDict dict
	keys         []string
	expected     dict
}

var nestedDictTests = []nestedDictTest{
	{dict{"a": dict{"b": true}}, []string{},
		dict{"a": dict{"b": true}}},
	{dict{"a": dict{"b": true}}, []string{"a"},
		dict{"a": true}},
	{dict{"a": dict{"b": true}}, []string{"c", "d"},
		dict{"a": dict{"b": true}, "c": dict{"d": true}}},
	{dict{"a": dict{"b": dict{"c": true}}}, []string{"a", "b", "c", "d"},
		dict{"a": dict{"b": dict{"c": dict{"_END": true, "d": true}, "d": true}}}},
}

func TestNestedDict(t *testing.T) {

	for _, test := range nestedDictTests {
		nestedDict(test.originalDict, test.keys)
		if output := reflect.DeepEqual(test.originalDict, test.expected); !output {
			t.Errorf("Output %q not equal to expected %q", test.originalDict, test.expected)
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
	expected             dict
}

var newTests = []newTest{
	//{includePrivateSuffix: false, expected: dict{}},
	//{includePrivateSuffix: true, expected: dict{}},
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

		if output := reflect.DeepEqual(extractor.TldTrie,
			test.expected); !output {
			t.Errorf("Output %q not equal to expected %q",
				extractor.TldTrie, test.expected)
		}
	}
}

type extractTest struct {
	includePrivateSuffix bool
	urlParams            UrlParams
	expected             *ExtractResult
	description          string
}

var extraExtractTests = []extractTest{
	{urlParams: UrlParams{Url: "maps.google.com.sg",
		IgnoreSubDomains: true},
		expected: &ExtractResult{
			SubDomain: "", Domain: "google", Suffix: "com.sg",
			RegisteredDomain: "google.com.sg",
		}, description: "Ignore SubDomains",
	},
	{urlParams: UrlParams{Url: "https://brb.i.am.going.to.be.a.fk"},
		expected: &ExtractResult{
			SubDomain: "brb.i.am.going.to", Domain: "be", Suffix: "a.fk",
			RegisteredDomain: "be.a.fk",
		}, description: "Asterisk",
	},
	{includePrivateSuffix: true,
		urlParams: UrlParams{Url: "https://brb.i.am.going.to.be.blogspot.com:5000/a/b/c/d.txt?id=42",
			IgnoreSubDomains: false, ConvertURLToPunyCode: false},
		expected: &ExtractResult{
			SubDomain: "brb.i.am.going.to", Domain: "be", Suffix: "blogspot.com",
			RegisteredDomain: "be.blogspot.com",
		}, description: "Include Private Suffix"},
}

// test cases ported from https://github.com/mjd2021usa/tldextract
var tldExtractGoTests = []extractTest{
	{urlParams: UrlParams{Url: ""}, expected: &ExtractResult{SubDomain: "", Domain: "", Suffix: "", RegisteredDomain: ""}, description: "empty string"},
	{urlParams: UrlParams{Url: "users@myhost.com"}, expected: &ExtractResult{SubDomain: "", Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com"}, description: "user@ address"},
	{urlParams: UrlParams{Url: "mailto:users@myhost.com"}, expected: &ExtractResult{SubDomain: "", Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com"}, description: "email address"},
	{urlParams: UrlParams{Url: "myhost.com:999"}, expected: &ExtractResult{SubDomain: "", Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com"}, description: "host:port"},
	{urlParams: UrlParams{Url: "myhost.com"}, expected: &ExtractResult{SubDomain: "", Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com"}, description: "basic host"},
	{urlParams: UrlParams{Url: "255.255.myhost.com"}, expected: &ExtractResult{SubDomain: "255.255", Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com"}, description: "basic host with numerit subdomains"},
	{urlParams: UrlParams{Url: "https://user:pass@foo.myhost.com:999/some/path?param1=value1&param2=value2"}, expected: &ExtractResult{SubDomain: "foo", Domain: "myhost", Suffix: "com", RegisteredDomain: "myhost.com"}, description: "Full URL with subdomain"},
	{urlParams: UrlParams{Url: "http://www.duckduckgo.com"}, expected: &ExtractResult{SubDomain: "www", Domain: "duckduckgo", Suffix: "com", RegisteredDomain: "duckduckgo.com"}, description: "Full URL with subdomain"},
	{urlParams: UrlParams{Url: "http://duckduckgo.co.uk/path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, expected: &ExtractResult{SubDomain: "", Domain: "duckduckgo", Suffix: "co.uk", RegisteredDomain: "duckduckgo.co.uk"}, description: "Full HTTP URL with no subdomain"},
	{urlParams: UrlParams{Url: "http://big.long.sub.domain.duckduckgo.co.uk/path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, expected: &ExtractResult{SubDomain: "big.long.sub.domain", Domain: "duckduckgo", Suffix: "co.uk", RegisteredDomain: "duckduckgo.co.uk"}, description: "Full HTTP URL with subdomain"},
	{urlParams: UrlParams{Url: "https://duckduckgo.co.uk/path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, expected: &ExtractResult{SubDomain: "", Domain: "duckduckgo", Suffix: "co.uk", RegisteredDomain: "duckduckgo.co.uk"}, description: "Full HTTPS URL with no subdomain"},
	{urlParams: UrlParams{Url: "ftp://peterparker:multipass@mail.duckduckgo.co.uk:666/path?param1=value1&param2=value2&param3=value3&param4=value4&src=https%3A%2F%2Fwww.yahoo.com%2F"}, expected: &ExtractResult{SubDomain: "mail", Domain: "duckduckgo", Suffix: "co.uk", RegisteredDomain: "duckduckgo.co.uk"}, description: "Full ftp URL with subdomain"},
	{urlParams: UrlParams{Url: "git+ssh://www.github.com/"}, expected: &ExtractResult{SubDomain: "www", Domain: "github", Suffix: "com", RegisteredDomain: "github.com"}, description: "Full git+ssh URL with subdomain"},
	// {urlParams: UrlParams{Url: "git+ssh://www.!github.com/"}, expected: &ExtractResult{SubDomain: "", Domain: "", Suffix: "", RegisteredDomain: ""}, description: "Full git+ssh URL with bad domain"},
	{urlParams: UrlParams{Url: "ssh://server.domain.com/"}, expected: &ExtractResult{SubDomain: "server", Domain: "domain", Suffix: "com", RegisteredDomain: "domain.com"}, description: "Full ssh URL with subdomain"},
	// {urlParams: UrlParams{Url: "//server.domain.com/path"}, expected: &ExtractResult{SubDomain: "server", Domain: "domain", Suffix: "com", RegisteredDomain: "domain.com"}, description: "Missing protocol URL with subdomain"},
	{urlParams: UrlParams{Url: "server.domain.com/path"}, expected: &ExtractResult{SubDomain: "server", Domain: "domain", Suffix: "com", RegisteredDomain: "domain.com"}, description: "Full ssh URL with subdomain"},
	{urlParams: UrlParams{Url: "10.10.10.10"}, expected: &ExtractResult{SubDomain: "", Domain: "10.10.10.10", Suffix: "", RegisteredDomain: "10.10.10.10"}, description: "Basic IPv4 Address"},
	{urlParams: UrlParams{Url: "http://10.10.10.10"}, expected: &ExtractResult{SubDomain: "", Domain: "10.10.10.10", Suffix: "", RegisteredDomain: "10.10.10.10"}, description: "Basic IPv4 Address URL"},
	{urlParams: UrlParams{Url: "http://10.10.10.256"}, expected: &ExtractResult{SubDomain: "", Domain: "", Suffix: "", RegisteredDomain: ""}, description: "Basic IPv4 Address URL with bad IP"},
	{urlParams: UrlParams{Url: "http://godaddy.godaddy"}, expected: &ExtractResult{SubDomain: "", Domain: "godaddy", Suffix: "godaddy", RegisteredDomain: "godaddy.godaddy"}, description: "Basic URL"},
	{urlParams: UrlParams{Url: "http://godaddy.godaddy.godaddy"}, expected: &ExtractResult{SubDomain: "godaddy", Domain: "godaddy", Suffix: "godaddy", RegisteredDomain: "godaddy.godaddy"}, description: "Basic URL with subdomain"},
	{urlParams: UrlParams{Url: "http://godaddy.godaddy.co.uk"}, expected: &ExtractResult{SubDomain: "godaddy", Domain: "godaddy", Suffix: "co.uk", RegisteredDomain: "godaddy.co.uk"}, description: "Basic URL with subdomain"},
	// {urlParams: UrlParams{Url: "http://godaddy"}, expected: &ExtractResult{SubDomain: "", Domain: "", Suffix: "", RegisteredDomain: ""}, description: "Basic URL with TLD only"},
	{urlParams: UrlParams{Url: "http://godaddy.cannon-fodder"}, expected: &ExtractResult{SubDomain: "", Domain: "", Suffix: "", RegisteredDomain: ""}, description: "Basic URL with bad TLD"},
	{urlParams: UrlParams{Url: "http://godaddy.godaddy.cannon-fodder"}, expected: &ExtractResult{SubDomain: "", Domain: "", Suffix: "", RegisteredDomain: ""}, description: "Basic URL with subdomainand bad TLD"},

	{urlParams: UrlParams{Url: "http://domainer.个人.hk", ConvertURLToPunyCode: true}, expected: &ExtractResult{SubDomain: "", Domain: "domainer", Suffix: "xn--ciqpn.hk", RegisteredDomain: "domainer.xn--ciqpn.hk"}, description: "Basic URL with mixed international TLD (result in punycode)"},
	{urlParams: UrlParams{Url: "http://domainer.公司.香港", ConvertURLToPunyCode: true}, expected: &ExtractResult{SubDomain: "", Domain: "domainer", Suffix: "xn--55qx5d.xn--j6w193g", RegisteredDomain: "domainer.xn--55qx5d.xn--j6w193g"}, description: "Basic URL with full international TLD (result in punycode)"},
	{urlParams: UrlParams{Url: "http://domainer.个人.hk"}, expected: &ExtractResult{SubDomain: "", Domain: "domainer", Suffix: "个人.hk", RegisteredDomain: "domainer.个人.hk"}, description: "Basic URL with mixed international TLD (result in unicode)"},
	{urlParams: UrlParams{Url: "http://domainer.公司.香港"}, expected: &ExtractResult{SubDomain: "", Domain: "domainer", Suffix: "公司.香港", RegisteredDomain: "domainer.公司.香港"}, description: "Basic URL with full international TLD (result in unicode)"},

	{urlParams: UrlParams{Url: "http://domainer.xn--ciqpn.hk", ConvertURLToPunyCode: true}, expected: &ExtractResult{SubDomain: "", Domain: "domainer", Suffix: "xn--ciqpn.hk", RegisteredDomain: "domainer.xn--ciqpn.hk"}, description: "Basic URL with mixed punycode international TLD (result in punycode)"},
	{urlParams: UrlParams{Url: "http://domainer.xn--55qx5d.xn--j6w193g", ConvertURLToPunyCode: true}, expected: &ExtractResult{SubDomain: "", Domain: "domainer", Suffix: "xn--55qx5d.xn--j6w193g", RegisteredDomain: "domainer.xn--55qx5d.xn--j6w193g"}, description: "Basic URL with full punycode international TLD (result in punycode)"},
	{urlParams: UrlParams{Url: "http://domainer.xn--ciqpn.hk"}, expected: &ExtractResult{SubDomain: "", Domain: "domainer", Suffix: "xn--ciqpn.hk", RegisteredDomain: "domainer.xn--ciqpn.hk"}, description: "Basic URL with mixed punycode international TLD (result in unicode)"},
	{urlParams: UrlParams{Url: "http://domainer.xn--55qx5d.xn--j6w193g"}, expected: &ExtractResult{SubDomain: "", Domain: "domainer", Suffix: "xn--55qx5d.xn--j6w193g", RegisteredDomain: "domainer.xn--55qx5d.xn--j6w193g"}, description: "Basic URL with full punycode international TLD (result in unicode)"},
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

const benchmarkURL = "https://maps.google.com/a/b/c/d/e?id=42"

func BenchmarkFastTld(b *testing.B) {
	extractorWithoutPrivateSuffix, _ := New(SuffixListParams{
		CacheFilePath:        getTestPSLFilePath(),
		IncludePrivateSuffix: false,
	})
	extractor := extractorWithoutPrivateSuffix

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.Extract(UrlParams{
			Url: benchmarkURL})
	}
}

func BenchmarkTldExtract(b *testing.B) {
	cache := "/tmp/tld.cache"
	extract, _ := tldextract.New(cache, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extract.Extract(benchmarkURL)
	}
}
