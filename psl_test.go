// go-fasttld is a high performance top level domains (TLD)
// extraction module implemented with compressed tries.
//
// This module is a port of the Python fasttld module,
// with additional modifications to support extraction
// of subcomponents from full URLs and IPv4 addresses.
package fasttld

import (
	"reflect"
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
		isIPv4Address := looksLikeIPv4Address(test.maybeIPv4Address)
		if isIPv4Address != test.isIPv4Address {
			t.Errorf("Output %t not equal to expected %t",
				isIPv4Address, test.isIPv4Address)
		}
	}
}

type getPublicSuffixListTest struct {
	cacheFilePath string
	expected      [3]([]string)
}

var getPublicSuffixListTests = []getPublicSuffixListTest{
	{cacheFilePath: "test/public_suffix_list.dat",
		expected: pslTestLists,
	},
}

func TestGetPublicSuffixList(t *testing.T) {
	for _, test := range getPublicSuffixListTests {
		suffixLists := getPublicSuffixList(test.cacheFilePath)
		if output := reflect.DeepEqual(suffixLists,
			test.expected); !output {
			t.Errorf("Output %q not equal to expected %q",
				suffixLists, test.expected)
		}
	}
}
