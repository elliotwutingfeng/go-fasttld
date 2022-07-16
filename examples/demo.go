package main

import (
	"log"

	"github.com/elliotwutingfeng/go-fasttld"
	"github.com/fatih/color"
)

func main() {
	url := "https://user@a.subdomain.example.ac.uk:5000/a/b?id=42"

	extractor, err := fasttld.New(fasttld.SuffixListParams{})
	if err != nil {
		log.Fatal(err)
	}
	res, _ := extractor.Extract(fasttld.URLParams{URL: url})

	var fontStyle = []color.Attribute{color.FgHiWhite, color.Bold}

	color.New(fontStyle...).Println("Domain")
	fasttld.PrintRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = user
	// res.SubDomain = a.subdomain
	// res.Domain = example
	// res.Suffix = ac.uk
	// res.RegisteredDomain = example.ac.uk
	// res.Port = 5000
	// res.Path = /a/b?id=42

	// Specify custom public suffix list file
	// cacheFilePath := "/absolute/path/to/file.dat"
	// extractor, _ = fasttld.New(fasttld.SuffixListParams{CacheFilePath: cacheFilePath})

	// IPv4 Address
	url = "https://127.0.0.1:5000"

	res, _ = extractor.Extract(fasttld.URLParams{URL: url})
	color.New(fontStyle...).Println("IPv4 Address")
	fasttld.PrintRes(url, res)
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

	res, _ = extractor.Extract(fasttld.URLParams{URL: url})
	color.New(fontStyle...).Println("IPv6 Address")
	fasttld.PrintRes(url, res)
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

	res, _ = extractor.Extract(fasttld.URLParams{URL: url})
	color.New(fontStyle...).Println("Internationalised label separators")
	fasttld.PrintRes(url, res)
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
	res, _ = extractor.Extract(fasttld.URLParams{URL: url})
	color.New(fontStyle...).Println("Exclude Private Domains")
	fasttld.PrintRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = google
	// res.Domain = blogspot
	// res.Suffix = com
	// res.RegisteredDomain = blogspot.com
	// res.Port = <no output>
	// res.Path = <no output>

	extractor, _ = fasttld.New(fasttld.SuffixListParams{IncludePrivateSuffix: true})
	res, _ = extractor.Extract(fasttld.URLParams{URL: url})
	color.New(fontStyle...).Println("Include Private Domains")
	fasttld.PrintRes(url, res)
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
	res, _ = extractor.Extract(fasttld.URLParams{URL: url, IgnoreSubDomains: true})
	color.New(fontStyle...).Println("Ignore Subdomains")
	fasttld.PrintRes(url, res)
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

	res, _ = extractor.Extract(fasttld.URLParams{URL: url, ConvertURLToPunyCode: true})
	color.New(fontStyle...).Println("Punycode")
	fasttld.PrintRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = hello
	// res.Domain = xn--rhqv96g
	// res.Suffix = com
	// res.RegisteredDomain = xn--rhqv96g.com
	// res.Port = <no output>
	// res.Path = <no output>

	res, _ = extractor.Extract(fasttld.URLParams{URL: url, ConvertURLToPunyCode: false})
	color.New(fontStyle...).Println("No Punycode")
	fasttld.PrintRes(url, res)
	// res.Scheme = https://
	// res.UserInfo = <no output>
	// res.SubDomain = hello
	// res.Domain = 世界
	// res.Suffix = com
	// res.RegisteredDomain = 世界.com
	// res.Port = <no output>
	// res.Path = <no output>
}
