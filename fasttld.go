// go-fasttld is a high performance top level domains (TLD)
// extraction module implemented with compressed tries.
//
// This module is a port of the Python fasttld module,
// with additional modifications to support extraction
// of subcomponents from full URLs and IPv4 addresses.
package fasttld

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
)

const defaultPSLFileName string = "public_suffix_list.dat"

// FastTLD provides the Extract() function, to extract
// URLs using TldTrie generated from the
// Public Suffix List file at cacheFilePath.
type FastTLD struct {
	TldTrie       *trie
	cacheFilePath string
}

// ExtractResult contains components extracted from URL.
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
// If IgnoreSubDomains = true, do not extract SubDomain.
//
// If ConvertURLToPunyCode = true, convert non-ASCII characters like 世界 to punycode.
type URLParams struct {
	URL                  string
	IgnoreSubDomains     bool
	ConvertURLToPunyCode bool
}

// trie is a node of the compressed trie
// used to store Public Suffix List TLDs.
type trie struct {
	end         bool
	hasChildren bool
	matches     map[string]*trie
}

// nestedDict stores a slice of keys in the trie, by traversing the trie using the keys as a "path",
// creating new tries for keys that do not exist yet.
//
// If a new path overlaps an existing path, flag the previous path's trie node as end = true.
func nestedDict(dic *trie, keys []string) {
	// credits: https://stackoverflow.com/questions/13687924 and https://github.com/jophy/fasttld
	var end bool
	var dicBk *trie

	keysExceptLast := keys[0 : len(keys)-1]
	lenKeys := len(keys)

	for _, key := range keysExceptLast {
		dicBk = dic
		// If dic.matches[key] does not exist
		if _, ok := dic.matches[key]; !ok {
			// Set dic.matches[key] to &Trie
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

// trieConstruct constructs a compressed trie to store Public Suffix List TLDs split at "." in reverse-order.
//
// For example: "us.gov.pl" will be stored in the order {"pl", "gov", "us"}.
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
func (f *FastTLD) Extract(e URLParams) *ExtractResult {
	urlParts := ExtractResult{}

	// Extract URL scheme
	netlocWithScheme := strings.Trim(e.URL, labelSeparatorsAndWhiteSpace)
	netloc := schemeRegex.ReplaceAllLiteralString(netlocWithScheme, "")
	urlParts.Scheme = netlocWithScheme[0 : len(netlocWithScheme)-len(netloc)]

	// Extract URL userinfo
	if atIdx := strings.IndexByte(netloc, '@'); atIdx != -1 {
		urlParts.UserInfo = netloc[0:atIdx]
		netloc = netloc[atIdx+1:]
	}

	// Check for IPv6 address
	var isIPv6 bool
	closingSquareBracketIdx := strings.IndexByte(netloc, ']')
	if len(netloc) != 0 {
		if netloc[0] == '[' {
			if closingSquareBracketIdx > 0 && looksLikeIPAddress(netloc[1:closingSquareBracketIdx]) {
				urlParts.Domain = netloc[1:closingSquareBracketIdx]
				urlParts.RegisteredDomain = netloc[1:closingSquareBracketIdx]
				isIPv6 = true
			} else {
				// Have opening square bracket but invalid IPv6 => Domain is invalid
				return &urlParts
			}
		} else if closingSquareBracketIdx != -1 {
			// netloc not empty and there is an erroneous closing square bracket
			return &urlParts
		}
	}

	var afterHost string
	hostEndIndex := -1
	// Separate URL host from subcomponents thereafter
	if isIPv6 {
		hostEndIndex = len(netloc[0:closingSquareBracketIdx]) + indexAny(netloc[closingSquareBracketIdx:], endOfHostDelimitersSet)
	} else {
		hostEndIndex = indexAny(netloc, endOfHostDelimitersSet)
	}
	if hostEndIndex != -1 {
		afterHost = netloc[hostEndIndex:]
		netloc = netloc[0:hostEndIndex]
	}

	var host string
	// Check if host cannot be converted to unicode
	if _, err := idna.ToUnicode(netloc); err != nil {
		log.Println(strings.SplitAfterN(err.Error(), "idna: invalid label", 2)[0])
		return &urlParts
	}

	if e.ConvertURLToPunyCode {
		host = formatAsPunycode(standardLabelSeparatorReplacer.Replace(netloc))
	} else {
		host = netloc
	}

	// Extract Port and "Path" if any
	if len(afterHost) != 0 {
		pathStartIndex := strings.IndexByte(afterHost, '/')
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
			// If there is any path/query/fragment after the URL authority component...
			// See https://stackoverflow.com/questions/47543432/what-do-we-call-the-combined-path-query-and-fragment-in-a-uri
			// For simplicity, we shall call this the "Path".
			urlParts.Path = afterHost[pathStartIndex+1:]
		}
	}

	if isIPv6 {
		return &urlParts
	}

	// Check for IPv4 address
	if looksLikeIPAddress(host) {
		urlParts.Domain = host
		urlParts.RegisteredDomain = host
		return &urlParts
	}

	// Define the root node
	node := f.TldTrie

	var hasSuffix bool
	var end bool
	sepIdx := len(host)
	previousSepIdx := sepIdx

	for !end {
		var label string
		previousSepIdx = sepIdx
		sepIdx = lastIndexAny(host[0:sepIdx], labelSeparators)
		if sepIdx != -1 {
			label = host[sepIdx+sepSize(host[sepIdx]) : previousSepIdx]
		} else {
			label = host[0:previousSepIdx]
			end = true
		}

		if _, ok := node.matches["*"]; ok {
			// check if label falls under any wildcard exception rule
			// e.g. !www.ck
			if _, ok := node.matches["!"+label]; ok {
				sepIdx = previousSepIdx
			}
			break
		}

		// check if label is part of a TLD
		if val, ok := node.matches[label]; ok {
			hasSuffix = true
			node = val
			if len(val.matches) == 0 {
				// label is at a leaf node (no children) ; break out of loop
				break
			}
		} else {
			if previousSepIdx != len(host) {
				sepIdx = previousSepIdx
			}
			break
		}
	}

	if hasSuffix {
		if sepIdx != -1 { // if there is a Domain
			urlParts.Suffix = host[sepIdx+sepSize(host[sepIdx]):]
			domainStartSepIdx := lastIndexAny(host[0:sepIdx], labelSeparators)
			if domainStartSepIdx != -1 { // if there is a SubDomain
				domainStartIdx := domainStartSepIdx + sepSize(host[domainStartSepIdx])
				urlParts.Domain = host[domainStartIdx:sepIdx]
				urlParts.RegisteredDomain = host[domainStartIdx:]
				if !e.IgnoreSubDomains { // if SubDomain is to be included
					urlParts.SubDomain = host[0:domainStartSepIdx]
				}
			} else {
				urlParts.Domain = host[domainStartSepIdx+1 : sepIdx]
				urlParts.RegisteredDomain = host[domainStartSepIdx+1:]
			}
		} else {
			// if only Suffix exists
			urlParts.Suffix = host
		}
	} else if sepIdx != -1 { // if there is a SubDomain
		domainStartSepIdx := lastIndexAny(host, labelSeparators)
		domainStartIdx := domainStartSepIdx + sepSize(host[domainStartSepIdx])
		urlParts.Domain = host[domainStartIdx:]
		if !e.IgnoreSubDomains { // if SubDomain is to be included
			urlParts.SubDomain = host[0:domainStartSepIdx]
		}
	} else { // if there is no SubDomain
		urlParts.Domain = host
	}

	return &urlParts
}

// New creates a new *FastTLD.
func New(n SuffixListParams) (*FastTLD, error) {
	cacheFilePath, err := filepath.Abs(n.CacheFilePath)
	invalidCacheFilePath := err != nil

	// If cacheFilePath is unreachable, use default Public Suffix List
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
