package fasttld

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
)

const defaultPSLFileName string = "public_suffix_list.dat"

// Extract URL scheme from string
var schemeRegex = regexp.MustCompile("^([A-Za-z0-9+-.]+:)?//")

// Hashmap with keys as strings
type Dict map[string]interface{}

type FastTLD struct {
	TldTrie       Dict
	cacheFilePath string
	TldTrie2      *Trie
}

type ExtractResult struct {
	SubDomain, Domain, Suffix, Port, RegisteredDomain string
}

type SuffixListParams struct {
	CacheFilePath        string
	IncludePrivateSuffix bool
}

type UrlParams struct {
	Url                  string
	IgnoreSubDomains     bool
	ConvertURLToPunyCode bool
}

type Trie struct {
	End         bool
	HasChildren bool
	Matches     map[string]*Trie
}

// credits: https://stackoverflow.com/questions/13687924 and https://github.com/jophy/fasttld
func NestedDict(dic Dict, keys []string) {
	if len(keys) == 0 {
	} else if len(keys) == 1 {
		dic[keys[0]] = true
	} else {
		keys_ := keys[0 : len(keys)-1]
		len_keys := len(keys)

		var dic_ interface{}

		dic_ = dic

		for _, key := range keys_ {
			if dic_bk, isDict := dic_.(Dict); isDict {
				if _, ok := dic_bk[key]; !ok { // dic_[key] does not exist
					dic_bk[key] = Dict{} // set to dict{}
					dic_ = dic_bk[key]   // point dic_ to it
				} else { // dic_[key] exists
					if _, isDict := dic_bk[key].(Dict); isDict {
						dic_ = dic_bk[key] // point dic_ to it
					} else {
						// it's a boolean
						dic_ = dic_bk
						dic_bk[keys[len_keys-2]] = Dict{"_END": true, keys[len_keys-1]: true}
					}
				}
			}
		}
		if val, isDict := dic_.(Dict); isDict {
			val[keys[len_keys-1]] = true
		}
	}
}

func NestedDict2(dic *Trie, keys []string) {
	var end bool
	var dic_ *Trie
	var dic_bk *Trie
	dic_ = dic

	keys_ := keys[0 : len(keys)-1]
	len_keys := len(keys)

	for _, key := range keys_ {
		dic_bk = dic_
		// if dic.matches[key] does not exist
		if _, ok := dic_.Matches[key]; !ok {
			// set dic.matches[key] to &Trie
			dic_.Matches[key] = &Trie{HasChildren: true, Matches: make(map[string]*Trie)}
		}
		dic_ = dic_.Matches[key] // point dic to it
		if len(dic_.Matches) == 0 && !dic_.HasChildren {
			end = true
			dic_ = dic_bk
			dic_.Matches[keys[len_keys-2]] = &Trie{End: true, Matches: make(map[string]*Trie)}
			dic_.Matches[keys[len_keys-2]].Matches[keys[len_keys-1]] = &Trie{Matches: make(map[string]*Trie)}
		}
	}
	if !end {
		dic_.Matches[keys[len_keys-1]] = &Trie{Matches: make(map[string]*Trie)}
	}
}

// Reverse a slice of strings in-place
func reverse(input []string) {
	for i, j := 0, len(input)-1; i < j; i, j = i+1, j-1 {
		input[i], input[j] = input[j], input[i]
	}
}

// Format string as punycode
func formatAsPunycode(s string) string {
	asPunyCode, err := idna.ToASCII(strings.ToLower(strings.TrimSpace(s)))
	if err != nil {
		log.Println(strings.SplitAfterN(err.Error(), "idna: invalid label", 2)[0])
		return ""
	}

	return asPunyCode
}

// Construct a compressed trie to store Public Suffix List TLDs split at "." in reverse-order
//
// For example: "us.gov.pl" will be stored in the order {"pl", "gov", "us"}
func trieConstruct(includePrivateSuffix bool, cacheFilePath string) Dict {
	tldTrie := Dict{}
	suffixLists := getPublicSuffixList(cacheFilePath)

	SuffixList := []string{}
	if !includePrivateSuffix {
		// public suffixes only
		SuffixList = suffixLists[0]
	} else {
		// public suffixes AND private suffixes
		SuffixList = suffixLists[2]
	}

	for _, suffix := range SuffixList {
		if strings.Contains(suffix, ".") {
			sp := strings.Split(suffix, ".")
			reverse(sp)
			NestedDict(tldTrie, sp)
		} else {
			tldTrie[suffix] = Dict{"_END": true}
		}
	}

	for key := range tldTrie {
		if val, ok := tldTrie[key].(Dict); ok {
			if len(val) == 1 {
				if _, ok := tldTrie["_END"]; ok {
					tldTrie[key] = true
				}
			}
		}
	}

	return tldTrie
}

func TrieConstruct2(includePrivateSuffix bool, cacheFilePath string) *Trie {
	tldTrie := &Trie{Matches: make(map[string]*Trie)}
	suffixLists := getPublicSuffixList(cacheFilePath)

	SuffixList := []string{}
	if !includePrivateSuffix {
		// public suffixes only
		SuffixList = suffixLists[0]
	} else {
		// public suffixes AND private suffixes
		SuffixList = suffixLists[2]
	}

	for _, suffix := range SuffixList {
		if strings.Contains(suffix, ".") {
			sp := strings.Split(suffix, ".")
			reverse(sp)
			NestedDict2(tldTrie, sp)
		} else {
			tldTrie.Matches[suffix] = &Trie{End: true, Matches: make(map[string]*Trie)}
		}
	}

	for key := range tldTrie.Matches {
		if len(tldTrie.Matches[key].Matches) == 0 && tldTrie.Matches[key].End {
			tldTrie.Matches[key] = &Trie{Matches: make(map[string]*Trie)}
		}
	}

	return tldTrie
}

// Extract subdomain, domain, suffix and registered domain from a given `url`
//
//  Example: "https://maps.google.com.ua/a/long/path?query=42"
//  subdomain: maps
//  domain: google
//  suffix: com.ua
//  registered domain: maps.google.com.ua
func (f *FastTLD) Extract(e UrlParams) *ExtractResult {
	urlParts := ExtractResult{}

	if e.ConvertURLToPunyCode {
		e.Url = formatAsPunycode(e.Url)
	}

	// Remove URL scheme
	// Credits: https://github.com/mjd2021usa/tldextract/blob/main/tldextract.go
	netloc := e.Url
	netloc = schemeRegex.ReplaceAllString(netloc, "")
	netloc = strings.Trim(netloc, ". \n\t\r")
	var afterHost string

	// Remove URL userinfo
	atIdx := strings.Index(netloc, "@")
	if atIdx != -1 {
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

	// extract port and path if any
	if lenAfterHost := len(afterHost); lenAfterHost != 0 {
		var maybePort string
		hasPort := afterHost[0] == ':'
		pathStartIndex := strings.Index(afterHost, "/")
		var invalidPort bool
		if hasPort {
			if pathStartIndex == -1 {
				maybePort = afterHost[1:]
			} else {
				maybePort = afterHost[1:pathStartIndex]
			}
			if port, err := strconv.Atoi(maybePort); !(err == nil && 0 <= port && port <= 65535) {
				maybePort = ""
				invalidPort = true
			}
			urlParts.Port = maybePort
		}
		if !invalidPort && pathStartIndex != -1 && pathStartIndex != lenAfterHost {
			// fmt.Println(afterHost[pathStartIndex+1:])
		}
	}

	// Determine if url is an IPv4 address
	if looksLikeIPv4Address(netloc) {
		urlParts.Domain = netloc
		urlParts.RegisteredDomain = netloc
		return &urlParts
	}

	labels := strings.Split(netloc, ".")

	var node Dict
	// define the root node
	node = f.TldTrie

	lenSuffix := 0
	suffixCharCount := 0
	lenLabels := len(labels)
	for idx := range labels {
		label := labels[lenLabels-idx-1]
		labelLength := len(label)

		// this node has sub-nodes and maybe an end-node.
		// eg. cn -> (cn, gov.cn)
		if _, ok := node["_END"]; ok {
			// check if there is a sub node
			// eg. gov.cn
			if val, ok := node[label]; ok {
				lenSuffix += 1
				suffixCharCount += labelLength
				if val, ok := val.(Dict); !ok {
					urlParts.Domain = labels[idx-1]
					break
				} else {
					node = val
					continue
				}
			}
		}

		if _, ok := node["*"]; ok {
			// check if there is a sub node
			// e.g. www.ck
			if _, ok := node["!"+label]; !ok {
				lenSuffix += 1
				suffixCharCount += labelLength
			} else {
				urlParts.Domain = label
			}
			break
		}

		// check if TLD in Public Suffix List
		if val, ok := node[label]; ok {
			lenSuffix += 1
			suffixCharCount += labelLength
			if val_, ok := val.(Dict); ok {
				node = val_
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
	len_url_domain := 0

	if 0 < lenSuffix && lenSuffix < lenLabels {
		urlParts.Domain = labels[lenLabels-lenSuffix-1]
		len_url_domain = len(urlParts.Domain)
		if !e.IgnoreSubDomains && (lenLabels-lenSuffix) >= 2 {
			urlParts.SubDomain = netloc[:netlocLen-len_url_domain-len_url_suffix-2]
			// urlParts.SubDomain = strings.Join(labels[0:lenLabels-lenSuffix-1], ".")
		}
	}

	if len_url_domain > 0 && len_url_suffix > 0 {
		urlParts.RegisteredDomain = urlParts.Domain + "." + urlParts.Suffix
	}

	return &urlParts
}

func (f *FastTLD) Extract2(e UrlParams) *ExtractResult {
	urlParts := ExtractResult{}

	if e.ConvertURLToPunyCode {
		e.Url = formatAsPunycode(e.Url)
	}

	// Remove URL scheme
	// Credits: https://github.com/mjd2021usa/tldextract/blob/main/tldextract.go
	netloc := e.Url
	netloc = schemeRegex.ReplaceAllString(netloc, "")
	netloc = strings.Trim(netloc, ". \n\t\r")
	var afterHost string

	// Remove URL userinfo
	atIdx := strings.Index(netloc, "@")
	if atIdx != -1 {
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

	// extract port and path if any
	if lenAfterHost := len(afterHost); lenAfterHost != 0 {
		var maybePort string
		hasPort := afterHost[0] == ':'
		pathStartIndex := strings.Index(afterHost, "/")
		var invalidPort bool
		if hasPort {
			if pathStartIndex == -1 {
				maybePort = afterHost[1:]
			} else {
				maybePort = afterHost[1:pathStartIndex]
			}
			if port, err := strconv.Atoi(maybePort); !(err == nil && 0 <= port && port <= 65535) {
				maybePort = ""
				invalidPort = true
			}
			urlParts.Port = maybePort
		}
		if !invalidPort && pathStartIndex != -1 && pathStartIndex != lenAfterHost {
			// fmt.Println(afterHost[pathStartIndex+1:])
		}
	}

	// Determine if url is an IPv4 address
	if looksLikeIPv4Address(netloc) {
		urlParts.Domain = netloc
		urlParts.RegisteredDomain = netloc
		return &urlParts
	}

	labels := strings.Split(netloc, ".")

	var node *Trie
	// define the root node
	node = f.TldTrie2

	lenSuffix := 0
	suffixCharCount := 0
	lenLabels := len(labels)
	for idx := range labels {
		label := labels[lenLabels-idx-1]
		labelLength := len(label)

		// this node has sub-nodes and maybe an end-node.
		// eg. cn -> (cn, gov.cn)
		if node.End {
			// check if there is a sub node
			// eg. gov.cn
			if val, ok := node.Matches[label]; ok {
				lenSuffix += 1
				suffixCharCount += labelLength
				if len(val.Matches) == 0 {
					urlParts.Domain = labels[idx-1]
					break
				} else {
					node = val
					continue
				}
			}
		}

		if _, ok := node.Matches["*"]; ok {
			// check if there is a sub node
			// e.g. www.ck
			if _, ok := node.Matches["!"+label]; !ok {
				lenSuffix += 1
				suffixCharCount += labelLength
			} else {
				urlParts.Domain = label
			}
			break
		}
		// check if TLD in Public Suffix List
		if val, ok := node.Matches[label]; ok {
			lenSuffix += 1
			suffixCharCount += labelLength
			if len(val.Matches) != 0 {
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
	len_url_domain := 0

	if 0 < lenSuffix && lenSuffix < lenLabels {
		urlParts.Domain = labels[lenLabels-lenSuffix-1]
		len_url_domain = len(urlParts.Domain)
		if !e.IgnoreSubDomains && (lenLabels-lenSuffix) >= 2 {
			urlParts.SubDomain = netloc[:netlocLen-len_url_domain-len_url_suffix-2]
			// urlParts.SubDomain = strings.Join(labels[0:lenLabels-lenSuffix-1], ".")
		}
	}

	if len_url_domain > 0 && len_url_suffix > 0 {
		urlParts.RegisteredDomain = urlParts.Domain + "." + urlParts.Suffix
	}

	return &urlParts
}

// New creates a new *FastTLD
func New(n SuffixListParams) (*FastTLD, error) {
	cachePath := filepath.Dir(n.CacheFilePath)
	if stat, err := os.Stat(cachePath); cachePath == "." || err != nil || !stat.IsDir() {
		// if cachePath is unreachable, use default Public Suffix List
		cachePath = getCurrentFilePath()
		n.CacheFilePath = cachePath + string(os.PathSeparator) + defaultPSLFileName

		// Download new Public Suffix List if local cache does not exist
		// or if local cache is older than 3 days
		autoUpdate(n.CacheFilePath)
	}

	// Construct tldTrie using list located at n.CacheFilePath
	tldTrie := trieConstruct(n.IncludePrivateSuffix, n.CacheFilePath)

	tldTrie2 := TrieConstruct2(n.IncludePrivateSuffix, n.CacheFilePath)

	return &FastTLD{tldTrie, n.CacheFilePath, tldTrie2}, nil
}
