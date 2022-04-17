package main

import (
	"fmt"
	"log"

	"github.com/elliotwutingfeng/go-fasttld"
)

func printRes(url string, res *fasttld.ExtractResult) {
	fmt.Println("              url:", url)
	fmt.Println("        subdomain:", res.SubDomain)
	fmt.Println("           domain:", res.Domain)
	fmt.Println("           suffix:", res.Suffix)
	fmt.Println("registered domain:", res.RegisteredDomain)
	fmt.Println("             port:", res.Port)
	fmt.Println("             path:", res.Path)
	fmt.Println("")
}

func main() {
	url := "https://a.long.subdomain.ox.ac.uk:5000/a/b/c/d/e/f/g/h/i?id=42"

	extractor, err := fasttld.New(fasttld.SuffixListParams{})
	if err != nil {
		log.Fatal(err)
	}
	res := extractor.Extract(fasttld.UrlParams{Url: url})
	fmt.Println("Simple Example")
	printRes(url, res)
	// res.SubDomain = a.long.subdomain
	// res.Domain = ox
	// res.Suffix = ac.uk
	// res.RegisteredDomain = ox.ac.uk
	// res.Port = 5000
	// res.Path = a/b/c/d/e/f/g/h/i?id=42

	// Specify custom public suffix list file
	// cacheFilePath := "/absolute/path/to/file.dat"
	// extractor, _ = fasttld.New(fasttld.SuffixListParams{CacheFilePath: cacheFilePath})

	// Manually update local cache
	showLogMessages := false
	if err := extractor.Update(showLogMessages); err != nil {
		log.Println(err)
	}

	// Private domains
	url = "https://google.blogspot.com"

	extractor, _ = fasttld.New(fasttld.SuffixListParams{})
	res = extractor.Extract(fasttld.UrlParams{Url: url})
	fmt.Println("Exclude Private Domains")
	printRes(url, res)
	// res.SubDomain = google
	// res.Domain = blogspot
	// res.Suffix = com
	// res.RegisteredDomain = blogspot.com
	// res.Port = <no output>
	// res.Path = <no output>

	extractor, _ = fasttld.New(fasttld.SuffixListParams{IncludePrivateSuffix: true})
	res = extractor.Extract(fasttld.UrlParams{Url: url})
	fmt.Println("Include Private Domains")
	printRes(url, res)
	// res.SubDomain = <no output>
	// res.Domain = google
	// res.Suffix = blogspot.com
	// res.RegisteredDomain = google.blogspot.com
	// res.Port = <no output>
	// res.Path = <no output>

	// Ignore Subdomains
	url = "https://maps.google.com"

	extractor, _ = fasttld.New(fasttld.SuffixListParams{})
	res = extractor.Extract(fasttld.UrlParams{Url: url, IgnoreSubDomains: true})
	fmt.Println("Ignore Subdomains")
	printRes(url, res)
	// res.SubDomain = <no output>
	// res.Domain = google
	// res.Suffix = com
	// res.RegisteredDomain = google.com
	// res.Port = <no output>
	// res.Path = <no output>

	// Punycode
	url = "https://hello.世界.com"

	res = extractor.Extract(fasttld.UrlParams{Url: url, ConvertURLToPunyCode: true})
	fmt.Println("Punycode")
	printRes(url, res)
	// res.SubDomain = hello
	// res.Domain = xn--rhqv96g
	// res.Suffix = com
	// res.RegisteredDomain = xn--rhqv96g.com
	// res.Port = <no output>
	// res.Path = <no output>

	res = extractor.Extract(fasttld.UrlParams{Url: url, ConvertURLToPunyCode: false})
	fmt.Println("No Punycode")
	printRes(url, res)
	// res.SubDomain = hello
	// res.Domain = 世界
	// res.Suffix = com
	// res.RegisteredDomain = 世界.com
	// res.Port = <no output>
	// res.Path = <no output>
}
