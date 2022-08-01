package fasttld

import (
	"fmt"
	"testing"

	"github.com/fatih/color"
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
		{"GoFastTld"},            // this module
		{"JPilloraGoTld"},        // github.com/jpillora/go-tld
		{"JoeGuoTldExtract"},     // github.com/joeguo/tldextract
		{"Mjd2021USATldExtract"}, // github.com/mjd2021usa/tldextract
	}

	cache := "/tmp/tld.cache"

	for _, benchmarkURL := range benchmarkURLs {
		for _, bm := range benchmarks {
			if bm.name == "GoFastTld" {
				GoFastTld, _ := New(SuffixListParams{
					CacheFilePath:        getTestPSLFilePath(),
					IncludePrivateSuffix: false,
				})
				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						GoFastTld.Extract(URLParams{URL: benchmarkURL})
					}
				})
			} else if bm.name == "JPilloraGoTld" {
				// Provides the Port and Path subcomponents
				// Cannot handle "+://google.com" and IP addresses
				// Cannot handle urls without Scheme subcomponent
				// Cannot handle trailing whitespace
				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						tld.Parse(benchmarkURL)
					}
				})
			} else if bm.name == "JoeGuoTldExtract" {
				JoeGuoTldExtract, _ := joeguotldextract.New(cache, false)
				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						JoeGuoTldExtract.Extract(benchmarkURL)
					}
				})

			} else if bm.name == "Mjd2021USATldExtract" {
				Mjd2021USATldExtract, _ := mjd2021usatldextract.New(cache, false)
				b.Run(fmt.Sprint(bm.name), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						Mjd2021USATldExtract.Extract(benchmarkURL)
					}
				})
			}
		}
		color.New().Println()
		color.New(color.FgHiGreen, color.Bold).Print("Benchmarks completed for URL : ")
		color.New(color.FgHiBlue).Println(benchmarkURL)
		color.New(color.FgHiWhite).Println("=======")
	}
}

/*

Omitted modules

github.com/M507/tlde | Almost exactly the same as github.com/joeguo/tldextract

github.com/ImVexed/fasturl | Fast, but cannot extract TLDs

github.com/weppos/publicsuffix-go | Cannot handle full URLs with scheme (i.e. https:// ftp:// etc.)

github.com/forease/gotld | Does not extract subdomain properly and cannot handle ip addresses

*/
