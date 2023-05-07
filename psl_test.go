package fasttld

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/spf13/afero"
)

type getPublicSuffixListTest struct {
	cacheFilePath string
	expectedLists suffixes
	hasError      bool
}

var getPublicSuffixListTests = []getPublicSuffixListTest{
	{cacheFilePath: fmt.Sprintf("test%spublic_suffix_list.dat", string(os.PathSeparator)),
		expectedLists: pslTestLists,
		hasError:      false,
	},
	{cacheFilePath: fmt.Sprintf("test%smini_public_suffix_list.dat", string(os.PathSeparator)),
		expectedLists: suffixes{[]string{"ac", "com.ac", "edu.ac", "gov.ac", "net.ac",
			"mil.ac", "org.ac", "*.ck", "!www.ck", "org.sg"}, []string{"blogspot.com"},
			[]string{"ac", "com.ac", "edu.ac", "gov.ac", "net.ac", "mil.ac",
				"org.ac", "*.ck", "!www.ck", "org.sg", "blogspot.com"}},
		hasError: false,
	},
	{cacheFilePath: fmt.Sprintf("test%spublic_suffix_list.dat.noexist", string(os.PathSeparator)),
		expectedLists: suffixes{[]string{}, []string{}, []string{}},
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
			test.expectedLists); !output && (len(suffixLists.publicSuffixes)+
			len(suffixLists.privateSuffixes)+
			len(suffixLists.allSuffixes)+
			len(test.expectedLists.publicSuffixes)+
			len(test.expectedLists.privateSuffixes)+
			len(test.expectedLists.allSuffixes)) != 0 {
			t.Errorf("Output %q not equal to expected %q",
				suffixLists, test.expectedLists)
		}
	}
}

func TestGetHardcodedPublicSuffixList(t *testing.T) {
	suffixLists, err := getHardcodedPublicSuffixList()
	if err != nil {
		t.Errorf("Expected no error. Got an error.")
	}
	if len(suffixLists.publicSuffixes) == 0 {
		t.Errorf("len(suffixLists.publicSuffixes) should be more than 0.")
	}
	if len(suffixLists.privateSuffixes) == 0 {
		t.Errorf("len(suffixLists.privateSuffixes) should be more than 0.")
	}
	if len(suffixLists.allSuffixes) == 0 {
		t.Errorf("len(suffixLists.allSuffixes) should be more than 0.")
	}
}

func TestNewHardcodedPSL(t *testing.T) {
	f, err := newHardcodedPSL(nil, SuffixListParams{})
	if err != nil {
		t.Errorf("newHardcodedPSL error: %q", err)
	}
	if f.tldTrie.matches.Len() == 0 {
		t.Errorf("tldTrie should not be empty")
	}
}

func TestDownloadFile(t *testing.T) {
	expectedResponse := []byte(`{"isItSunday": true}`)
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(expectedResponse)
		r.Header.Get("") // removes unused parameter warning
	}))
	defer goodServer.Close()
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		r.Header.Get("") // removes unused parameter warning
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
	requiredComments := "// ===BEGIN ICANN DOMAINS===\n// ===END ICANN DOMAINS===\n// ===BEGIN PRIVATE DOMAINS===\n// ===END PRIVATE DOMAINS==="
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(requiredComments))
		r.Header.Get("") // removes unused parameter warning
	}))
	defer goodServer.Close()
	emptyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(""))
		r.Header.Get("") // removes unused parameter warning
	}))
	defer emptyServer.Close()
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		r.Header.Get("") // removes unused parameter warning
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

		// error should only be returned if Public Suffix List with requiredComments cannot
		// be downloaded from any of the sources.
		err := update(file, []string{primarySource, fallbackSource})
		if test.expectError && err == nil {
			t.Errorf("Expected update() error, got no error.")
		}
		if !test.expectError && err != nil {
			t.Errorf("Expected no update() error, got an error.")
		}
	}

	// None of the servers return content with requiredComments
	if err := update(file, []string{emptyServer.URL, emptyServer.URL}); err == nil {
		t.Errorf("Expected update() error, got no error.")
	}
}

func TestFileLastModifiedHours(t *testing.T) {
	filesystem := new(afero.MemMapFs)
	file, _ := afero.TempFile(filesystem, "", "ioutil-test")
	fileinfo, _ := filesystem.Stat(file.Name())
	if hours := fileLastModifiedHours(fileinfo); int(hours) != 0 {
		t.Errorf("Expected hours elapsed since last modification to be 0 immediately after file creation. %f", hours)
	}
	defer file.Close()
}
