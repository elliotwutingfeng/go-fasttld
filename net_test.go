package fasttld

import (
	"testing"
)

type looksLikeIPAddressTest struct {
	maybeIPAddress string
	isIPAddress    bool
}

var looksLikeIPAddressTests = []looksLikeIPAddressTest{
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

func TestLooksLikeIPAddress(t *testing.T) {
	for _, test := range looksLikeIPAddressTests {
		isIPAddress := looksLikeIPAddress(test.maybeIPAddress)
		if isIPAddress != test.isIPAddress {
			t.Errorf("Output %t not equal to expected %t",
				isIPAddress, test.isIPAddress)
		}
	}
}
