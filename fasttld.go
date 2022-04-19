// go-fasttld is a high performance top level domains (TLD)
// extraction module implemented with compressed tries.
//
// This module is a port of the Python fasttld module,
// with additional modifications to support extraction
// of subcomponents from full URLs and IPv4 addresses.
package fasttld

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

const defaultPSLFileName string = "public_suffix_list.dat"

// Extract URL scheme from string
var schemeRegex = regexp.MustCompile("^([A-Za-z0-9+-.]+:)?//")

type fastTLD struct {
	TldTrie       *trie
	cacheFilePath string
}

// Components extracted from URL
type ExtractResult struct {
	Scheme, UserInfo, SubDomain, Domain, Suffix, Port, Path, RegisteredDomain string
}

// Parameters for specifying path to Public Suffix List file and
// whether to extract private suffixes (e.g. blogspot.com).
type SuffixListParams struct {
	CacheFilePath        string
	IncludePrivateSuffix bool
}

// Specify Url to extract components from.
//
// If IgnoreSubDomains = true, do not extract subdomains.
//
// If ConvertURLToPunyCode = true, convert non-ASCII characters like 世界 to punycode.
type UrlParams struct {
	Url                  string
	IgnoreSubDomains     bool
	ConvertURLToPunyCode bool
}

type trie struct {
	end         bool
	hasChildren bool
	matches     map[string]*trie
}

// Store a slice of keys in the trie, by traversing the trie using the keys as a "path",
// creating new tries for keys that do not exist yet.
//
// If a new path overlaps an existing path, flag the previous path's trie node as End = true
func nestedDict(dic *trie, keys []string) {
	// credits: https://stackoverflow.com/questions/13687924 and https://github.com/jophy/fasttld
	var end bool
	var dic_bk *trie

	keys_ := keys[0 : len(keys)-1]
	len_keys := len(keys)

	for _, key := range keys_ {
		dic_bk = dic
		// if dic.matches[key] does not exist
		if _, ok := dic.matches[key]; !ok {
			// set dic.matches[key] to &Trie
			dic.matches[key] = &trie{hasChildren: true, matches: make(map[string]*trie)}
		}
		dic = dic.matches[key] // point dic to it
		if len(dic.matches) == 0 && !dic.hasChildren {
			end = true
			dic = dic_bk
			dic.matches[keys[len_keys-2]] = &trie{end: true, matches: make(map[string]*trie)}
			dic.matches[keys[len_keys-2]].matches[keys[len_keys-1]] = &trie{matches: make(map[string]*trie)}
		}
	}
	if !end {
		dic.matches[keys[len_keys-1]] = &trie{matches: make(map[string]*trie)}
	}
}

// Reverse a slice of strings in-place.
func reverse(input []string) {
	for i, j := 0, len(input)-1; i < j; i, j = i+1, j-1 {
		input[i], input[j] = input[j], input[i]
	}
}

// Format string as punycode.
func formatAsPunycode(s string) string {
	asPunyCode, err := idna.ToASCII(strings.ToLower(strings.TrimSpace(s)))
	if err != nil {
		log.Println(strings.SplitAfterN(err.Error(), "idna: invalid label", 2)[0])
		return ""
	}

	return asPunyCode
}

// Construct a compressed trie to store Public Suffix List TLDs split at "." in reverse-order.
//
// For example: "us.gov.pl" will be stored in the order {"pl", "gov", "us"}
func trieConstruct(includePrivateSuffix bool, cacheFilePath string) (*trie, error) {
	tldTrie := &trie{matches: make(map[string]*trie)}
	suffixLists, err := getPublicSuffixList(cacheFilePath)
	if err != nil {
		log.Println(err)
		return tldTrie, err
	}

	var suffixList []string
	if !includePrivateSuffix {
		// public suffixes only
		suffixList = suffixLists[0]
	} else {
		// public suffixes AND private suffixes
		suffixList = suffixLists[2]
	}

	for _, suffix := range suffixList {
		if strings.Contains(suffix, ".") {
			sp := strings.Split(suffix, ".")
			reverse(sp)
			nestedDict(tldTrie, sp)
		} else {
			tldTrie.matches[suffix] = &trie{end: true, matches: make(map[string]*trie)}
		}
	}

	for key := range tldTrie.matches {
		if len(tldTrie.matches[key].matches) == 0 && tldTrie.matches[key].end {
			tldTrie.matches[key] = &trie{matches: make(map[string]*trie)}
		}
	}

	return tldTrie, nil
}

// Extract components from a given `url`.
func (f *fastTLD) Extract(e UrlParams) *ExtractResult {
	urlParts := ExtractResult{}

	if e.ConvertURLToPunyCode {
		e.Url = formatAsPunycode(e.Url)
	}

	// Extract URL scheme
	// Credits: https://github.com/mjd2021usa/tldextract/blob/main/tldextract.go
	netlocWithScheme := strings.Trim(e.Url, ". \n\t\r\uFEFF\u200b\u200c\u200d") // trim whitespace and '.'
	netloc := schemeRegex.ReplaceAllString(netlocWithScheme, "")

	urlParts.Scheme = netlocWithScheme[0 : len(netlocWithScheme)-len(netloc)]

	var afterHost string

	// Extract URL userinfo
	if atIdx := strings.Index(netloc, "@"); atIdx != -1 {
		urlParts.UserInfo = netloc[:atIdx]
		netloc = netloc[atIdx+1:]
	}

	// Separate URL host from subcomponents thereafter
	hostEndIndex := strings.IndexFunc(netloc, func(r rune) bool {
		switch r {
		case ':', '/', '?', '&', '#':
			return true
		}
		return false
	})
	if hostEndIndex != -1 {
		afterHost = netloc[hostEndIndex:]
		netloc = netloc[0:hostEndIndex]
	}

	// extract port and "Path" if any
	if lenAfterHost := len(afterHost); lenAfterHost != 0 {
		pathStartIndex := strings.Index(afterHost, "/")
		var (
			maybePort   string
			invalidPort bool
		)
		if hasPort := afterHost[0] == ':'; hasPort {
			if pathStartIndex == -1 {
				maybePort = afterHost[1:]
			} else {
				maybePort = afterHost[1:pathStartIndex]
			}
			if port, err := strconv.Atoi(maybePort); !(err == nil && 0 <= port && port <= 65535) {
				invalidPort = true
			} else {
				urlParts.Port = maybePort
			}
		}
		if !invalidPort && pathStartIndex != -1 && pathStartIndex != lenAfterHost {
			// if there is any path/query/fragment after the authority URI component...
			// see https://stackoverflow.com/questions/47543432/what-do-we-call-the-combined-path-query-and-fragment-in-a-uri
			// for simplicity, we shall call this the "Path"
			urlParts.Path = afterHost[pathStartIndex+1:]
		}

	}

	if looksLikeIPv4Address(netloc) {
		urlParts.Domain = netloc
		urlParts.RegisteredDomain = netloc
		return &urlParts
	}

	labels := strings.Split(netloc, ".")

	var node *trie
	// define the root node
	node = f.TldTrie

	var lenSuffix int
	var suffixCharCount int
	lenLabels := len(labels)
	for idx := range labels {
		label := labels[lenLabels-idx-1]
		labelLength := len(label)

		// this node has sub-nodes and maybe an end-node.
		// eg. cn -> (cn, gov.cn)
		if node.end {
			// check if there is a sub node
			// eg. gov.cn
			if val, ok := node.matches[label]; ok {
				lenSuffix += 1
				suffixCharCount += labelLength
				if len(val.matches) == 0 {
					urlParts.Domain = labels[idx-1]
					break
				} else {
					node = val
					continue
				}
			}
		}

		if _, ok := node.matches["*"]; ok {
			// check if there is a sub node
			// e.g. www.ck
			if _, ok := node.matches["!"+label]; !ok {
				lenSuffix += 1
				suffixCharCount += labelLength
			} else {
				urlParts.Domain = label
			}
			break
		}
		// check if TLD in Public Suffix List
		if val, ok := node.matches[label]; ok {
			lenSuffix += 1
			suffixCharCount += labelLength
			if len(val.matches) != 0 {
				node = val
			} else {
				break
			}
		} else {
			break
		}
	}

	netlocLen := len(netloc)
	if lenSuffix != 0 {
		urlParts.Suffix = netloc[netlocLen-suffixCharCount-lenSuffix+1:]
	}

	len_url_suffix := len(urlParts.Suffix)
	var len_url_domain int

	if 0 < lenSuffix && lenSuffix < lenLabels {
		urlParts.Domain = labels[lenLabels-lenSuffix-1]
		len_url_domain = len(urlParts.Domain)
		if !e.IgnoreSubDomains && (lenLabels-lenSuffix) >= 2 {
			urlParts.SubDomain = netloc[:netlocLen-len_url_domain-len_url_suffix-2]
		}
	}

	if len_url_domain > 0 && len_url_suffix > 0 {
		urlParts.RegisteredDomain = urlParts.Domain + "." + urlParts.Suffix
	}

	return &urlParts
}

// Number of hours elapsed since last modified time of fileinfo.
func fileLastModifiedHours(fileinfo fs.FileInfo) float64 {
	return time.Now().Sub(fileinfo.ModTime()).Hours()
}

// New creates a new *FastTLD.
func New(n SuffixListParams) (*fastTLD, error) {
	cacheFilePath, err := filepath.Abs(n.CacheFilePath)
	invalidCacheFilePath := err != nil

	// if cacheFilePath is unreachable, use default Public Suffix List
	if stat, err := os.Stat(strings.TrimSpace(cacheFilePath)); invalidCacheFilePath || err != nil || stat.IsDir() || stat.Size() == 0 {
		n.CacheFilePath = getCurrentFilePath() + string(os.PathSeparator) + defaultPSLFileName
		// Update Public Suffix List if it doesn't exist or is more than 3 days old
		if fileinfo, err := os.Stat(n.CacheFilePath); err != nil || fileLastModifiedHours(fileinfo) > 72 {
			// Create local file at n.CacheFilePath
			if file, err := os.Create(n.CacheFilePath); err == nil {
				err = update(file, publicSuffixListSources)
				defer file.Close()
			}
		}
	}

	// Construct *trie using list located at n.CacheFilePath
	tldTrie, err := trieConstruct(n.IncludePrivateSuffix, n.CacheFilePath)

	return &fastTLD{tldTrie, n.CacheFilePath}, err
}
