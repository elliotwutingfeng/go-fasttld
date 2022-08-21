// Package fasttld is a high performance top level domains (TLD)
// extraction module implemented with compressed tries.
//
// This module is a port of the Python fasttld module,
// with additional modifications to support extraction
// of subcomponents from full URLs, IPv4 addresses, and IPv6 addresses.
package fasttld

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
)

const defaultPSLFileName string = "public_suffix_list.dat"
const largestPortNumber int = 65535
const pslMaxAgeHours float64 = 72

// FastTLD provides the Extract() function, to extract
// URLs using tldTrie generated from the
// Public Suffix List file at cacheFilePath.
type FastTLD struct {
	cacheFilePath string
	tldTrie       *trie
}

// HostType indicates whether parsed URL
// contains a HostName, IPv4 address, IPv6 address
// or none of them
type HostType int

// None, HostName, IPv4 and IPv6 indicate whether parsed URL
// contains a HostName, IPv4 address, IPv6 address
// or none of them
const (
	None HostType = iota
	HostName
	IPv4
	IPv6
)

// ExtractResult contains components extracted from URL.
type ExtractResult struct {
	Scheme, UserInfo, SubDomain, Domain, Suffix, RegisteredDomain, Port, Path string
	HostType                                                                  HostType
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
	matches     map[string]*trie
	end         bool
	hasChildren bool
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
		if _, ok := dic.matches[key]; !ok {
			dic.matches[key] = &trie{hasChildren: true, matches: make(map[string]*trie)}
		}
		dic = dic.matches[key]
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
	if includePrivateSuffix {
		suffixList = suffixLists.allSuffixes
	} else {
		suffixList = suffixLists.publicSuffixes
	}

	for _, suffix := range suffixList {
		if strings.ContainsRune(suffix, '.') {
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
func (f *FastTLD) Extract(e URLParams) (ExtractResult, error) {
	urlParts := ExtractResult{}

	// Extract URL scheme
	netloc := fastTrim(e.URL, whitespaceRuneSet, trimBoth)
	if schemeEndIndex := getSchemeEndIndex(netloc); schemeEndIndex != -1 {
		urlParts.Scheme = netloc[0:schemeEndIndex]
		netloc = netloc[schemeEndIndex:]
	}

	// Extract URL userinfo
	if atIdx := indexLastByteBefore(netloc, '@', invalidUserInfoCharsSet); atIdx != -1 {
		urlParts.UserInfo = netloc[0:atIdx]
		netloc = netloc[atIdx+1:]
	}

	// Find square brackets (if any) and host end index
	openingSquareBracketIdx := -1
	closingSquareBracketIdx := -1
	hostEndIdx := -1

	for i, r := range []byte(netloc) {
		if r == '[' {
			// Check for opening square bracket
			if i > 0 {
				// Reject if opening square bracket is not first character of hostname
				return urlParts, errors.New("opening square bracket is not first character of hostname")
			}
			openingSquareBracketIdx = i
		}
		if r == ']' {
			// Check for closing square bracket
			closingSquareBracketIdx = i
		}

		if openingSquareBracketIdx == -1 {
			if closingSquareBracketIdx != -1 {
				// Reject if closing square bracket present but no opening square bracket
				return urlParts, errors.New("closing square bracket present but no opening square bracket")
			}
			if endOfHostDelimitersSet.contains(r) {
				// If no square brackets
				// Check for endOfHostDelimitersSet
				hostEndIdx = i
				break
			}
		} else if closingSquareBracketIdx > openingSquareBracketIdx && endOfHostWithPortDelimitersSet.contains(r) {
			// If opening + closing square bracket are present in correct order
			// check for endOfHostWithPortDelimitersSet
			hostEndIdx = i
			break
		}

		if i == len(netloc)-1 && closingSquareBracketIdx < openingSquareBracketIdx {
			// Reject if end of netloc reached but incomplete square bracket pair
			return urlParts, errors.New("incomplete square bracket pair")
		}
	}

	if closingSquareBracketIdx == len(netloc)-1 {
		hostEndIdx = -1
	} else if closingSquareBracketIdx != -1 {
		hostEndIdx = closingSquareBracketIdx + 1
	}

	// Check for IPv6 address
	if closingSquareBracketIdx > openingSquareBracketIdx {
		if !isIPv6(netloc[1:closingSquareBracketIdx]) {
			// Have square brackets but invalid IPv6 address => Domain is invalid
			return urlParts, errors.New("invalid IPv6 address")
		}
		if hostEndIdx != -1 {
			afterHost := netloc[hostEndIdx:]
			if indexAnyASCII(afterHost, endOfHostDelimitersSet) != 0 {
				// Reject IPv6 if there are invalid trailing characters after IPv6 address
				return urlParts, errors.New("invalid trailing characters after IPv6 address")
			}
		}
		// Closing square bracket in correct place and IPv6 is valid
		urlParts.HostType = IPv6
		urlParts.Domain = netloc[1:closingSquareBracketIdx]
		urlParts.RegisteredDomain = netloc[1:closingSquareBracketIdx]
	}

	var afterHost string
	// Separate URL host from subcomponents thereafter
	if hostEndIdx != -1 {
		afterHost = netloc[hostEndIdx:]
		netloc = netloc[0:hostEndIdx]
	}

	// Extract Port and "Path" if any
	if len(afterHost) != 0 {
		pathStartIndex := indexAnyASCII(afterHost, endOfHostWithPortDelimitersSet)
		if afterHost[0] == ':' {
			var maybePort string
			if pathStartIndex == -1 {
				maybePort = afterHost[1:]
			} else {
				maybePort = afterHost[1:pathStartIndex]
			}
			if port, err := strconv.Atoi(maybePort); err == nil && 0 <= port && port <= largestPortNumber {
				urlParts.Port = maybePort
			} else {
				return urlParts, errors.New("invalid port")
			}
		}
		if pathStartIndex != -1 && pathStartIndex != len(afterHost) {
			// If there is any path/query/fragment after the URL authority component...
			// See https://stackoverflow.com/questions/47543432/what-do-we-call-the-combined-path-query-and-fragment-in-a-uri
			// For simplicity, we shall call this the "Path".
			urlParts.Path = afterHost[pathStartIndex:]
		}
	}

	if urlParts.HostType == IPv6 {
		return urlParts, nil
	}

	if e.ConvertURLToPunyCode {
		netloc = formatAsPunycode(netloc)
	} else if _, err := idna.ToUnicode(netloc); err != nil {
		// host is invalid if host cannot be converted to Unicode
		//
		// skip if host already converted to punycode
		log.Println(strings.SplitAfterN(err.Error(), "idna: invalid label", 2)[0])
		return urlParts, err
	}

	// Check for TLD Suffix
	node := f.tldTrie

	var (
		hasSuffix      bool
		end            bool
		previousSepIdx int
	)
	sepIdx, suffixStartIdx, suffixEndIdx := len(netloc), len(netloc), len(netloc)

	for !end {
		var label string
		previousSepIdx = sepIdx
		sepIdx = lastIndexAny(netloc[0:sepIdx], labelSeparatorsRuneSet)
		if sepIdx != -1 {
			label = netloc[sepIdx+sepSize(netloc[sepIdx]) : previousSepIdx]
			if len(label) == 0 {
				// allow consecutive label separators if suffix not found yet
				if !hasSuffix {
					suffixEndIdx = previousSepIdx
					continue
				}
				// any occurrences of consecutive label separators on left-hand side of
				// partial or full suffix are illegal.
				return urlParts, errors.New("invalid consecutive label separators on left-hand side of partial or full suffix")
			}
		} else {
			label = netloc[0:previousSepIdx]
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
			suffixStartIdx = sepIdx
			if !hasSuffix {
				// index of end of suffix without trailing label separators
				suffixEndIdx = previousSepIdx
				hasSuffix = true
			}
			node = val
			if len(val.matches) == 0 {
				// label is at a leaf node (no children) ; break out of loop
				break
			}
		} else {
			if previousSepIdx != len(netloc) {
				sepIdx = previousSepIdx
			}
			break
		}
	}

	// Check for IPv4 address
	// Ensure first rune is numeric before expensive isIPv4()
	if len(netloc) != 0 && numericSet.contains(netloc[0]) && isIPv4(netloc) {
		urlParts.HostType = IPv4
		urlParts.Domain = netloc[0:previousSepIdx]
		urlParts.RegisteredDomain = urlParts.Domain
		return urlParts, nil
	}

	if sepIdx == -1 {
		sepIdx, suffixStartIdx = len(netloc), len(netloc)
	}

	// Reject if invalidHostNameChars or consecutive label separators
	// appears before Suffix
	if hasSuffix {
		if hasInvalidChars(netloc[0:suffixStartIdx]) {
			return urlParts, errors.New("invalid characters in hostname")
		}
	} else {
		if hasInvalidChars(netloc[0:previousSepIdx]) {
			return urlParts, errors.New("invalid characters in hostname")
		}
	}

	var domainStartSepIdx int
	if hasSuffix {
		if sepIdx < len(netloc) { // If there is a Domain
			urlParts.Suffix = netloc[sepIdx+sepSize(netloc[sepIdx]) : suffixEndIdx]
			domainStartSepIdx = lastIndexAny(netloc[0:sepIdx], labelSeparatorsRuneSet)
			if domainStartSepIdx != -1 { // If there is a SubDomain
				domainStartIdx := domainStartSepIdx + sepSize(netloc[domainStartSepIdx])
				urlParts.Domain = netloc[domainStartIdx:sepIdx]
				urlParts.RegisteredDomain = netloc[domainStartIdx:suffixEndIdx]
			} else {
				urlParts.Domain = netloc[0:sepIdx]
				urlParts.RegisteredDomain = netloc[0:suffixEndIdx]
			}
		} else {
			// Only Suffix exists
			urlParts.Suffix = netloc[0:suffixEndIdx]
		}
	} else {
		domainStartSepIdx = lastIndexAny(netloc[0:previousSepIdx], labelSeparatorsRuneSet)
		var domainStartIdx int
		if domainStartSepIdx != -1 { // If there is a SubDomain
			domainStartIdx = domainStartSepIdx + sepSize(netloc[domainStartSepIdx])
		}
		urlParts.Domain = netloc[domainStartIdx:previousSepIdx]
	}
	if !e.IgnoreSubDomains && domainStartSepIdx != -1 { // If SubDomain is to be included
		urlParts.SubDomain = netloc[0:domainStartSepIdx]
	}

	if len(urlParts.Domain) == 0 {
		return urlParts, errors.New("empty domain")
	}
	urlParts.HostType = HostName
	return urlParts, nil
}

// New creates a new *FastTLD using data from a Public Suffix List file.
func New(n SuffixListParams) (*FastTLD, error) {
	cacheFilePath, err := filepath.Abs(n.CacheFilePath)
	cacheFilePathIsInvalid := err != nil

	// If cacheFilePath is unreachable, use default Public Suffix List file
	if stat, err := os.Stat(strings.TrimSpace(cacheFilePath)); cacheFilePathIsInvalid || err != nil || stat.IsDir() || stat.Size() == 0 {
		n.CacheFilePath = getCurrentFilePath() + string(os.PathSeparator) + defaultPSLFileName
		// Update Public Suffix List file if it doesn't exist or is older than pslMaxAgeHours. Proceed with existing file if update fails.
		if fileinfo, err := os.Stat(n.CacheFilePath); err != nil || fileLastModifiedHours(fileinfo) > pslMaxAgeHours {
			if file, err := os.Create(n.CacheFilePath); err == nil {
				if err := update(file, publicSuffixListSources); err != nil {
					log.Println(err)
				}
				defer file.Close()
			}
		}
	}

	// Construct *trie using Public Suffix List file located at n.CacheFilePath
	tldTrie, err := trieConstruct(n.IncludePrivateSuffix, n.CacheFilePath)

	return &FastTLD{cacheFilePath: n.CacheFilePath, tldTrie: tldTrie}, err
}
