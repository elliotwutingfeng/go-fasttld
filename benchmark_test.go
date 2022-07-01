package fasttld

import (
	"fmt"
	"testing"

	tlde "github.com/M507/tlde/src"
	joeguotldextract "github.com/joeguo/tldextract"
	tld "github.com/jpillora/go-tld"
	mjd2021usatldextract "github.com/mjd2021usa/tldextract"
)

func BenchmarkComparison(b *testing.B) {
	var benchmarkURLs = []string{
		"https://news.google.com",
		"https://iupac.org/iupac-announces-the-2021-top-ten-emerging-technologies-in-chemistry/",
		"https://www.google.com/maps/dir/Parliament+Place,+Parliament+House+Of+Singapore,+" +
			"Singapore/Parliament+St,+London,+UK/@25.2440033,33.6721455,4z/data=!3m1!4b1!4m14!4m13!1m5!1m1!1s0x31d" +
			"a19a0abd4d71d:0xeda26636dc4ea1dc!2m2!1d103.8504863!2d1.2891543!1m5!1m1!1s0x487604c5aaa7da5b:0xf13a2" +
			"197d7e7dd26!2m2!1d-0.1260826!2d51.5017061!3e4",
	}

	benchmarks := []struct {
		name string
	}{
		{"GoFastTld"}, // this module
		//{"JPilloraGoTld"},        // github.com/jpillora/go-tld
		//{"JoeGuoTldExtract"},     // github.com/joeguo/tldextract
		//{"Mjd2021USATldExtract"}, // github.com/mjd2021usa/tldextract
		//{"M507Tlde"},             // github.com/M507/tlde
		// {"ImVexedFastURL"},       // github.com/ImVexed/fasturl
		// {"WepposPublicSuffixGo"}, // github.com/weppos/publicsuffix-go
		// {"ForeEaseGoTld"},        // github.com/forease/gotld
	}

	GoFastTld, _ := New(SuffixListParams{
		CacheFilePath:        getTestPSLFilePath(),
		IncludePrivateSuffix: false,
	})

	cache := "/tmp/tld.cache"

	JoeGuoTldExtract, _ := joeguotldextract.New(cache, false)

	Mjd2021USATldExtract, _ := mjd2021usatldextract.New(cache, false)

	M507Tlde, _ := tlde.New(cache, false)

	for _, benchmarkURL := range benchmarkURLs {
		for _, bm := range benchmarks {
			if bm.name == "GoFastTld" {
				// this module

				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						GoFastTld.Extract(URLParams{URL: benchmarkURL})
					}
				})

			} else if bm.name == "JPilloraGoTld" {
				// this module also provides the PORT and PATH subcomponents
				// it cannot handle "+://google.com" and IP addresses
				// it cannot handle urls without scheme component
				// it cannot handle trailing whitespace

				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						tld.Parse(benchmarkURL)
					}
				})

			} else if bm.name == "JoeGuoTldExtract" {

				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						JoeGuoTldExtract.Extract(benchmarkURL)
					}
				})

			} else if bm.name == "Mjd2021USATldExtract" {

				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						Mjd2021USATldExtract.Extract(benchmarkURL)
					}
				})

			} else if bm.name == "M507Tlde" {
				// Appears to be the same as github.com/joeguo/tldextract

				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						M507Tlde.Extract(benchmarkURL)
					}
				})

			} /* else if bm.name == "ImVexedFastURL" {
				// Uses the Ragel state-machine
				// this module cannot extract TLDs

				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						fasturl.ParseURL(benchmarkURL)
					}
				})

			}  else if bm.name == "WepposPublicSuffixGo" {
				// this module cannot handle full URLs with scheme (i.e. https:// ftp:// etc.)

				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						publicsuffix.Parse(benchmarkURL)
					}
				})

			} else if bm.name == "ForeEaseGoTld" {
				// this module does not extract subdomain properly and cannot handle ip addresses

				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						gotld.GetSubdomain(benchmarkURL, 2048)
					}
				})

			} */
		}
		fmt.Println()
		fmt.Println("Benchmarks completed for URL :", benchmarkURL)
		fmt.Println("=======")
	}
}
