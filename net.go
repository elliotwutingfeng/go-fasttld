package fasttld

import (
	"strings"
	"unicode/utf8"
)

// IP address lengths (bytes).
const (
	IPv4len = 4
	IPv6len = 16
)

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

// isIPv4 returns true if s is a literal IPv4 address
func isIPv4(s string) bool {
	for i := 0; i < IPv4len; i++ {
		if len(s) == 0 {
			// Missing octets.
			return false
		}
		if i > 0 {
			r, size := utf8.DecodeRuneInString(s)
			if strings.IndexRune(labelSeparators, r) == -1 {
				return false
			}
			s = s[size:]
		}
		n, c, ok := dtoi(s)
		if !ok || n > 0xFF {
			return false
		}
		if c > 1 && s[0] == '0' {
			// Reject non-zero components with leading zeroes.
			return false
		}
		s = s[c:]
	}
	if len(s) != 0 {
		return false
	}
	return true
}

// isIPv6 returns true if s is a literal IPv6 address as described in RFC 4291
// and RFC 5952.
func isIPv6(s string) bool {
	ellipsis := -1 // position of ellipsis in ip

	// Might have leading ellipsis
	if len(s) >= 2 && s[0] == ':' && s[1] == ':' {
		ellipsis = 0
		s = s[2:]
		// Might be only ellipsis
		if len(s) == 0 {
			return true
		}
	}

	// Loop, parsing hex numbers followed by colon.
	i := 0
	for i < IPv6len {
		// Hex number.
		n, c, ok := xtoi(s)
		if !ok || n > 0xFFFF {
			return false
		}

		// If followed by any separator in labelSeparators, might be in trailing IPv4.
		if c < len(s) && strings.IndexRune(labelSeparators, []rune(s[c:])[0]) != -1 {
			if ellipsis < 0 && i != IPv6len-IPv4len {
				// Not the right place.
				return false
			}
			if i+IPv4len > IPv6len {
				// Not enough room.
				return false
			}
			if !isIPv4(s) {
				return false
			}
			s = ""
			i += IPv4len
			break
		}

		// Save this 16-bit chunk.
		i += 2

		// Stop at end of string.
		s = s[c:]
		if len(s) == 0 {
			break
		}

		// Otherwise must be followed by colon and more.
		if s[0] != ':' || len(s) == 1 {
			return false
		}
		s = s[1:]

		// Look for ellipsis.
		if s[0] == ':' {
			if ellipsis >= 0 { // already have one
				return false
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
		return false
	}

	// If didn't parse enough, expand ellipsis.
	if i < IPv6len {
		if ellipsis < 0 {
			return false
		}
	} else if ellipsis >= 0 {
		// Ellipsis must represent at least one 0 group.
		return false
	}
	return true
}
