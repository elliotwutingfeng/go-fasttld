package fasttld

import "testing"

type looksLikeIPAddressTest struct {
	maybeIPAddress string
	isIPAddress    bool
}

var looksLikeIPv4AddressTests = []looksLikeIPAddressTest{
	{maybeIPAddress: "",
		isIPAddress: false,
	},
	{maybeIPAddress: " ",
		isIPAddress: false,
	},
	{maybeIPAddress: "google.com",
		isIPAddress: false,
	},
	{maybeIPAddress: "1google.com",
		isIPAddress: false,
	},
	{maybeIPAddress: "127.0.0.1",
		isIPAddress: true,
	},
	{maybeIPAddress: "127.0.0.256",
		isIPAddress: false,
	},
}

var looksLikeIPv6AddressTests = []looksLikeIPAddressTest{
	{maybeIPAddress: "",
		isIPAddress: false,
	},
	{maybeIPAddress: " ",
		isIPAddress: false,
	},
	{maybeIPAddress: "google.com",
		isIPAddress: false,
	},
	{maybeIPAddress: "1google.com",
		isIPAddress: false,
	},
	{maybeIPAddress: "aBcD:ef01:2345:6789:aBcD:ef01:2345:6789",
		isIPAddress: true,
	},
	{maybeIPAddress: "gGgG:ef01:2345:6789:aBcD:ef01:2345:6789",
		isIPAddress: false,
	},
	{maybeIPAddress: "aBcD:ef01:2345:6789:aBcD:ef01:127.0.0.1",
		isIPAddress: true,
	},
	{maybeIPAddress: "aBcD:ef01:2345:6789:aBcD:ef01:127.0.0.256",
		isIPAddress: false,
	},
}

func TestParseIPv4(t *testing.T) {
	for _, test := range looksLikeIPv4AddressTests {
		isIPv4Address := parseIPv4(test.maybeIPAddress) != nil
		if isIPv4Address != test.isIPAddress {
			t.Errorf("Output %t not equal to expected %t",
				isIPv4Address, test.isIPAddress)
		}
	}
}

func TestParseIPv6(t *testing.T) {
	for _, test := range looksLikeIPv6AddressTests {
		isIPv6Address := parseIPv6(test.maybeIPAddress) != nil
		if isIPv6Address != test.isIPAddress {
			t.Errorf("Output %t not equal to expected %t",
				isIPv6Address, test.isIPAddress)
		}
	}
}
