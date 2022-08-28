package fasttld

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
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

	fd, err := os.Open(cacheFilePath)
	if err != nil {
		log.Println(err)
		return psl, err
	}
	defer fd.Close()

	fileScanner := bufio.NewScanner(fd)
	fileScanner.Split(bufio.ScanLines)
	var isPrivateSuffix bool
	for fileScanner.Scan() {
		psl, isPrivateSuffix = processLine(fileScanner.Text(), psl, isPrivateSuffix)
	}
	return psl, nil
}

// getInlinePublicSuffixList retrieves Public Suffixes and Private Suffixes from inline Public Suffix list.
//
// publicSuffixes: ICANN domains. Example: com, net, org etc.
//
// privateSuffixes: PRIVATE domains. Example: blogspot.co.uk, appspot.com etc.
//
// allSuffixes: Both ICANN and PRIVATE domains.
func getInlinePublicSuffixList() (suffixes, error) {
	var psl suffixes
	var isPrivateSuffix bool
	for _, line := range strings.Split(inlinePSL, "\n") {
		psl, isPrivateSuffix = processLine(line, psl, isPrivateSuffix)
	}
	return psl, nil
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
		bodyBytes, err = io.ReadAll(resp.Body)
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
func fileLastModifiedHours(fileinfo fs.FileInfo) float64 {
	return time.Now().Sub(fileinfo.ModTime()).Hours()
}

// update updates the local cache of Public Suffix List
func update(file afero.File,
	publicSuffixListSources []string) error {
	var downloadSuccess bool
	for _, publicSuffixListSource := range publicSuffixListSources {
		// Write GET request body to local file
		if bodyBytes, err := downloadFile(publicSuffixListSource); err != nil {
			log.Println(err)
		} else {
			file.Seek(0, 0)
			file.Write(bodyBytes)
			downloadSuccess = true
			break
		}
	}
	if !downloadSuccess {
		return errors.New("failed to fetch any Public Suffix List from all mirrors")
	}
	log.Println("Public Suffix List updated.")
	return nil
}

func getDefaultCachePaths() (string, string, error) {
	currentFilePath, ok := getCurrentFilePath()
	if !ok {
		return "", "", errors.New("Cannot get path to current module file")
	}
	defaultCacheFolderPath := currentFilePath + string(os.PathSeparator) + defaultPSLFolder
	defaultCacheFilePath := defaultCacheFolderPath + string(os.PathSeparator) + defaultPSLFileName

	return defaultCacheFolderPath, defaultCacheFilePath, nil
}

// Update updates the default Public Suffix list file.
func (t *FastTLD) Update() error {
	defaultCacheFolderPath, defaultCacheFilePath, err := getDefaultCachePaths()
	if err := os.MkdirAll(defaultCacheFolderPath, 0777); err != nil {
		return err
	}
	file, err := os.Create(defaultCacheFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return update(file, publicSuffixListSources)
}
