package fasttld

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/afero"
	"golang.org/x/net/idna"
)

var publicSuffixListSources = []string{
	"https://publicsuffix.org/list/public_suffix_list.dat",
	"https://raw.githubusercontent.com/publicsuffix/list/master/public_suffix_list.dat",
}

type suffixes struct {
	publicSuffixes  []string
	privateSuffixes []string
	allSuffixes     []string
}

func processLine(rawLine string, psl suffixes, isPrivateSuffix bool) (suffixes, bool) {
	line := strings.TrimSpace(rawLine)
	if "// ===BEGIN PRIVATE DOMAINS===" == line {
		isPrivateSuffix = true
	}
	if len(line) == 0 || strings.HasPrefix(line, "//") {
		return psl, isPrivateSuffix
	}
	suffix, err := idna.ToASCII(line)
	if err != nil {
		// skip line if unable to convert to ascii
		log.Println(line, '|', err)
		return psl, isPrivateSuffix
	}
	if isPrivateSuffix {
		psl.privateSuffixes = append(psl.privateSuffixes, suffix)
		if suffix != line {
			// add non-punycode version if it is different from punycode version
			psl.privateSuffixes = append(psl.privateSuffixes, line)
		}
	} else {
		psl.publicSuffixes = append(psl.publicSuffixes, suffix)
		if suffix != line {
			// add non-punycode version if it is different from punycode version
			psl.publicSuffixes = append(psl.publicSuffixes, line)
		}
	}
	psl.allSuffixes = append(psl.allSuffixes, suffix)
	if suffix != line {
		// add non-punycode version if it is different from punycode version
		psl.allSuffixes = append(psl.allSuffixes, line)
	}
	return psl, isPrivateSuffix
}

// getPublicSuffixList retrieves Public Suffixes and Private Suffixes from Public Suffix list located at cacheFilePath.
//
// publicSuffixes: ICANN domains. Example: com, net, org etc.
//
// privateSuffixes: PRIVATE domains. Example: blogspot.co.uk, appspot.com etc.
//
// allSuffixes: Both ICANN and PRIVATE domains.
func getPublicSuffixList(cacheFilePath string) (suffixes, error) {
	var psl suffixes
	b, err := os.ReadFile(cacheFilePath)
	if err != nil {
		log.Println(err)
		return psl, err
	}
	var isPrivateSuffix bool
	for _, line := range strings.Split(string(b), "\n") {
		psl, isPrivateSuffix = processLine(line, psl, isPrivateSuffix)
	}
	return psl, nil
}

// getHardcodedPublicSuffixList retrieves Public Suffixes and Private Suffixes from hardcoded Public Suffix list.
//
// publicSuffixes: ICANN domains. Example: com, net, org etc.
//
// privateSuffixes: PRIVATE domains. Example: blogspot.co.uk, appspot.com etc.
//
// allSuffixes: Both ICANN and PRIVATE domains.
func getHardcodedPublicSuffixList() (suffixes, error) {
	var psl suffixes
	var isPrivateSuffix bool
	for _, line := range strings.Split(hardcodedPSL, "\n") {
		psl, isPrivateSuffix = processLine(line, psl, isPrivateSuffix)
	}
	return psl, nil
}

// newHardcodedPSL creates a new *FastTLD using data from a hardcoded Public Suffix List file.
func newHardcodedPSL(err error, n SuffixListParams) (*FastTLD, error) {
	log.Println(err, "Fallback to hardcoded Public Suffix List")
	tldTrie, err := trieConstruct(n.IncludePrivateSuffix, "")
	return &FastTLD{cacheFilePath: "", tldTrie: tldTrie, includePrivateSuffix: n.IncludePrivateSuffix}, err
}

// downloadFile downloads file from url as byte slice
func downloadFile(url string) ([]byte, error) {
	// Make HTTP GET request
	var bodyBytes []byte
	resp, err := http.Get(url)
	if err != nil {
		return bodyBytes, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err = afero.ReadAll(resp.Body)
	} else {
		err = errors.New("Download failed, HTTP status code : " + fmt.Sprint(resp.StatusCode))
	}
	return bodyBytes, err
}

// getCurrentFilePath returns path to current module file
//
// Similar to os.path.dirname(os.path.realpath(__file__)) in Python
//
// Credits: https://andrewbrookins.com/tech/golang-get-directory-of-the-current-file
func getCurrentFilePath() (string, bool) {
	_, file, _, ok := runtime.Caller(0)
	return filepath.Dir(file), ok
}

// Number of hours elapsed since last modified time of fileinfo.
func fileLastModifiedHours(fileinfo os.FileInfo) float64 {
	return time.Now().Sub(fileinfo.ModTime()).Hours()
}

// update updates the local cache of Public Suffix List
func update(file afero.File,
	publicSuffixListSources []string) error {
	for _, publicSuffixListSource := range publicSuffixListSources {
		// Write GET request body to local file
		if bodyBytes, err := downloadFile(publicSuffixListSource); err != nil {
			log.Println(err)
		} else {
			if !validPSLDelimiters(bodyBytes) {
				continue
			}
			if _, err := file.Seek(0, 0); err != nil {
				log.Println(err)
				continue
			}
			if _, err := file.Write(bodyBytes); err != nil {
				log.Println(err)
				continue
			}
			log.Println("Public Suffix List updated.")
			return nil
		}
	}
	return errors.New("failed to fetch any Public Suffix List from all mirrors")
}

func validPSLDelimiters(contents []byte) bool {
	return bytes.Contains(contents, []byte("// ===BEGIN ICANN DOMAINS===")) &&
		bytes.Contains(contents, []byte("// ===END ICANN DOMAINS===")) &&
		bytes.Contains(contents, []byte("// ===BEGIN PRIVATE DOMAINS===")) &&
		bytes.Contains(contents, []byte("// ===END PRIVATE DOMAINS==="))
}

func checkCacheFile(cacheFilePath string) (bool, float64) {
	cacheFilePath, pathValidErr := filepath.Abs(strings.TrimSpace(cacheFilePath))
	stat, fileinfoErr := os.Stat(cacheFilePath)
	var lastModifiedHours float64
	if fileinfoErr == nil {
		lastModifiedHours = fileLastModifiedHours(stat)
	}

	var validDelimiters bool
	if contents, err := os.ReadFile(cacheFilePath); err == nil {
		validDelimiters = validPSLDelimiters(contents)
	}
	return pathValidErr == nil && fileinfoErr == nil && !stat.IsDir() && validDelimiters, lastModifiedHours
}

// Update updates the default Public Suffix list file and updates its suffix trie using the updated file.
// If cache file path is not the same as the default cache file path, this will be a no-op.
func (f *FastTLD) Update() error {
	filesystem := new(afero.OsFs)
	defaultCacheFilePath := afero.GetTempDir(filesystem, "") + defaultPSLFileName

	if f.cacheFilePath != defaultCacheFilePath {
		return errors.New("No-op. Only default Public Suffix list file can be updated")
	}
	file, err := os.OpenFile(defaultCacheFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	if updateErr := update(file, publicSuffixListSources); updateErr != nil {
		return updateErr
	}
	tldTrie, err := trieConstruct(f.includePrivateSuffix, defaultCacheFilePath)
	if err == nil {
		f.tldTrie = tldTrie
		f.cacheFilePath = defaultCacheFilePath
	}
	return err
}
