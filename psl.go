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

// getPublicSuffixList retrieves Public Suffixes and Private Suffixes from Public Suffix list located at cacheFilePath.
//
// publicSuffixes: ICANN domains. Example: com, net, org etc.
//
// privateSuffixes: PRIVATE domains. Example: blogspot.co.uk, appspot.com etc.
//
// allSuffixes: Both ICANN and PRIVATE domains.
func getPublicSuffixList(cacheFilePath string) (suffixes, error) {
	publicSuffixes := []string{}
	privateSuffixes := []string{}
	allSuffixes := []string{}

	fd, err := os.Open(cacheFilePath)
	if err != nil {
		log.Println(err)
		return suffixes{publicSuffixes, privateSuffixes, allSuffixes}, err
	}
	defer fd.Close()

	fileScanner := bufio.NewScanner(fd)
	fileScanner.Split(bufio.ScanLines)
	isPrivateSuffix := false
	for fileScanner.Scan() {
		line := strings.TrimSpace(fileScanner.Text())
		if "// ===BEGIN PRIVATE DOMAINS===" == line {
			isPrivateSuffix = true
		}
		if len(line) == 0 || strings.HasPrefix(line, "//") {
			continue
		}
		suffix, err := idna.ToASCII(line)
		if err != nil {
			// skip line if unable to convert to ascii
			log.Println(line, '|', err)
			continue
		}
		if isPrivateSuffix {
			privateSuffixes = append(privateSuffixes, suffix)
			if suffix != line {
				// add non-punycode version if it is different from punycode version
				privateSuffixes = append(privateSuffixes, line)
			}
		} else {
			publicSuffixes = append(publicSuffixes, suffix)
			if suffix != line {
				// add non-punycode version if it is different from punycode version
				publicSuffixes = append(publicSuffixes, line)
			}
		}
		allSuffixes = append(allSuffixes, suffix)
		if suffix != line {
			// add non-punycode version if it is different from punycode version
			allSuffixes = append(allSuffixes, line)
		}
	}
	return suffixes{publicSuffixes, privateSuffixes, allSuffixes}, nil
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
func getCurrentFilePath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Cannot get current module file path")
	}
	return filepath.Dir(file)
}

// Number of hours elapsed since last modified time of fileinfo.
func fileLastModifiedHours(fileinfo fs.FileInfo) float64 {
	return time.Now().Sub(fileinfo.ModTime()).Hours()
}

// update updates the local cache of Public Suffix List
func update(file afero.File,
	publicSuffixListSources []string) error {
	downloadSuccess := false
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
	if downloadSuccess {
		log.Println("Public Suffix List updated.")
	} else {
		return errors.New("failed to fetch any Public Suffix List from all mirrors")
	}

	return nil
}

// Update updates the local cache of Public Suffix list if t.cacheFilePath is not
// the same as path to current module file (i.e. no custom file path specified).
func (t *FastTLD) Update() error {
	if t.cacheFilePath != getCurrentFilePath()+string(os.PathSeparator)+defaultPSLFileName {
		return errors.New("function Update() only applies to default Public Suffix List, not custom Public Suffix List")
	}
	file, err := os.Create(t.cacheFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return update(file, publicSuffixListSources)
}
