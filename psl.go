// go-fasttld is a high performance top level domains (TLD)
// extraction module implemented with compressed tries.
//
// This module is a port of the Python fasttld module,
// with additional modifications to support extraction
// of subcomponents from full URLs and IPv4 addresses.
package fasttld

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/spf13/afero"
	"golang.org/x/net/idna"
)

var publicSuffixListSources = []string{
	"https://publicsuffix.org/list/public_suffix_list.dat",
	"https://raw.githubusercontent.com/publicsuffix/list/master/public_suffix_list.dat",
}

// An IP is a single IP address, a slice of bytes.
// Functions in this package accept either 4-byte (IPv4)
// or 16-byte (IPv6) slices as input.
//
// Note that in this documentation, referring to an
// IP address as an IPv4 address or an IPv6 address
// is a semantic property of the address, not just the
// length of the byte slice: a 16-byte slice can still
// be an IPv4 address.
type IP []byte

// IP address lengths (bytes).
const (
	IPv4len = 4
	IPv6len = 16
)

var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

// Bigger than we need, not too big to worry about overflow
const big = 0xFFFFFF

// Decimal to integer.
// Returns number, characters consumed, success.
func dtoi(s string) (n int, i int, ok bool) {
	n = 0
	for i = 0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
		if n >= big {
			return big, i, false
		}
	}
	if i == 0 {
		return 0, 0, false
	}
	return n, i, true
}

// IPv4 returns the IP address (in 16-byte form) of the
// IPv4 address a.b.c.d.
func IPv4(a, b, c, d byte) IP {
	p := make(IP, IPv6len)
	copy(p, v4InV6Prefix)
	p[12] = a
	p[13] = b
	p[14] = c
	p[15] = d
	return p
}

// Parse IPv4 address (d.d.d.d).
func parseIPv4(s string) IP {
	var p [IPv4len]byte
	for i := 0; i < IPv4len; i++ {
		if len(s) == 0 {
			// Missing octets.
			return nil
		}
		if i > 0 {
			r, size := utf8.DecodeRuneInString(s)
			if r == '.' || r == '\u3002' || r == '\uff0e' || r == '\uff61' {
				s = s[size:]
			} else {
				return nil
			}

		}
		n, c, ok := dtoi(s)
		if !ok || n > 0xFF {
			return nil
		}
		if c > 1 && s[0] == '0' {
			// Reject non-zero components with leading zeroes.
			return nil
		}
		s = s[c:]
		p[i] = byte(n)
	}
	if len(s) != 0 {
		return nil
	}
	return IPv4(p[0], p[1], p[2], p[3])
}

// parseIPv6 parses s as a literal IPv6 address described in RFC 4291
// and RFC 5952.
func parseIPv6(s string) (ip IP) {
	ip = make(IP, IPv6len)
	ellipsis := -1 // position of ellipsis in ip

	// Might have leading ellipsis
	if len(s) >= 2 && s[0] == ':' && s[1] == ':' {
		ellipsis = 0
		s = s[2:]
		// Might be only ellipsis
		if len(s) == 0 {
			return ip
		}
	}

	// Loop, parsing hex numbers followed by colon.
	i := 0
	for i < IPv6len {
		// Hex number.
		n, c, ok := xtoi(s)
		if !ok || n > 0xFFFF {
			return nil
		}

		// If followed by dot, might be in trailing IPv4.
		// TODO: handle internationalised period delimiters for trailing IPv4.
		if c < len(s) && s[c] == '.' {
			if ellipsis < 0 && i != IPv6len-IPv4len {
				// Not the right place.
				return nil
			}
			if i+IPv4len > IPv6len {
				// Not enough room.
				return nil
			}
			ip4 := parseIPv4(s)
			if ip4 == nil {
				return nil
			}
			ip[i] = ip4[12]
			ip[i+1] = ip4[13]
			ip[i+2] = ip4[14]
			ip[i+3] = ip4[15]
			s = ""
			i += IPv4len
			break
		}

		// Save this 16-bit chunk.
		ip[i] = byte(n >> 8)
		ip[i+1] = byte(n)
		i += 2

		// Stop at end of string.
		s = s[c:]
		if len(s) == 0 {
			break
		}

		// Otherwise must be followed by colon and more.
		if s[0] != ':' || len(s) == 1 {
			return nil
		}
		s = s[1:]

		// Look for ellipsis.
		if s[0] == ':' {
			if ellipsis >= 0 { // already have one
				return nil
			}
			ellipsis = i
			s = s[1:]
			if len(s) == 0 { // can be at end
				break
			}
		}
	}

	// Must have used entire string.
	if len(s) != 0 {
		return nil
	}

	// If didn't parse enough, expand ellipsis.
	if i < IPv6len {
		if ellipsis < 0 {
			return nil
		}
		n := IPv6len - i
		for j := i - 1; j >= ellipsis; j-- {
			ip[j+n] = ip[j]
		}
		for j := ellipsis + n - 1; j >= ellipsis; j-- {
			ip[j] = 0
		}
	} else if ellipsis >= 0 {
		// Ellipsis must represent at least one 0 group.
		return nil
	}
	return ip
}

// Hexadecimal to integer.
// Returns number, characters consumed, success.
func xtoi(s string) (n int, i int, ok bool) {
	n = 0
	for i = 0; i < len(s); i++ {
		if '0' <= s[i] && s[i] <= '9' {
			n *= 16
			n += int(s[i] - '0')
		} else if 'a' <= s[i] && s[i] <= 'f' {
			n *= 16
			n += int(s[i]-'a') + 10
		} else if 'A' <= s[i] && s[i] <= 'F' {
			n *= 16
			n += int(s[i]-'A') + 10
		} else {
			break
		}
		if n >= big {
			return 0, i, false
		}
	}
	if i == 0 {
		return 0, i, false
	}
	return n, i, true
}

// ParseIP parses s as an IP address, returning the result.
// The string s can be in IPv4 dotted decimal ("192.0.2.1"), IPv6
// ("2001:db8::68"), or IPv4-mapped IPv6 ("::ffff:192.0.2.1") form.
// If s is not a valid textual representation of an IP address,
// ParseIP returns nil.
func parseIP(s string) IP {
	for _, char := range s {
		switch char {
		case '\u002e', '\u3002', '\uff0e', '\uff61':
			return parseIPv4(s)
		case ':':
			return parseIPv6(s)
		}
	}
	return nil
}

// looksLikeIPv4Address returns true if maybeIPv4Address is an IPv4 address
func looksLikeIPv4Address(maybeIPv4Address string) bool {
	return parseIP(maybeIPv4Address) != nil
}

// getPublicSuffixList retrieves Public Suffixes and Private Suffixes from Public Suffix list located at cacheFilePath.
//
// PublicSuffixes: ICANN domains. Example: com, net, org etc.
//
// PrivateSuffixes: PRIVATE domains. Example: blogspot.co.uk, appspot.com etc.
//
// AllSuffixes: Both ICANN and PRIVATE domains.
func getPublicSuffixList(cacheFilePath string) ([3]([]string), error) {
	PublicSuffixes := []string{}
	PrivateSuffixes := []string{}
	AllSuffixes := []string{}

	fd, err := os.Open(cacheFilePath)
	if err != nil {
		log.Println(err)
		return [3]([]string){PublicSuffixes, PrivateSuffixes, AllSuffixes}, err
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
	return [3]([]string){PublicSuffixes, PrivateSuffixes, AllSuffixes}, nil
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
	// Create local file at cacheFilePath
	file, err := os.Create(t.cacheFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return update(file, publicSuffixListSources)
}
