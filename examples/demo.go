package main

import (
	"log"

	"github.com/elliotwutingfeng/go-fasttld"
	"github.com/fatih/color"
)

func main() {
	var fontStyle = []color.Attribute{color.FgHiWhite, color.Bold}

	// Hostname
	url := "https://user@a.subdomain.example.ac.uk:5000/a/b?id=42"

	extractor, err := fasttld.New(fasttld.SuffixListParams{})
	if err != nil {
		log.Fatal(err)
	}
	res, _ := extractor.Extract(fasttld.URLParams{URL: url})
	color.New(fontStyle...).Println("Hostname")
	fasttld.PrintRes(url, res)

	// Specify custom public suffix list file
	// cacheFilePath := "/absolute/path/to/file.dat"
	// extractor, _ = fasttld.New(fasttld.SuffixListParams{CacheFilePath: cacheFilePath})

	// IPv4 Address
	url = "https://127.0.0.1:5000"

	res, _ = extractor.Extract(fasttld.URLParams{URL: url})
	color.New(fontStyle...).Println("IPv4 Address")
	fasttld.PrintRes(url, res)

	// IPv6 Address
	url = "https://[aBcD:ef01:2345:6789:aBcD:ef01:2345:6789]:5000"

	res, _ = extractor.Extract(fasttld.URLParams{URL: url})
	color.New(fontStyle...).Println("IPv6 Address")
	fasttld.PrintRes(url, res)

	// Internationalised label separators
	url = "https://brb\u002ei\u3002am\uff0egoing\uff61to\uff0ebe\u3002a\uff61fk"

	res, _ = extractor.Extract(fasttld.URLParams{URL: url})
	color.New(fontStyle...).Println("Internationalised label separators")
	fasttld.PrintRes(url, res)

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

	extractor, _ = fasttld.New(fasttld.SuffixListParams{IncludePrivateSuffix: true})
	res, _ = extractor.Extract(fasttld.URLParams{URL: url})
	color.New(fontStyle...).Println("Include Private Domains")
	fasttld.PrintRes(url, res)

	// Ignore Subdomains
	url = "https://maps.google.com"

	extractor, _ = fasttld.New(fasttld.SuffixListParams{})
	res, _ = extractor.Extract(fasttld.URLParams{URL: url, IgnoreSubDomains: true})
	color.New(fontStyle...).Println("Ignore Subdomains")
	fasttld.PrintRes(url, res)

	// Punycode
	url = "https://hello.世界.com"

	res, _ = extractor.Extract(fasttld.URLParams{URL: url, ConvertURLToPunyCode: true})
	color.New(fontStyle...).Println("Punycode")
	fasttld.PrintRes(url, res)

	res, _ = extractor.Extract(fasttld.URLParams{URL: url, ConvertURLToPunyCode: false})
	color.New(fontStyle...).Println("No Punycode")
	fasttld.PrintRes(url, res)

	// Parsing errors
	url = "https://example!.com" // invalid characters in hostname

	color.New(fontStyle...).Println("Parsing errors")
	color.New().Println("The following line should be an error message")
	if res, err = extractor.Extract(fasttld.URLParams{URL: url}); err != nil {
		color.New(color.FgHiRed, color.Bold).Print("Error: ")
		color.New(color.FgHiWhite).Println(err)
	}
	fasttld.PrintRes(url, res) // Partially extracted subcomponents can still be retrieved
}
