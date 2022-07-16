package fasttld

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

var errs = [...]error{
	errors.New("opening square bracket is not first character of hostname"),
	errors.New("closing square bracket present but no opening square bracket"),
	errors.New("invalid square bracket pair"),
	errors.New("incomplete square bracket pair"),
	errors.New("invalid IPv6 address"),
	errors.New("invalid trailing characters after IPv6 address"),
	errors.New("invalid consecutive label separators on left-hand side of partial or full suffix"),
	errors.New("invalid characters in hostname before suffix"),
	errors.New("invalid characters in hostname"),
	errors.New("empty domain"),
	errors.New("invalid port"),
}

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
		t.Errorf("Top level c must have matches map of length 1")
	}
	if _, ok := originalDict.matches["c"].matches["b"]; !ok {
		t.Errorf("Top level c must have b in matches map")
	}
	if !originalDict.matches["c"].end {
		t.Errorf("Top level c must have end = true")
	}
	// Top level a
	if len(originalDict.matches["a"].matches) != 2 {
		t.Errorf("Top level a must have matches map of length 2")
	}
	// a -> d
	if _, ok := originalDict.matches["a"].matches["d"]; !ok {
		t.Errorf("Top level a must have d in matches map")
	}
	if len(originalDict.matches["a"].matches["d"].matches) != 0 {
		t.Errorf("a -> d must have empty matches map")
	}
	// a -> b
	if _, ok := originalDict.matches["a"].matches["b"]; !ok {
		t.Errorf("Top level a must have b in matches map")
	}
	if !originalDict.matches["a"].matches["b"].end {
		t.Errorf("a -> b must have end = true")
	}
	if len(originalDict.matches["a"].matches["b"].matches) != 1 {
		t.Errorf("a -> b must have matches map of length 1")
	}
	// a -> b -> c
	if _, ok := originalDict.matches["a"].matches["b"].matches["c"]; !ok {
		t.Errorf("a -> b must have c in matches map")
	}
	if len(originalDict.matches["a"].matches["b"].matches["c"].matches) != 0 {
		t.Errorf("a -> b -> c must have empty matches map")
	}
	if !originalDict.matches["a"].end {
		t.Errorf("Top level a must have end = true")
	}
	// d -> f
	if originalDict.matches["d"].end {
		t.Errorf("Top level d must have end = false")
	}
	if originalDict.matches["d"].matches["f"].end {
		t.Errorf("d -> f must have end = false")
	}
	if len(originalDict.matches["d"].matches["f"].matches) != 0 {
		t.Errorf("d -> f must have empty matches map")
	}
}

func TestTrieConstructInvalidPath(t *testing.T) {
	if _, err := trieConstruct(false, fmt.Sprintf("test%sthis_file_does_not_exist.dat", string(os.PathSeparator))); err == nil {
		t.Errorf("error returned by trieConstruct should not be nil")
	}
}

func TestTrie(t *testing.T) {
	trie, err := trieConstruct(false, fmt.Sprintf("test%smini_public_suffix_list.dat", string(os.PathSeparator)))
	if err != nil {
		t.Errorf("trieConstruct failed | %q", err)
	}
	if lenTrieMatches := len(trie.matches); lenTrieMatches != 2 {
		t.Errorf("Expected top level Trie matches map length of 2. Got %d.", lenTrieMatches)
	}
	for _, tld := range []string{"ac", "ck"} {
		if _, ok := trie.matches[tld]; !ok {
			t.Errorf("Top level %q must exist", tld)
		}
	}
	if !trie.matches["ac"].end {
		t.Errorf("Top level ac must have end = true")
	}
	if trie.matches["ck"].end {
		t.Errorf("Top level ck must have end = false")
	}
	if len(trie.matches["ck"].matches) != 2 {
		t.Errorf("Top level ck must have matches map of length 2")
	}
	if _, ok := trie.matches["ck"].matches["*"]; !ok {
		t.Errorf("Top level ck must have * in matches map")
	}
	if len(trie.matches["ck"].matches["*"].matches) != 0 {
		t.Errorf("ck -> * must have empty matches map")
	}
	if _, ok := trie.matches["ck"].matches["!www"]; !ok {
		t.Errorf("Top level ck must have !www in matches map")
	}
	if len(trie.matches["ck"].matches["!www"].matches) != 0 {
		t.Errorf("ck -> !www must have empty matches map")
	}
	for _, tld := range []string{"com", "edu", "gov", "net", "mil", "org"} {
		if _, ok := trie.matches["ac"].matches[tld]; !ok {
			t.Errorf("Top level ac must have %q in matches map", tld)
		}
		if len(trie.matches["ac"].matches[tld].matches) != 0 {
			t.Errorf("ac -> %q must have empty matches map", tld)
		}
	}
}

type newTest struct {
	cacheFilePath        string
	includePrivateSuffix bool
	expected             int
}

var newTests = []newTest{
	{cacheFilePath: fmt.Sprintf("test%spublic_suffix_list.dat", string(os.PathSeparator)), includePrivateSuffix: false, expected: 1656},
	{cacheFilePath: fmt.Sprintf("test%spublic_suffix_list.dat", string(os.PathSeparator)), includePrivateSuffix: true, expected: 1674},
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
	err                  error
	description          string
}

var schemeTests = []extractTest{
	{urlParams: URLParams{URL: "h://example.com"},
		expected: &ExtractResult{
			Scheme: "h://", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Single character Scheme"},
	{urlParams: URLParams{URL: "hTtPs://example.com"},
		expected: &ExtractResult{
			Scheme: "hTtPs://", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Capitalised Scheme"},
	{urlParams: URLParams{URL: "git-ssh://example.com"},
		expected: &ExtractResult{
			Scheme: "git-ssh://", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Scheme with -"},
	{urlParams: URLParams{URL: "https://username:password@foo.example.com:999/some/path?param1=value1&param2=葡萄"},
		expected: &ExtractResult{
			Scheme: "https://", UserInfo: "username:password", SubDomain: "foo",
			Domain: "example", Suffix: "com", RegisteredDomain: "example.com",
			Port: "999", Path: "/some/path?param1=value1&param2=葡萄"}, description: "Full https URL with SubDomain"},
	{urlParams: URLParams{URL: "http://www.example.com"},
		expected: &ExtractResult{
			Scheme: "http://", SubDomain: "www",
			Domain: "example", Suffix: "com", RegisteredDomain: "example.com"},
		description: "Full http URL with SubDomain no path"},
	{urlParams: URLParams{
		URL: "http://example.co.uk/path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "co.uk",
			RegisteredDomain: "example.co.uk",
			Path:             "/path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		description: "Full http URL with no SubDomain"},
	{urlParams: URLParams{
		URL: "http://big.long.sub.domain.example.co.uk/path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		expected: &ExtractResult{Scheme: "http://", SubDomain: "big.long.sub.domain",
			Domain: "example", Suffix: "co.uk", RegisteredDomain: "example.co.uk",
			Path: "/path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		description: "Full http URL with SubDomain"},
	{urlParams: URLParams{
		URL: "ftp://username名字:password@mail.example.co.uk:666/path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		expected: &ExtractResult{Scheme: "ftp://", UserInfo: "username名字:password", SubDomain: "mail",
			Domain: "example", Suffix: "co.uk", RegisteredDomain: "example.co.uk", Port: "666",
			Path: "/path?param1=value1&param2=葡萄&param3=value3&param4=value4&src=https%3A%2F%2Fwww.example.net%2F"},
		description: "Full ftp URL with SubDomain"},
	{urlParams: URLParams{URL: "git+ssh://www.example.com/"},
		expected: &ExtractResult{Scheme: "git+ssh://", SubDomain: "www",
			Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Path: "/"}, description: "Full git+ssh URL with SubDomain"},
	{urlParams: URLParams{URL: "ssh://server.example.com/"},
		expected: &ExtractResult{Scheme: "ssh://", SubDomain: "server",
			Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Path: "/"}, description: "Full ssh URL with SubDomain"},
	{urlParams: URLParams{URL: "http://www.www.net"},
		expected: &ExtractResult{Scheme: "http://", SubDomain: "www",
			Domain: "www", Suffix: "net", RegisteredDomain: "www.net"}, description: "Multiple www"},
}
var noSchemeTests = []extractTest{
	{urlParams: URLParams{URL: "localhost"}, expected: &ExtractResult{Domain: "localhost"}, description: "localhost"},
	{urlParams: URLParams{URL: "org"}, expected: &ExtractResult{Suffix: "org"}, err: errs[9], description: "Single TLD | Suffix Only"},
	{urlParams: URLParams{URL: "co.th"}, expected: &ExtractResult{Suffix: "co.th"}, err: errs[9], description: "Double TLD | Suffix Only"},
	{urlParams: URLParams{URL: "users@example.com"}, expected: &ExtractResult{UserInfo: "users", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "UserInfo + Domain | No Scheme"},
	{urlParams: URLParams{URL: "mailto:users@example.com"}, expected: &ExtractResult{UserInfo: "mailto:users", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Mailto | No Scheme"},
	{urlParams: URLParams{URL: "example.com:999"}, expected: &ExtractResult{Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Port: "999"}, description: "Domain + Port | No Scheme"},
	{urlParams: URLParams{URL: "example.com"}, expected: &ExtractResult{Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Domain | No Scheme"},
	{urlParams: URLParams{URL: "255.255.example.com"}, expected: &ExtractResult{SubDomain: "255.255", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "Numeric SubDomain + Domain | No Scheme"},
	{urlParams: URLParams{URL: "server.example.com/path"}, expected: &ExtractResult{SubDomain: "server", Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Path: "/path"}, description: "SubDomain, Domain and Path | No Scheme"},
}
var userInfoTests = []extractTest{
	{urlParams: URLParams{URL: "https://username@example.com"}, expected: &ExtractResult{Scheme: "https://",
		UserInfo: "username", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "username"},
	{urlParams: URLParams{URL: "https://password@example.com"}, expected: &ExtractResult{Scheme: "https://",
		UserInfo: "password", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "username + password"},
	{urlParams: URLParams{URL: "https://:password@example.com"}, expected: &ExtractResult{Scheme: "https://",
		UserInfo: ":password", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "colon but empty username"},
	{urlParams: URLParams{URL: "https://username:@example.com"}, expected: &ExtractResult{Scheme: "https://",
		UserInfo: "username:", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "colon but empty password"},
	{urlParams: URLParams{URL: "https://usern@me:password@example.com"}, expected: &ExtractResult{Scheme: "https://",
		UserInfo: "usern@me:password", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "@ in username"},
	{urlParams: URLParams{URL: "https://usern@me:p@ssword@example.com"}, expected: &ExtractResult{Scheme: "https://",
		UserInfo: "usern@me:p@ssword", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "@ in password"},
	{urlParams: URLParams{URL: "https://usern@me:@example.com"}, expected: &ExtractResult{Scheme: "https://",
		UserInfo: "usern@me:", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "colon but empty password; @ in username"},
	{urlParams: URLParams{URL: "https://:p@ssword@example.com"}, expected: &ExtractResult{Scheme: "https://",
		UserInfo: ":p@ssword", Domain: "example", Suffix: "com", RegisteredDomain: "example.com"}, description: "colon but empty username; @ in password"},
	{urlParams: URLParams{URL: "https://usern@m%40e:password@example.com/p@th?q=@go"}, expected: &ExtractResult{Scheme: "https://",
		UserInfo: "usern@m%40e:password", Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Path: "/p@th?q=@go"}, description: "@ in UserInfo and Path"},
}
var ipv4Tests = []extractTest{
	{urlParams: URLParams{URL: "127.0.0.1"},
		expected: &ExtractResult{Domain: "127.0.0.1",
			RegisteredDomain: "127.0.0.1"}, description: "Basic IPv4 Address"},
	{urlParams: URLParams{URL: "http://127.0.0.1:5000"},
		expected: &ExtractResult{
			Scheme: "http://", Domain: "127.0.0.1", RegisteredDomain: "127.0.0.1", Port: "5000"},
		description: "Basic IPv4 Address with Scheme and Port"},
	{urlParams: URLParams{URL: "127\uff0e0\u30020\uff611"},
		expected: &ExtractResult{Domain: "127\uff0e0\u30020\uff611",
			RegisteredDomain: "127\uff0e0\u30020\uff611"}, description: "Basic IPv4 Address | Internationalised label separators"},
	{urlParams: URLParams{URL: "http://127\uff0e0\u30020\uff611:5000"},
		expected: &ExtractResult{Scheme: "http://", Domain: "127\uff0e0\u30020\uff611", Port: "5000",
			RegisteredDomain: "127\uff0e0\u30020\uff611"}, description: "Basic IPv4 Address with Scheme and Port | Internationalised label separators"},
}
var ipv6Tests = []extractTest{
	{urlParams: URLParams{URL: "[aBcD:ef01:2345:6789:aBcD:ef01:2345:6789]"},
		expected: &ExtractResult{Domain: "aBcD:ef01:2345:6789:aBcD:ef01:2345:6789",
			RegisteredDomain: "aBcD:ef01:2345:6789:aBcD:ef01:2345:6789"}, description: "Basic IPv6 Address"},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:ef01:2345:6789]:5000"},
		expected: &ExtractResult{
			Scheme: "http://", Domain: "aBcD:ef01:2345:6789:aBcD:ef01:2345:6789", RegisteredDomain: "aBcD:ef01:2345:6789:aBcD:ef01:2345:6789", Port: "5000"},
		description: "Basic IPv6 Address with Scheme and Port"},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:ef01:127.0.0.1]:5000"},
		expected: &ExtractResult{
			Scheme: "http://", Domain: "aBcD:ef01:2345:6789:aBcD:ef01:127.0.0.1", RegisteredDomain: "aBcD:ef01:2345:6789:aBcD:ef01:127.0.0.1", Port: "5000"},
		description: "Basic IPv6 Address + trailing IPv4 address with Scheme and Port"},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:ef01:127\uff0e0\u30020\uff611]:5000"},
		expected: &ExtractResult{
			Scheme: "http://", Domain: "aBcD:ef01:2345:6789:aBcD:ef01:127\uff0e0\u30020\uff611", RegisteredDomain: "aBcD:ef01:2345:6789:aBcD:ef01:127\uff0e0\u30020\uff611", Port: "5000"},
		description: "Basic IPv6 Address + trailing IPv4 address with Scheme and Port | Internationalised label separators"},
	{urlParams: URLParams{URL: "http://[::2345:6789:aBcD:ef01:2345:678]:5000"},
		expected: &ExtractResult{Scheme: "http://", Domain: "::2345:6789:aBcD:ef01:2345:678",
			RegisteredDomain: "::2345:6789:aBcD:ef01:2345:678", Port: "5000"},
		description: "Basic IPv6 Address with Scheme and Port | have leading ellipsis"},
	{urlParams: URLParams{URL: "http://[::]:5000"},
		expected: &ExtractResult{Scheme: "http://", Domain: "::",
			RegisteredDomain: "::", Port: "5000"},
		description: "Basic IPv6 Address with Scheme and Port | only ellipsis"},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:ef01::]:5000"},
		expected: &ExtractResult{Scheme: "http://", Domain: "aBcD:ef01:2345:6789:aBcD:ef01::",
			RegisteredDomain: "aBcD:ef01:2345:6789:aBcD:ef01::", Port: "5000"},
		description: "Basic IPv6 Address with Scheme and Port bad IP with even number of trailing empty hextets"},
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
			RegisteredDomain: "be.blogspot.com", Port: "5000", Path: "/a/b/c/d.txt?id=42",
		}, description: "Include Private Suffix"},
	{includePrivateSuffix: true,
		urlParams: URLParams{URL: "global.prod.fastly.net"},
		expected: &ExtractResult{
			Suffix: "global.prod.fastly.net",
		}, err: errs[9], description: "Include Private Suffix | Suffix only"},
}
var periodsAndWhiteSpacesTests = []extractTest{
	{urlParams: URLParams{URL: "http://127.0.0.1.."},
		expected: &ExtractResult{Scheme: "http://", Domain: "127.0.0.1", RegisteredDomain: "127.0.0.1"}, description: "Consecutive label separators after IPv4 address",
	},
	{urlParams: URLParams{URL: "http://127\uff0e0\u30020\uff611..:5000"},
		expected: &ExtractResult{Scheme: "http://", Domain: "127\uff0e0\u30020\uff611",
			Port: "5000", RegisteredDomain: "127\uff0e0\u30020\uff611"}, description: "Consecutive label separators between IPv4 address and Port",
	},
	{urlParams: URLParams{URL: "http://127.0.0.1  "},
		expected: &ExtractResult{Scheme: "http://", Domain: "127.0.0.1", RegisteredDomain: "127.0.0.1"}, description: "Spaces after IPv4 address",
	},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:ef01:2345:6789]  "},
		expected: &ExtractResult{Scheme: "http://", Domain: "aBcD:ef01:2345:6789:aBcD:ef01:2345:6789",
			RegisteredDomain: "aBcD:ef01:2345:6789:aBcD:ef01:2345:6789"}, description: "Spaces after IPv6 address",
	},
	{urlParams: URLParams{URL: "localhost.\u3002"}, expected: &ExtractResult{Domain: "localhost"}, description: "localhost with trailing periods"},
	{urlParams: URLParams{URL: "https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk\uff0e\u002e\u3002"},
		expected: &ExtractResult{Scheme: "https://", SubDomain: "brb\u002ei\u3002am\uff0egoing\uff61to", Domain: "be",
			Suffix: "a\uff61fk", RegisteredDomain: "be\u3002a\uff61fk"},
		description: "Consecutive label separators after Suffix",
	},
	{urlParams: URLParams{URL: "https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk"},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb\u002ei\u3002am\uff0egoing\uff61to", Domain: "be", Suffix: "a\uff61fk",
			RegisteredDomain: "be\u3002a\uff61fk",
		}, description: "Internationalised label separators",
	},
	{urlParams: URLParams{URL: "a\uff61fk"},
		expected: &ExtractResult{Suffix: "a\uff61fk"}, err: errs[9], description: "Internationalised label separators | Suffix only",
	},
	{urlParams: URLParams{URL: " https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk/a/b/c. \uff61 "},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb\u002ei\u3002am\uff0egoing\uff61to", Domain: "be", Suffix: "a\uff61fk",
			RegisteredDomain: "be\u3002a\uff61fk", Path: "/a/b/c. \uff61",
		}, description: "Surrounded by extra whitespace"},

	{urlParams: URLParams{URL: " https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk/a/B/c. \uff61 ",
		ConvertURLToPunyCode: true},
		expected: &ExtractResult{
			Scheme: "https://", SubDomain: "brb.i.am.going.to", Domain: "be", Suffix: "a.fk",
			RegisteredDomain: "be.a.fk", Path: "/a/B/c. \uff61",
		}, description: "Surrounded by extra whitespace | PunyCode"},
	{urlParams: URLParams{URL: "http://1.1.1.1 &@2.2.2.2:33/4.4.4.4?1.1.1.1# @3.3.3.3/"},
		expected: &ExtractResult{
			Scheme: "http://", UserInfo: "1.1.1.1 &", Domain: "2.2.2.2",
			RegisteredDomain: "2.2.2.2", Port: "33", Path: "/4.4.4.4?1.1.1.1# @3.3.3.3/",
		}, description: "Whitespace in UserInfo"},
}
var invalidTests = []extractTest{
	{urlParams: URLParams{URL: "localhost!"}, expected: &ExtractResult{}, err: errs[8], description: "localhost + invalid character !"},
	{urlParams: URLParams{URL: "localhost-"}, expected: &ExtractResult{}, err: errs[8], description: "localhost + invalid character -"},
	{urlParams: URLParams{}, expected: &ExtractResult{}, err: errs[9], description: "empty string"},
	{urlParams: URLParams{URL: "https://"}, expected: &ExtractResult{Scheme: "https://"}, err: errs[9], description: "Scheme only"},
	{urlParams: URLParams{URL: "1b://example.com"}, expected: &ExtractResult{}, err: errs[10], description: "Scheme beginning with non-alphabet (parser unsuccessfully tries to interpret runes after colon as port"},
	{urlParams: URLParams{URL: "maps.google.com.sg:8589934592/this/path/will/not/be/parsed"}, expected: &ExtractResult{}, err: errs[10], description: "Invalid Port number"},
	{urlParams: URLParams{URL: "http://.\u3002127.0.0.1"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Consecutive label separators before IPv4 address",
	},
	{urlParams: URLParams{URL: "http://.\u3002[aBcD:ef01:2345:6789:aBcD:ef01:2345:6789]"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[0], description: "Consecutive label separators before IPv6 address",
	},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:ef01:2345:6789].."},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[5], description: "Consecutive label separators after IPv6 address",
	},
	{urlParams: URLParams{URL: "http://example.com :50"},
		expected: &ExtractResult{Scheme: "http://", Port: "50"}, err: errs[8], description: "Spaces between domain and Port/Path",
	},
	{urlParams: URLParams{URL: "http://  127.0.0.1"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Spaces before IPv4 address",
	},
	{urlParams: URLParams{URL: "http://127.0.0.1  :50"},
		expected: &ExtractResult{Scheme: "http://", Port: "50"}, err: errs[8], description: "Spaces between IPv4 address and Port/Path",
	},
	{urlParams: URLParams{URL: "http://  [aBcD:ef01:2345:6789:aBcD:ef01:2345:6789]"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[0], description: "Spaces before IPv6 address",
	},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:ef01:2345:6789]  :50"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[5], description: "Spaces between IPv6 address and Port/Path",
	},
	{urlParams: URLParams{URL: "https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61\u3002fk"},
		expected: &ExtractResult{Scheme: "https://"}, err: errs[6], description: "Consecutive label separators within Suffix",
	},
	{urlParams: URLParams{URL: ".\u3002a\uff61fk"}, expected: &ExtractResult{}, err: errs[8], description: "TLD only, multiple leading label separators"},
	{urlParams: URLParams{URL: "https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe.\u3002a\uff61fk"}, expected: &ExtractResult{Scheme: "https://"}, err: errs[8], description: "Consecutive label separators between Domain and Suffix"},
	{urlParams: URLParams{URL: "https://brb\u002ei\u3002am\uff0egoing\uff61to.\uff0ebe\u3002a\uff61fk"}, expected: &ExtractResult{Scheme: "https://"}, err: errs[8], description: "Consecutive label separators between SubDomain and Domain"},
	{urlParams: URLParams{URL: "https://brb\u002ei\u3002.am.\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk"}, expected: &ExtractResult{Scheme: "https://"}, err: errs[8], description: "Consecutive label separators within SubDomain"},
	{urlParams: URLParams{URL: "https://\uff0eexample.com"}, expected: &ExtractResult{Scheme: "https://"}, err: errs[8], description: "Hostname starting with label separator"},
	{urlParams: URLParams{URL: "//server.example.com/path"}, expected: &ExtractResult{Scheme: "//", SubDomain: "server", Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Path: "/path"}, description: "Double-slash only Scheme with subdomain"},
	{urlParams: URLParams{URL: "http://temasek"}, expected: &ExtractResult{Scheme: "http://", Suffix: "temasek"}, err: errs[9], description: "Basic URL with TLD only"},
	{urlParams: URLParams{URL: "http://temasek.this-tld-cannot-be-real"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "temasek", Domain: "this-tld-cannot-be-real"}, description: "Basic URL with bad TLD"},
	{urlParams: URLParams{URL: "http://temasek.temasek.this-tld-cannot-be-real"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "temasek.temasek", Domain: "this-tld-cannot-be-real"}, description: "Basic URL with subdomain and bad TLD"},
	{urlParams: URLParams{URL: "http://127.0.0.256"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "127.0.0", Domain: "256"}, description: "Basic IPv4 Address URL with bad IP"},
	{urlParams: URLParams{URL: "http://127\uff0e0\u30020\uff61256:5000"},
		expected: &ExtractResult{Scheme: "http://", SubDomain: "127\uff0e0\u30020", Port: "5000",
			Domain: "256"}, description: "Basic IPv4 Address with Scheme and Port and bad IP | Internationalised label separators"},
	{urlParams: URLParams{URL: "http://192.168.01.1:5000"},
		expected:    &ExtractResult{Scheme: "http://", SubDomain: "192.168.01", Domain: "1", Port: "5000"},
		description: "Basic IPv4 Address with Scheme and Port and bad IP | octet with leading zero"},
	{urlParams: URLParams{URL: "http://a:b@xn--tub-1m9d15sfkkhsifsbqygyujjrw60.com"},
		expected: &ExtractResult{Scheme: "http://", UserInfo: "a:b"}, err: errors.New("idna: invalid label \"tub-1m9d15sfkkhsifsbqygyujjrw60\""), description: "Invalid punycode Domain"},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:ef01:2345:6789:5000"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[3],
		description: "Basic IPv6 Address with Scheme and Port with no closing bracket"},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:::]:5000"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[4],
		description: "Basic IPv6 Address with Scheme and Port and bad IP | odd number of empty hextets"},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:ef01:2345:fffffffffffffffff]:5000"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[4],
		description: "Basic IPv6 Address with Scheme and Port and bad IP | hextet too big"},
	{urlParams: URLParams{URL: "http://[aBcD:ef01:2345:6789:aBcD:ef01:127\uff0e256\u30020\uff611]:5000"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[4],
		description: "Basic IPv6 Address + trailing IPv4 address with Scheme and Port and bad IPv4 | Internationalised label separators"},
	{urlParams: URLParams{URL: "http://["},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[3],
		description: "Single opening square bracket"},
	{urlParams: URLParams{URL: "http://a["},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[0],
		description: "Single opening square bracket after alphabet"},
	{urlParams: URLParams{URL: "http://]"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[1],
		description: "Single closing square bracket"},
	{urlParams: URLParams{URL: "http://a]"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[1],
		description: "Single closing square bracket after alphabet"},
	{urlParams: URLParams{URL: "http://]["},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[1],
		description: "closing square bracket before opening square bracket"},
	{urlParams: URLParams{URL: "http://a]["},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[1],
		description: "closing square bracket before opening square bracket after alphabet"},
	{urlParams: URLParams{URL: "http://[]"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[4],
		description: "Empty pair of square brackets"},
	{urlParams: URLParams{URL: "http://a[]"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[0],
		description: "Empty pair of square brackets after alphabet"},
	{urlParams: URLParams{URL: "http://a[127.0.0.1]"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[0],
		description: "IPv4 in square brackets after alphabet"},
	{urlParams: URLParams{URL: "http://a[aBcD:ef01:2345:6789:aBcD:ef01:127\uff0e255\u30020\uff611]"},
		expected: &ExtractResult{Scheme: "http://"}, err: errs[0],
		description: "IPv6 in square brackets after alphabet"},
	{urlParams: URLParams{URL: "http://[127.0.0.1]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "IPv4 in square brackets"},

	// Test cases from net/ip-test.go
	{urlParams: URLParams{URL: "http://[-0.0.0.0]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[0.-1.0.0]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[0.0.-2.0]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[0.0.0.-3]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[127.0.0.256]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[abc]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[123:]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[fe80::1%lo0]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[fe80::1%911]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[a1:a2:a3:a4::b1:b2:b3:b4]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[127.001.002.003]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[::ffff:127.001.002.003]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[123.000.000.000]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[1.2..4]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "http://[0123.0.0.1]"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "net/ip-test.go"},
	{urlParams: URLParams{URL: "git+ssh://www.!example.com/"}, expected: &ExtractResult{Scheme: "git+ssh://", Path: "/"}, err: errs[8], description: "Full git+ssh URL with bad Domain"},
}
var internationalTLDTests = []extractTest{
	{urlParams: URLParams{URL: "https://𝖊𝖝𝖆𝖒𝖕𝖑𝖊.𝖈𝖔𝖒.𝖘𝖌", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "com.sg", RegisteredDomain: "example.com.sg"}},
	{urlParams: URLParams{URL: "http://example.敎育.hk/地图/A/b/C?编号=42", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--lcvr32d.hk", RegisteredDomain: "example.xn--lcvr32d.hk", Path: "/地图/A/b/C?编号=42"}, description: "Basic URL with mixed international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://example.обр.срб/地图/A/b/C?编号=42", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--90azh.xn--90a3ac", RegisteredDomain: "example.xn--90azh.xn--90a3ac", Path: "/地图/A/b/C?编号=42"}, description: "Basic URL with full international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://example.敎育.hk/地图/A/b/C?编号=42"}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "敎育.hk", RegisteredDomain: "example.敎育.hk", Path: "/地图/A/b/C?编号=42"}, description: "Basic URL with mixed international TLD (result in unicode)"},
	{urlParams: URLParams{URL: "http://example.обр.срб/地图/A/b/C?编号=42"}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "обр.срб", RegisteredDomain: "example.обр.срб", Path: "/地图/A/b/C?编号=42"}, description: "Basic URL with full international TLD (result in unicode)"},
	{urlParams: URLParams{URL: "http://example.xn--ciqpn.hk/地图/A/b/C?编号=42", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--ciqpn.hk", RegisteredDomain: "example.xn--ciqpn.hk", Path: "/地图/A/b/C?编号=42"}, description: "Basic URL with mixed punycode international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://example.xn--90azh.xn--90a3ac/地图/A/b/C?编号=42", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--90azh.xn--90a3ac", RegisteredDomain: "example.xn--90azh.xn--90a3ac", Path: "/地图/A/b/C?编号=42"}, description: "Basic URL with full punycode international TLD (result in punycode)"},
	{urlParams: URLParams{URL: "http://example.xn--ciqpn.hk"}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--ciqpn.hk", RegisteredDomain: "example.xn--ciqpn.hk"}, description: "Basic URL with mixed punycode international TLD (no further conversion to punycode)"},
	{urlParams: URLParams{URL: "http://example.xn--90azh.xn--90a3ac"}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "xn--90azh.xn--90a3ac", RegisteredDomain: "example.xn--90azh.xn--90a3ac"}, description: "Basic URL with full punycode international TLD (no further conversion to punycode)"},
	{urlParams: URLParams{URL: "http://xN--h1alffa9f.xn--90azh.xn--90a3ac"}, expected: &ExtractResult{Scheme: "http://", Domain: "xN--h1alffa9f", Suffix: "xn--90azh.xn--90a3ac", RegisteredDomain: "xN--h1alffa9f.xn--90azh.xn--90a3ac"}, description: "Mixed case Punycode Domain with full punycode international TLD (no further conversion to punycode) See: https://github.com/golang/go/issues/48778"},
	{urlParams: URLParams{URL: "http://xN--h1alffa9f.xn--90azh.xn--90a3ac", ConvertURLToPunyCode: true}, expected: &ExtractResult{Scheme: "http://", Domain: "xn--h1alffa9f", Suffix: "xn--90azh.xn--90a3ac", RegisteredDomain: "xn--h1alffa9f.xn--90azh.xn--90a3ac"}, description: "Mixed case Punycode Domain with full punycode international TLD (with further conversion to punycode)"},
}
var domainOnlySingleTLDTests = []extractTest{
	{urlParams: URLParams{URL: "https://example.ai/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "ai", RegisteredDomain: "example.ai", Path: "/en"}, description: "Domain only + ai"},
	{urlParams: URLParams{URL: "https://example.co/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "co", RegisteredDomain: "example.co", Path: "/en"}, description: "Domain only + co"},
	{urlParams: URLParams{URL: "https://example.sg/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "sg", RegisteredDomain: "example.sg", Path: "/en"}, description: "Domain only + sg"},
	{urlParams: URLParams{URL: "https://example.tv/en"}, expected: &ExtractResult{Scheme: "https://", Domain: "example", Suffix: "tv", RegisteredDomain: "example.tv", Path: "/en"}, description: "Domain only + tv"},
}
var pathTests = []extractTest{
	{urlParams: URLParams{URL: "http://www.example.com/this:that"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "www", Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Path: "/this:that"}, description: "Colon in Path"},
	{urlParams: URLParams{URL: "http://example.com/oid/[order_id]"}, expected: &ExtractResult{Scheme: "http://", Domain: "example", Suffix: "com", RegisteredDomain: "example.com", Path: "/oid/[order_id]"}, description: "Square brackets in Path"},
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
var lookoutTests = []extractTest{ // some tests from lookout.net
	{urlParams: URLParams{URL: "http://GOO\u200b\u2060\ufeffgoo.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Invalid chars"},
	{urlParams: URLParams{URL: "http://\u0646\u0627\u0645\u0647\u200c\u0627\u06cc.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Invalid chars"},
	{urlParams: URLParams{URL: "http://\u0000\u0dc1\u0dca\u200d\u0dbb\u0dd3.com.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Invalid chars"},
	{urlParams: URLParams{URL: "http://\u0dc1\u0dca\u200d\u0dbb\u0dd3.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Invalid chars"},
	{urlParams: URLParams{URL: "http://look\ufeffout.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Invalid chars"},
	{urlParams: URLParams{URL: "http://www\u00A0.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Invalid chars"},
	{urlParams: URLParams{URL: "http://\u1680.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Invalid chars"},
	{urlParams: URLParams{URL: "%68%74%74%70%3a%2f%2f%77%77%77%2e%65%78%61%6d%70%6c%65%2e%63%6f%6d%2f.urltest.lookout.net"}, expected: &ExtractResult{
		SubDomain: "%68%74%74%70%3a%2f%2f%77%77%77%2e%65%78%61%6d%70%6c%65%2e%63%6f%6d%2f.urltest", Domain: "lookout",
		Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http%3a%2f%2f%77%77%77%2e%65%78%61%6d%70%6c%65%2e%63%6f%6d%2f.urltest.lookout.net"}, expected: &ExtractResult{
		SubDomain: "http%3a%2f%2f%77%77%77%2e%65%78%61%6d%70%6c%65%2e%63%6f%6d%2f.urltest", Domain: "lookout",
		Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://%25.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://",
		SubDomain: "%25.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://%25DOMAIN:foobar@urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", UserInfo: "%25DOMAIN:foobar",
		SubDomain: "urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded UserInfo"},
	{urlParams: URLParams{URL: "http://%30%78%63%30%2e%30%32%35%30.01%2e.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "%30%78%63%30%2e%30%32%35%30.01%2e.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://%30%78%63%30%2e%30%32%35%30.01.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "%30%78%63%30%2e%30%32%35%30.01.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://%3g%78%63%30%2e%30%32%35%30%2E.01.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "%3g%78%63%30%2e%30%32%35%30%2E.01.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://%77%77%77%2e%65%78%61%6d%70%6c%65%2e%63%6f%6d.urltest.lookout.net%3a%38%30"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "%77%77%77%2e%65%78%61%6d%70%6c%65%2e%63%6f%6d.urltest.lookout", Domain: "net%3a%38%30"}, description: "Percentage encoded SubDomain and Domain"},
	{urlParams: URLParams{URL: "http://%A1%C1.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "%A1%C1.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://%E4%BD%A0%E5%A5%BD\u4f60\u597d.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "%E4%BD%A0%E5%A5%BD\u4f60\u597d.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded and Unicode SubDomain"},
	{urlParams: URLParams{URL: "http://%ef%b7%90zyx.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "%ef%b7%90zyx.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://%ef%bc%85%ef%bc%90%ef%bc%90.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "%ef%bc%85%ef%bc%90%ef%bc%90.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://%ef%bc%85%ef%bc%94%ef%bc%91.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "%ef%bc%85%ef%bc%94%ef%bc%91.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://%zz%66%a.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "%zz%66%a.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://-foo.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Start with dash"},
	{urlParams: URLParams{URL: "http:////////user:@urltest.lookout.net?foo"}, expected: &ExtractResult{Scheme: "http:////////", UserInfo: "user:", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "?foo", RegisteredDomain: "lookout.net"}, description: "Multiple slashes in Scheme"},
	{urlParams: URLParams{URL: "http://192.168.0.1 hello.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Space in SubDomain"},
	{urlParams: URLParams{URL: "http://192.168.0.257.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "192.168.0.257.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "IPv4 Address in SubDomain"},
	{urlParams: URLParams{URL: "http://B\u00fccher.de.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "B\u00fccher.de.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://GOO \u3000goo.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Space in SubDomain"},
	{urlParams: URLParams{URL: "http://Goo%20 goo%7C|.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Space in SubDomain"},
	{urlParams: URLParams{URL: "http://[google.com.].urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "Square Brackets in SubDomain"},
	{urlParams: URLParams{URL: "http://[urltest.lookout.net]/"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[4], description: "Square brackets but not IPv6"},
	{urlParams: URLParams{URL: "http://\u001f.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Control Character in SubDomain"},
	{urlParams: URLParams{URL: "http://\u0378.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u0378.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode U+0378"},
	{urlParams: URLParams{URL: "http://\u03b2\u03cc\u03bb\u03bf\u03c2.com.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u03b2\u03cc\u03bb\u03bf\u03c2.com.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\u03b2\u03cc\u03bb\u03bf\u03c2.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u03b2\u03cc\u03bb\u03bf\u03c2.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\u0442(.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Parenthesis in SubDomain"},
	{urlParams: URLParams{URL: "http://\u04c0.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u04c0.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\u06dd.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u06dd.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\u09dc.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u09dc.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\u15ef\u15ef\u15ef.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u15ef\u15ef\u15ef.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\u180e.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u180e.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\u1e9e.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u1e9e.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\u2183.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u2183.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\u2665.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u2665.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\u4f60\u597d\u4f60\u597d.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\u4f60\u597d\u4f60\u597d.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\ufdd0zyx.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\ufdd0zyx.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\uff05\uff10\uff10.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\uff05\uff10\uff10.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\uff05\uff14\uff11.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\uff05\uff14\uff11.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\uff10\uff38\uff43\uff10\uff0e\uff10\uff12\uff15\uff10\uff0e\uff10\uff11.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\uff10\uff38\uff43\uff10\uff0e\uff10\uff12\uff15\uff10\uff0e\uff10\uff11.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://\uff27\uff4f.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "\uff27\uff4f.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://ab--cd.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "ab--cd.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Bad double-hyphen in SubDomain (still accepted)"},
	{urlParams: URLParams{URL: "http://fa\u00df.de.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "fa\u00df.de.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://foo-.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "foo-.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Bad SubDomain label end with dash (still accepted)"},
	{urlParams: URLParams{URL: "http://foo\u0300.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "foo\u0300.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://gOoGle.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "gOoGle.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Mixed case letters"},
	{urlParams: URLParams{URL: "http://hello%00.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "hello%00.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Percentage encoded SubDomain"},
	{urlParams: URLParams{URL: "http://look\u0341out.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "look\u0341out.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://look\u034fout.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "look\u034fout.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://look\u05beout.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "look\u05beout.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://look\u202eout.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "look\u202eout.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://look\u2060.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "look\u2060.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://look\u206bout.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "look\u206bout.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://look\u2ff0out.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "look\u2ff0out.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://look\ufffaout.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "look\ufffaout.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://uRLTest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "uRLTest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Mixed case letters"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Simple SubDomain+Domain"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/%20foo"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/%20foo", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/%3A%3a%3C%3c"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/%3A%3a%3C%3c", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/%7Ffp3%3Eju%3Dduvgw%3Dd"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/%7Ffp3%3Eju%3Dduvgw%3Dd", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/%A1%C1/?foo=%EF%BD%81"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/%A1%C1/?foo=%EF%BD%81", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/%A1%C1/?foo=???"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/%A1%C1/?foo=???", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/%EF%BD%81/?foo=%A1%C1"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/%EF%BD%81/?foo=%A1%C1", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/(%28:%3A%29)"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/(%28:%3A%29)", RegisteredDomain: "lookout.net"}, description: "Parentheses in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/././foo"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/././foo", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/./.foo"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/./.foo", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net////../.."}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "////../..", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/?%02hello%7f bye"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/?%02hello%7f bye", RegisteredDomain: "lookout.net"}, description: "Space in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/?%40%41123"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/?%40%41123", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/???/?foo=%A1%C1"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/???/?foo=%A1%C1", RegisteredDomain: "lookout.net"}, description: "Consecutive question marks"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/?D%C3%BCrst"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/?D%C3%BCrst", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/?D%FCrst"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/?D%FCrst", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/?as?df"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/?as?df", RegisteredDomain: "lookout.net"}, description: "Multiple question marks"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/?foo=bar"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/?foo=bar", RegisteredDomain: "lookout.net"}, description: "Path with Query Parameters"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/?q=&lt;asdf&gt;"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/?q=&lt;asdf&gt;", RegisteredDomain: "lookout.net"}, description: "Path with Query Parameters"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/?q=\"asdf\""}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/?q=\"asdf\"", RegisteredDomain: "lookout.net"}, description: "Path with inverted commas"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/?q=\u4f60\u597d"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/?q=\u4f60\u597d", RegisteredDomain: "lookout.net"}, description: "Unicode in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/@asdf%40"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/@asdf%40", RegisteredDomain: "lookout.net"}, description: "@ in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/D%C3%BCrst"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/D%C3%BCrst", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/D%FCrst"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/D%FCrst", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/\u2025/foo"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/\u2025/foo", RegisteredDomain: "lookout.net"}, description: "Unicode in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/\u202e/foo/\u202d/bar"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/\u202e/foo/\u202d/bar", RegisteredDomain: "lookout.net"}, description: "Unicode in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/\u4f60\u597d\u4f60\u597d"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/\u4f60\u597d\u4f60\u597d", RegisteredDomain: "lookout.net"}, description: "Unicode in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/\ufdd0zyx"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/\ufdd0zyx", RegisteredDomain: "lookout.net"}, description: "Unicode in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/\ufeff/foo"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/\ufeff/foo", RegisteredDomain: "lookout.net"}, description: "Unicode in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo", RegisteredDomain: "lookout.net"}, description: "Simple SubDomain+Domain+Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo    bar/?   foo   =   bar     #    foo"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo    bar/?   foo   =   bar     #    foo", RegisteredDomain: "lookout.net"}, description: "Space in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo%"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo%", RegisteredDomain: "lookout.net"}, description: "Trailing percentage sign in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo%00%51"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo%00%51", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo%2"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo%2", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo%2Ehtml"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo%2Ehtml", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo%2\u00c2\u00a9zbar"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo%2\u00c2\u00a9zbar", RegisteredDomain: "lookout.net"}, description: "Unicode in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo%2fbar"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo%2fbar", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo%2zbar"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo%2zbar", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo%3fbar"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo%3fbar", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo%41%7a"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo%41%7a", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/%2e"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/%2e", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/%2e%2"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/%2e%2", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/%2e./%2e%2e/.%2e/%2e.bar"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/%2e./%2e%2e/.%2e/%2e.bar", RegisteredDomain: "lookout.net"}, description: "Percentage encoded Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/."}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/.", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/../../.."}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/../../..", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/../../../ton"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/../../../ton", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/..bar"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/..bar", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/./"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/./", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/bar/.."}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/bar/..", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/bar/../"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/bar/../", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/bar/../ton"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/bar/../ton", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/bar/../ton/../../a"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/bar/../ton/../../a", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/bar//.."}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/bar//..", RegisteredDomain: "lookout.net"}, description: "Dots in Path, Multiple slashes"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo/bar//../.."}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo/bar//../..", RegisteredDomain: "lookout.net"}, description: "Dots in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo?bar=baz#"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo?bar=baz#", RegisteredDomain: "lookout.net"}, description: "Query Parameters in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo\\tbar"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo\\tbar", RegisteredDomain: "lookout.net"}, description: "Backslash in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net/foo\t\ufffd%91"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Path: "/foo\t\ufffd%91", RegisteredDomain: "lookout.net"}, description: "Tab in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net:80/"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", Port: "80", RegisteredDomain: "lookout.net", Path: "/"}, description: "Port"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net::80::443/"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[10], description: "Bad Port"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net::==80::==443::/"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[10], description: "Bad Port"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net\\\\foo\\\\bar"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net", Path: "\\\\foo\\\\bar"}, description: "Multiple backslashes in Path"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net\u2a7480/"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest.lookout", Domain: "net\u2a7480", Path: "/"}, description: "Unicode in Domain"},
	{urlParams: URLParams{URL: "http://urltest.lookout.net\uff0ffoo/"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "urltest.lookout", Domain: "net\uff0ffoo", Path: "/"}, description: "Unicode in Domain"},
	{urlParams: URLParams{URL: "http://www.foo\u3002bar.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "www.foo\u3002bar.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://www.loo\u0138out.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "www.loo\u0138out.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://www.lookout.\u0441\u043e\u043c.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "www.lookout.\u0441\u043e\u043c.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://www.lookout.net\uff1a80.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Reject full-width colon"},
	{urlParams: URLParams{URL: "http://www.lookout\u2027net.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://", SubDomain: "www.lookout\u2027net.urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Unicode in SubDomain"},
	{urlParams: URLParams{URL: "http://www\u2025urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errs[8], description: "Invalid Character"},
	{urlParams: URLParams{URL: "http://xn--0.urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http://"}, err: errors.New("idna: invalid label \"0\""), description: "Invalid Punycode"},
	{urlParams: URLParams{URL: "http:\\\\\\\\urltest.lookout.net\\\\foo"}, expected: &ExtractResult{Scheme: "http:\\\\\\\\", SubDomain: "urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net", Path: "\\\\foo"}, description: "Multiple forward slashes in Scheme"},
	{urlParams: URLParams{URL: "http:///\\/\\/\\/\\/urltest.lookout.net"}, expected: &ExtractResult{Scheme: "http:///\\/\\/\\/\\/", SubDomain: "urltest", Domain: "lookout", Suffix: "net", RegisteredDomain: "lookout.net"}, description: "Multiple mixed slashes in Scheme"},
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
		userInfoTests,
		ipv4Tests,
		ipv6Tests,
		ignoreSubDomainsTests,
		privateSuffixTests,
		periodsAndWhiteSpacesTests,
		invalidTests,
		internationalTLDTests,
		domainOnlySingleTLDTests,
		pathTests,
		wildcardTests,
		lookoutTests,
	} {
		for _, test := range testCollection {
			var extractor *FastTLD
			if test.includePrivateSuffix {
				extractor = extractorWithPrivateSuffix
			} else {
				extractor = extractorWithoutPrivateSuffix
			}
			res, err := extractor.Extract(test.urlParams)

			if output := reflect.DeepEqual(res,
				test.expected); !output {
				t.Errorf("Output %q not equal to expected output %q | %q",
					res, test.expected, test.description)
			}

			if !(err == nil && test.err == nil) &&
				((err == nil && test.err != nil) ||
					(err != nil && test.err == nil) ||
					!reflect.DeepEqual(err.Error(),
						test.err.Error())) {
				t.Errorf("Error %q not equal to expected error %q | %q",
					err, test.err, test.description)
			}
		}
	}
}
