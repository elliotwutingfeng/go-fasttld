package main

import (
	"fmt"
	"log"

	"github.com/elliotwutingfeng/go-fasttld"
)

func printRes(url string, res *fasttld.ExtractResult) {
	fmt.Println("              url:", url)
	fmt.Println("           scheme:", res.Scheme)
	fmt.Println("         userinfo:", res.UserInfo)
	fmt.Println("        subdomain:", res.SubDomain)
	fmt.Println("           domain:", res.Domain)
	fmt.Println("           suffix:", res.Suffix)
	fmt.Println("registered domain:", res.RegisteredDomain)
	fmt.Println("             port:", res.Port)
	fmt.Println("             path:", res.Path)
	fmt.Println("")
}

func main() {
	url := "https://some-user@a.long.subdomain.ox.ac.uk:5000/a/b/c/d/e/f/g/h/i?id=42"

	extractor, err := fasttld.New(fasttld.SuffixListParams{})
	if err != nil {
		log.Fatal(err)
	}
	res := extractor.Extract(fasttld.URLParams{URL: url})
	fmt.Println("Domain")
	printRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = some-user
	// res.SubDomain = a.long.subdomain
	// res.Domain = ox
	// res.Suffix = ac.uk
	// res.RegisteredDomain = ox.ac.uk
	// res.Port = 5000
	// res.Path = /a/b/c/d/e/f/g/h/i?id=42

	// Specify custom public suffix list file
	// cacheFilePath := "/absolute/path/to/file.dat"
	// extractor, _ = fasttld.New(fasttld.SuffixListParams{CacheFilePath: cacheFilePath})

	// IPv4 Address
	url = "https://127.0.0.1:5000"

	res = extractor.Extract(fasttld.URLParams{URL: url})
	fmt.Println("IPv4 Address")
	printRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = <no output>
	// res.Domain = 127.0.0.1
	// res.Suffix = <no output>
	// res.RegisteredDomain = 127.0.0.1
	// res.Port = 5000
	// res.Path = <no output>

	// IPv6 Address
	url = "https://[aBcD:ef01:2345:6789:aBcD:ef01:2345:6789]:5000"

	res = extractor.Extract(fasttld.URLParams{URL: url})
	fmt.Println("IPv6 Address")
	printRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = <no output>
	// res.Domain = aBcD:ef01:2345:6789:aBcD:ef01:2345:6789
	// res.Suffix = <no output>
	// res.RegisteredDomain = aBcD:ef01:2345:6789:aBcD:ef01:2345:6789
	// res.Port = 5000
	// res.Path = <no output>

	// Internationalised label separators
	url = "https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk"

	res = extractor.Extract(fasttld.URLParams{URL: url})
	fmt.Println("Internationalised label separators")
	printRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = brb\u002ei\u3002am\uff0egoing\uff61to
	// res.Domain = be
	// res.Suffix = a\uff61fk
	// res.RegisteredDomain = be\u3002a\uff61fk
	// res.Port = <no output>
	// res.Path = <no output>

	// Manually update local cache
	if err := extractor.Update(); err != nil {
		log.Println(err)
	}

	// Private domains
	url = "https://google.blogspot.com"

	extractor, _ = fasttld.New(fasttld.SuffixListParams{})
	res = extractor.Extract(fasttld.URLParams{URL: url})
	fmt.Println("Exclude Private Domains")
	printRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = google
	// res.Domain = blogspot
	// res.Suffix = com
	// res.RegisteredDomain = blogspot.com
	// res.Port = <no output>
	// res.Path = <no output>

	extractor, _ = fasttld.New(fasttld.SuffixListParams{IncludePrivateSuffix: true})
	res = extractor.Extract(fasttld.URLParams{URL: url})
	fmt.Println("Include Private Domains")
	printRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = <no output>
	// res.Domain = google
	// res.Suffix = blogspot.com
	// res.RegisteredDomain = google.blogspot.com
	// res.Port = <no output>
	// res.Path = <no output>

	// Ignore Subdomains
	url = "https://maps.google.com"

	extractor, _ = fasttld.New(fasttld.SuffixListParams{})
	res = extractor.Extract(fasttld.URLParams{URL: url, IgnoreSubDomains: true})
	fmt.Println("Ignore Subdomains")
	printRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = <no output>
	// res.Domain = google
	// res.Suffix = com
	// res.RegisteredDomain = google.com
	// res.Port = <no output>
	// res.Path = <no output>

	// Punycode
	url = "https://hello.世界.com"

	res = extractor.Extract(fasttld.URLParams{URL: url, ConvertURLToPunyCode: true})
	fmt.Println("Punycode")
	printRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = hello
	// res.Domain = xn--rhqv96g
	// res.Suffix = com
	// res.RegisteredDomain = xn--rhqv96g.com
	// res.Port = <no output>
	// res.Path = <no output>

	res = extractor.Extract(fasttld.URLParams{URL: url, ConvertURLToPunyCode: false})
	fmt.Println("No Punycode")
	printRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = hello
	// res.Domain = 世界
	// res.Suffix = com
	// res.RegisteredDomain = 世界.com
	// res.Port = <no output>
	// res.Path = <no output>
}
