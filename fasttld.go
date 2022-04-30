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

const periodDelimiters string = "\u002e\u3002\uff0e\uff61"

const periodDelimitersAndWhiteSpace string = periodDelimiters + " \n\t\r\uFEFF\u200b\u200c\u200d"

// Extract URL scheme from string
var schemeRegex = regexp.MustCompile("^([A-Za-z0-9+-.]+:)?//")

// For replacing international period delimiters when converting to punycode
var periodDelimitersRegex = regexp.MustCompile("[" + periodDelimiters + "]")

// FastTLD provides the Extract() function, to extract
// URLs using TldTrie generated from the
// Public Suffix List file at cacheFilePath
type FastTLD struct {
	TldTrie       *trie
	cacheFilePath string
}

// ExtractResult contains components extracted from URL
type ExtractResult struct {
	Scheme, UserInfo, SubDomain, Domain, Suffix, Port, Path, RegisteredDomain string
}

// SuffixListParams contains parameters for specifying path to Public Suffix List file and
// whether to extract private suffixes (e.g. blogspot.com).
type SuffixListParams struct {
	CacheFilePath        string
	IncludePrivateSuffix bool
}

// URLParams specifies URL to extract components from.
//
// If IgnoreSubDomains = true, do not extract subdomains.
//
// If ConvertURLToPunyCode = true, convert non-ASCII characters like 世界 to punycode.
type URLParams struct {
	URL                  string
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
	var dicBk *trie

	keysExceptLast := keys[0 : len(keys)-1]
	lenKeys := len(keys)

	for _, key := range keysExceptLast {
		dicBk = dic
		// if dic.matches[key] does not exist
		if _, ok := dic.matches[key]; !ok {
			// set dic.matches[key] to &Trie
			dic.matches[key] = &trie{hasChildren: true, matches: make(map[string]*trie)}
		}
		dic = dic.matches[key] // point dic to it
		if len(dic.matches) == 0 && !dic.hasChildren {
			end = true
			dic = dicBk
			dic.matches[keys[lenKeys-2]] = &trie{end: true, matches: make(map[string]*trie)}
			dic.matches[keys[lenKeys-2]].matches[keys[lenKeys-1]] = &trie{matches: make(map[string]*trie)}
		}
	}
	if !end {
		dic.matches[keys[lenKeys-1]] = &trie{matches: make(map[string]*trie)}
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
	asPunyCode, err := idna.ToASCII(periodDelimitersRegex.ReplaceAllLiteralString(strings.ToLower(s), "."))
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

func sepSize(r byte) int {
	// r is the first byte of any of the runes in periodDelimiters
	if r == 46 {
		// first byte of '.' is 46
		// size of '.' is 1
		return 1
	}
	// first byte of any period delimiter other than '.' is not 46
	// size of delimiter is 3
	return 3
}

// Extract components from a given `url`.
func (f *FastTLD) Extract(e URLParams) *ExtractResult {
	urlParts := ExtractResult{}

	// Extract URL scheme
	netlocWithScheme := strings.Trim(e.URL, periodDelimitersAndWhiteSpace)
	if e.ConvertURLToPunyCode {
		netlocWithScheme = formatAsPunycode(netlocWithScheme)
	}
	netloc := schemeRegex.ReplaceAllLiteralString(netlocWithScheme, "")

	urlParts.Scheme = netlocWithScheme[0 : len(netlocWithScheme)-len(netloc)]

	var afterHost string

	// Extract URL userinfo
	if atIdx := strings.IndexRune(netloc, '@'); atIdx != -1 {
		urlParts.UserInfo = netloc[0:atIdx]
		netloc = netloc[atIdx+1:]
	}

	// Separate URL host from subcomponents thereafter
	if hostEndIndex := strings.IndexAny(netloc, "/:?&#"); hostEndIndex != -1 {
		afterHost = netloc[hostEndIndex:]
		netloc = netloc[0:hostEndIndex]
	}

	// extract port and "Path" if any
	if len(afterHost) != 0 {
		pathStartIndex := strings.IndexRune(afterHost, '/')
		var (
			maybePort   string
			invalidPort bool
		)
		if afterHost[0] == ':' {
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
		if !invalidPort && pathStartIndex != -1 && pathStartIndex != len(afterHost) {
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

	// define the root node
	node := f.TldTrie

	var hasSuffix bool
	var end bool
	sepIdx := len(netloc)
	previousSepIdx := sepIdx

	for !end {
		var label string
		previousSepIdx = sepIdx
		sepIdx = strings.LastIndexAny(netloc[0:sepIdx], periodDelimiters)
		if sepIdx != -1 {
			label = netloc[sepIdx+sepSize(netloc[sepIdx]) : previousSepIdx]
		} else {
			label = netloc[0:previousSepIdx]
			end = true
		}

		// this node has sub-nodes and maybe an end-node.
		// eg. cn -> (cn, gov.cn)
		if node.end {
			// check if there is a sub node
			// eg. gov.cn
			if val, ok := node.matches[label]; ok {
				hasSuffix = true
				if len(val.matches) == 0 {
					break
				}
				node = val
				continue
			}
		}

		if _, ok := node.matches["*"]; ok {
			// check if there is a sub node
			// e.g. www.ck
			if _, ok := node.matches["!"+label]; !ok {
				hasSuffix = true
			}
			break
		}
		// check if TLD in Public Suffix List
		if val, ok := node.matches[label]; ok {
			hasSuffix = true
			if len(val.matches) != 0 {
				node = val
			} else {
				break
			}
		} else {
			if previousSepIdx != len(netloc) {
				sepIdx = previousSepIdx
			}
			break
		}
	}

	if hasSuffix {
		if sepIdx != -1 {
			// if there is a domain
			urlParts.Suffix = netloc[sepIdx+sepSize(netloc[sepIdx]):]
			domainStartSepIdx := strings.LastIndexAny(netloc[0:sepIdx], periodDelimiters)
			if domainStartSepIdx != -1 {
				// if subdomains exist
				domainStartIdx := domainStartSepIdx + sepSize(netloc[domainStartSepIdx])
				urlParts.Domain = netloc[domainStartIdx:sepIdx]
				urlParts.RegisteredDomain = netloc[domainStartIdx:]
				if !e.IgnoreSubDomains {
					// if subdomains are to be included
					urlParts.SubDomain = netloc[0:domainStartSepIdx]
				}
			} else {
				urlParts.Domain = netloc[domainStartSepIdx+1 : sepIdx]
				urlParts.RegisteredDomain = netloc[domainStartSepIdx+1:]
			}
		} else {
			// if only suffix exists
			urlParts.Suffix = netloc
		}
	}

	return &urlParts
}

// Number of hours elapsed since last modified time of fileinfo.
func fileLastModifiedHours(fileinfo fs.FileInfo) float64 {
	return time.Now().Sub(fileinfo.ModTime()).Hours()
}

// New creates a new *FastTLD.
func New(n SuffixListParams) (*FastTLD, error) {
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

	return &FastTLD{tldTrie, n.CacheFilePath}, err
}
