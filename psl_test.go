// go-fasttld is a high performance top level domains (TLD)
// extraction module implemented with compressed tries.
//
// This module is a port of the Python fasttld module,
// with additional modifications to support extraction
// of subcomponents from full URLs and IPv4 addresses.
package fasttld

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/spf13/afero"
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
	expectedLists [3]([]string)
	hasError      bool
}

var getPublicSuffixListTests = []getPublicSuffixListTest{
	{cacheFilePath: "test/public_suffix_list.dat",
		expectedLists: pslTestLists,
		hasError:      false,
	},
	{cacheFilePath: "test/mini_public_suffix_list.dat",
		expectedLists: [3][]string{{"ac", "com.ac", "edu.ac", "gov.ac", "net.ac",
			"mil.ac", "org.ac", "*.ck", "!www.ck"}, {"blogspot.com"},
			{"ac", "com.ac", "edu.ac", "gov.ac", "net.ac", "mil.ac",
				"org.ac", "*.ck", "!www.ck", "blogspot.com"}},
		hasError: false,
	},
	{cacheFilePath: "test/public_suffix_list.dat.noexist",
		expectedLists: [3][]string{{}, {}, {}},
		hasError:      true,
	},
}

func TestGetPublicSuffixList(t *testing.T) {
	for _, test := range getPublicSuffixListTests {
		suffixLists, err := getPublicSuffixList(test.cacheFilePath)
		if test.hasError && err == nil {
			t.Errorf("Expected an error. Got no error.")
		}
		if !test.hasError && err != nil {
			t.Errorf("Expected no error. Got an error.")
		}
		if output := reflect.DeepEqual(suffixLists,
			test.expectedLists); !output {
			t.Errorf("Output %q not equal to expected %q",
				suffixLists, test.expectedLists)
		}
	}
}

func TestDownloadFile(t *testing.T) {
	expectedResponse := []byte(`{"isItSunday": true}`)
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(expectedResponse)
	}))
	defer goodServer.Close()
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer badServer.Close()

	// HTTP Status Code 200
	res, _ := downloadFile(goodServer.URL)
	if output := reflect.DeepEqual(expectedResponse,
		res); !output {
		t.Errorf("Output %q not equal to expected %q",
			res, expectedResponse)
	}

	// HTTP Status Code 404
	res, _ = downloadFile(badServer.URL)
	if len(res) != 0 {
		t.Errorf("Response should be empty.")
	}

	// Malformed URL
	res, _ = downloadFile("!example.com")
	if len(res) != 0 {
		t.Errorf("Response should be empty.")
	}
}

func TestUpdateCustomSuffixList(t *testing.T) {
	extractor, err := New(SuffixListParams{CacheFilePath: "test/mini_public_suffix_list.dat"})
	if err != nil {
		t.Errorf("%q", err)
	}
	if err = extractor.Update(); err == nil {
		t.Errorf("Expected error when trying to Update() custom Public Suffix List.")
	}
}

type updateTest struct {
	mainServerAvailable, fallbackServerAvailable, expectError bool
}

var updateTests = []updateTest{
	{true, true, false},
	{true, false, false},
	{false, true, false},
	{false, false, true},
}

func TestUpdate(t *testing.T) {
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte{})
	}))
	defer goodServer.Close()
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer badServer.Close()

	filesystem := new(afero.MemMapFs)
	file, _ := afero.TempFile(filesystem, "", "ioutil-test")
	defer file.Close()
	for _, test := range updateTests {
		var primarySource, fallbackSource string
		if test.mainServerAvailable {
			primarySource = goodServer.URL
		} else {
			primarySource = badServer.URL
		}
		if test.fallbackServerAvailable {
			fallbackSource = goodServer.URL
		} else {
			fallbackSource = badServer.URL
		}

		// error should only be returned if Public Suffix List cannot
		// be downloaded from any of the sources.
		err := update(file, []string{primarySource, fallbackSource})
		if test.expectError && err == nil {
			t.Errorf("Expected update() error, got no error.")
		}
		if !test.expectError && err != nil {
			t.Errorf("Expected no update() error, got an error.")
		}
	}
}
