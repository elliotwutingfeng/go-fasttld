package fasttld

import (
	"testing"

	tlde "github.com/M507/tlde/src"
	joeguotldextract "github.com/joeguo/tldextract"
	tld "github.com/jpillora/go-tld"
	mjd2021usatldextract "github.com/mjd2021usa/tldextract"
)

var benchmarkURLs = []string{
	"https://news.google.com", "https://iupac.org/iupac-announces-the-2021-top-ten-emerging-technologies-in-chemistry/",
	"https://www.google.com/maps/dir/Parliament+Place,+Parliament+House+Of+Singapore,+" +
		"Singapore/Parliament+St,+London,+UK/@25.2440033,33.6721455,4z/data=!3m1!4b1!4m14!4m13!1m5!1m1!1s0x31d" +
		"a19a0abd4d71d:0xeda26636dc4ea1dc!2m2!1d103.8504863!2d1.2891543!1m5!1m1!1s0x487604c5aaa7da5b:0xf13a2" +
		"197d7e7dd26!2m2!1d-0.1260826!2d51.5017061!3e4",
}

var benchmarkURL = benchmarkURLs[0]

// this module
func BenchmarkGoFastTld(b *testing.B) {
	extractorWithoutPrivateSuffix, _ := New(SuffixListParams{
		CacheFilePath:        getTestPSLFilePath(),
		IncludePrivateSuffix: false,
	})
	extractor := extractorWithoutPrivateSuffix

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.Extract(URLParams{
			URL: benchmarkURL})
	}
}

// github.com/jpillora/go-tld
func BenchmarkJPilloraGoTld(b *testing.B) {
	// this module also provides the PORT and PATH subcomponents
	// it cannot handle "+://google.com" and IP addresses
	// it cannot handle urls without scheme component

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tld.Parse(benchmarkURL)
	}
}

// github.com/joeguo/tldextract
func BenchmarkJoeGuoTldExtract(b *testing.B) {
	cache := "/tmp/tld.cache"
	extract, _ := joeguotldextract.New(cache, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extract.Extract(benchmarkURL)
	}
}

// github.com/mjd2021usa/tldextract
func BenchmarkMjd2021USATldExtract(b *testing.B) {
	cache := "/tmp/tld.cache"
	extract, _ := mjd2021usatldextract.New(cache, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extract.Extract(benchmarkURL)
	}
}

// github.com/M507/tlde
func BenchmarkM507Tlde(b *testing.B) {
	// Appears to be the same as github.com/joeguo/tldextract
	cache := "/tmp/tld.cache"
	extract, _ := tlde.New(cache, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extract.Extract(benchmarkURL)
	}
}

/*
// github.com/weppos/publicsuffix-go
func BenchmarkPublicSuffixGo(b *testing.B) {
	// this module cannot handle full URLs with scheme (i.e. https:// ftp:// etc.)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		publicsuffix.Parse(benchmarkURL)
	}
}
*/

/*
// github.com/forease/gotld
func BenchmarkGoTldForeEase(b *testing.B) {
	// does not extract subdomain properly, cannot handle ip addresses
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gotld.GetSubdomain(benchmarkURL, 2048)
	}
}
*/
