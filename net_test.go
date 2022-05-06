package fasttld

import (
	"testing"
)

type looksLikeIPv4AddressTest struct {
	maybeIPv4Address string
	isIPv4Address    bool
}

var looksLikeIPv4AddressTests = []looksLikeIPv4AddressTest{
	{maybeIPv4Address: "",
		isIPv4Address: false,
	},
	{maybeIPv4Address: "google.com",
		isIPv4Address: false,
	},
	{maybeIPv4Address: "1google.com",
		isIPv4Address: false,
	},
	{maybeIPv4Address: "127.0.0.1",
		isIPv4Address: true,
	},
}

func TestLooksLikeIPv4Address(t *testing.T) {
	for _, test := range looksLikeIPv4AddressTests {
		isIPv4Address := looksLikeIPAddress(test.maybeIPv4Address)
		if isIPv4Address != test.isIPv4Address {
			t.Errorf("Output %t not equal to expected %t",
				isIPv4Address, test.isIPv4Address)
		}
	}
}
