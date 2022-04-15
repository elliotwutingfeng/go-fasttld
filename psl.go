package fasttld

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

const (
	publicSuffixListSource         string = "https://publicsuffix.org/list/public_suffix_list.dat"
	publicSuffixListSourceFallback string = "https://raw.githubusercontent.com/publicsuffix/list/master/public_suffix_list.dat"
)

// Extract URL scheme from string
var schemeRegex = regexp.MustCompile("^[A-Za-z0-9+-.]+://")

// Return true if `maybeIPv4Address` is an IPv4 address
func looksLikeIPv4Address(maybeIPv4Address string) bool {
	return net.ParseIP(maybeIPv4Address) != nil
}

// Extract Public Suffixes and Private Suffixes from Public Suffix list located at `cacheFilePath`
//
// PublicSuffixes: ICANN domains. Example: com, net, org etc.
//
// PrivateSuffixes: PRIVATE domains. Example: blogspot.co.uk, appspot.com etc.
//
// AllSuffixes: Both ICANN and PRIVATE domains.
func getPublicSuffixList(cacheFilePath string) [3]([]string) {
	PublicSuffixes := []string{}
	PrivateSuffixes := []string{}
	AllSuffixes := []string{}

	if _, err := os.Stat(cacheFilePath); err != nil {
		// if file at cacheFilePath does not exist
		log.Fatal("Path: " + cacheFilePath + " | Public suffix list file not found.")
	}
	fd, err := os.Open(cacheFilePath)
	if err != nil {
		log.Fatal(err)
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
			log.Fatal(err)
		}
		if isPrivateSuffix {
			PrivateSuffixes = append(PrivateSuffixes, suffix)
			if suffix != line {
				// add non-punycode version if it is different from punycode version
				PrivateSuffixes = append(PrivateSuffixes, line)
			}
		} else {
			PublicSuffixes = append(PublicSuffixes, suffix)
			if suffix != line {
				// add non-punycode version if it is different from punycode version
				PublicSuffixes = append(PublicSuffixes, line)
			}
		}
		AllSuffixes = append(AllSuffixes, suffix)
		if suffix != line {
			// add non-punycode version if it is different from punycode version
			AllSuffixes = append(AllSuffixes, line)
		}

	}
	return [3]([]string){PublicSuffixes, PrivateSuffixes, AllSuffixes}
}

// Download a url to a local file without loading the whole file into memory
func downloadFile(cacheFilePath string, url string) error {
	// Make HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create local file at cacheFilePath
	out, err := os.Create(cacheFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write GET request body to local file
	_, err = io.Copy(out, resp.Body)
	return err
}

// Update local cache of Public Suffix List
//
// This function will update the local cache of Public Suffix List if it is more than 3 days old
func autoUpdate(cacheFilePath string) {
	cacheFileNeedsUpdate := false
	if file, err := os.Stat(cacheFilePath); err == nil {
		// if file at cacheFilePath exists
		// check if it needs to be updated (requirement: older than 3 days)
		modifiedtime := file.ModTime()
		if time.Now().Sub(modifiedtime).Hours() > 72 {
			cacheFileNeedsUpdate = true
		}
	} else if errors.Is(err, os.ErrNotExist) {
		// file at cacheFilePath does not exist
		cacheFileNeedsUpdate = true
	} else {
		// file may or may not exist. Treat file as non-existent
		cacheFileNeedsUpdate = true
	}
	if cacheFileNeedsUpdate {
		showLogMessages := false
		update(cacheFilePath, showLogMessages)
	}
}

// Get path to current module file
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

// Update local cache of Public Suffix List
func update(cacheFilePath string, showLogMessages bool) error {
	download_success := false
	// Try main source
	if err := downloadFile(cacheFilePath, publicSuffixListSource); err != nil {
		log.Println(err)
	} else {
		download_success = true
	}
	// If that fails, try fallback source
	if !download_success {
		if err := downloadFile(cacheFilePath, publicSuffixListSourceFallback); err != nil {
			log.Println(err)
			errorMsg := "Failed to fetch Public Suffix List from both main source and fallback source"
			return errors.New(errorMsg)
		} else {
			download_success = true
		}
	}

	if download_success && showLogMessages {
		log.Println(filepath.Base(cacheFilePath), "downloaded")
	}

	return nil
}

func (t *FastTLD) Update(showLogMessages bool) error {
	return update(t.cacheFilePath, showLogMessages)
}
