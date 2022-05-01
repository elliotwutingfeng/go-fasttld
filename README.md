# go-fasttld

[![Go Reference](https://img.shields.io/badge/go-reference-blue?logo=go&logoColor=white&style=for-the-badge)](https://pkg.go.dev/github.com/elliotwutingfeng/go-fasttld)
[![Go Report Card](https://goreportcard.com/badge/github.com/elliotwutingfeng/go-fasttld?style=for-the-badge)](https://goreportcard.com/report/github.com/elliotwutingfeng/go-fasttld)
[![Codecov Coverage](https://img.shields.io/codecov/c/github/elliotwutingfeng/go-fasttld?color=bright-green&logo=codecov&style=for-the-badge&token=GB00MYK51E)](https://codecov.io/gh/elliotwutingfeng/go-fasttld)
[![Mentioned in Awesome Go](https://img.shields.io/static/v1?logo=awesomelists&label=&labelColor=CCA6C4&logoColor=261120&message=Mentioned%20in%20awesome&color=494368&style=for-the-badge)](https://github.com/avelino/awesome-go)

[![GitHub license](https://img.shields.io/badge/LICENSE-BSD--3--CLAUSE-GREEN?style=for-the-badge)](LICENSE)

**go-fasttld** is a high performance [top level domains (TLD)](https://en.wikipedia.org/wiki/Top-level_domain) extraction module implemented with [compressed tries](https://en.wikipedia.org/wiki/Trie).

This module is a port of the Python [fasttld](https://github.com/jophy/fasttld) module, with additional modifications to support extraction of subcomponents from full URLs and IPv4 addresses.

![Trie](Trie_example.svg)

## Background

**go-fasttld** extracts subcomponents like [top level domains (TLDs)](https://en.wikipedia.org/wiki/Top-level_domain), subdomains and hostnames from [URLs](https://en.wikipedia.org/wiki/URL) efficiently by using the regularly-updated [Mozilla Public Suffix List](http://www.publicsuffix.org) and the [compressed trie](https://en.wikipedia.org/wiki/Trie) data structure.

For example, it extracts the `com` TLD, `maps` subdomain, and `google` domain from `https://maps.google.com:8080/a/long/path/?query=42`.

**go-fasttld** also supports extraction of private domains listed in the [Mozilla Public Suffix List](http://www.publicsuffix.org) like 'blogspot.co.uk' and 'sinaapp.com', and extraction of IPv4 addresses (e.g. `https://127.0.0.1`).

### Why not split on "." and take the last element instead?

Splitting on "." and taking the last element only works for simple TLDs like `.com`, but not more complex ones like `oseto.nagasaki.jp`.

### Compressed trie example

Valid TLDs from the [Mozilla Public Suffix List](http://www.publicsuffix.org) are appended to the compressed trie in reverse-order.

```sh
Given the following TLDs
au
nsw.edu.au
com.ac
edu.ac
gov.ac

and the example URI host `example.nsw.edu.au`

The compressed trie will be structured as follows:

START
 ╠═ au 🚩 ✅
 ║  ╚═ edu ✅
 ║     ╚═ nsw 🚩 ✅
 ╚═ ac
    ╠═ com 🚩
    ╠═ edu 🚩
    ╚═ gov 🚩

=== Symbol meanings ===
🚩 : path to this node is a valid TLD
✅ : path to this node found in example URI host `example.nsw.edu.au`
```

The URI host subcomponents are parsed from right-to-left until no more matching nodes can be found. In this example, the path of matching nodes are `au -> edu -> nsw`. Reversing the nodes gives the extracted TLD `nsw.edu.au`.

## Installation

```sh
go get github.com/elliotwutingfeng/go-fasttld
```

## Quick Start

Full demo available in the _examples_ folder

```go
// Initialise fasttld extractor
extractor, _ := fasttld.New(fasttld.SuffixListParams{})

//Extract URL subcomponents
url := "https://some-user@a.long.subdomain.ox.ac.uk:5000/a/b/c/d/e/f/g/h/i?id=42"
res := extractor.Extract(fasttld.URLParams{URL: url})

// Display results
fmt.Println(res.Scheme)           // https://
fmt.Println(res.UserInfo)         // some-user
fmt.Println(res.SubDomain)        // a.long.subdomain
fmt.Println(res.Domain)           // ox
fmt.Println(res.Suffix)           // ac.uk
fmt.Println(res.RegisteredDomain) // ox.ac.uk
fmt.Println(res.Port) // 5000
fmt.Println(res.Path) // a/b/c/d/e/f/g/h/i?id=42
```

## Public Suffix List options

### Specify custom public suffix list file

You can use a custom public suffix list file by setting `CacheFilePath` in `fasttld.SuffixListParams{}` to its absolute path.

```go
cacheFilePath := "/absolute/path/to/file.dat"
extractor, _ := fasttld.New(fasttld.SuffixListParams{CacheFilePath: cacheFilePath})
```

### Updating the default Public Suffix List cache

Whenever `fasttld.New` is called without specifying `CacheFilePath` in `fasttld.SuffixListParams{}`, the local cache of the default Public Suffix List is updated automatically if it is more than 3 days old. You can also manually update the cache by using `Update()`.

```go
// Automatic update performed if `CacheFilePath` is not specified
// and local cache is more than 3 days old
extractor, _ := fasttld.New(fasttld.SuffixListParams{})

// Manually update local cache
if err := extractor.Update(); err != nil {
    log.Println(err)
}
```

### Private domains

According to the [Mozilla.org wiki](https://wiki.mozilla.org/Public_Suffix_List/Uses), the Mozilla Public Suffix List contains private domains like `blogspot.com` and `sinaapp.com`.

By default, **go-fasttld** _excludes_ these private domains (i.e. `IncludePrivateSuffix = false`)

```go
extractor, _ := fasttld.New(fasttld.SuffixListParams{})

url := "https://google.blogspot.com"
res := extractor.Extract(fasttld.URLParams{URL: url})

// res.Scheme = https://
// res.UserInfo = <no output>
// res.SubDomain = google
// res.Domain = blogspot
// res.Suffix = com
// res.RegisteredDomain = blogspot.com
// res.Port = <no output>
// res.Path = <no output>
```

You can _include_ private domains by setting `IncludePrivateSuffix = true`

```go
extractor, _ := fasttld.New(fasttld.SuffixListParams{IncludePrivateSuffix: true})

url := "https://google.blogspot.com"
res := extractor.Extract(fasttld.URLParams{URL: url})

// res.Scheme = https://
// res.UserInfo = <no output>
// res.SubDomain = <no output>
// res.Domain = google
// res.Suffix = blogspot.com
// res.RegisteredDomain = google.blogspot.com
// res.Port = <no output>
// res.Path = <no output>
```

## Extraction options

### Ignore Subdomains

You can ignore subdomains by setting `IgnoreSubDomains = true`. By default, subdomains are extracted.

```go
extractor, _ := fasttld.New(fasttld.SuffixListParams{})

url := "https://maps.google.com"
res := extractor.Extract(fasttld.URLParams{URL: url, IgnoreSubDomains: true})

// res.Scheme = https://
// res.UserInfo = <no output>
// res.SubDomain = <no output>
// res.Domain = google
// res.Suffix = com
// res.RegisteredDomain = google.com
// res.Port = <no output>
// res.Path = <no output>
```

### Punycode

Convert internationalised URLs to [punycode](https://en.wikipedia.org/wiki/Punycode) before extraction by setting `ConvertURLToPunyCode = true`. By default, URLs are not converted to punycode.

```go
extractor, _ := fasttld.New(fasttld.SuffixListParams{})

url := "https://hello.世界.com"
res := extractor.Extract(fasttld.URLParams{URL: url, ConvertURLToPunyCode: true})

// res.Scheme = https://
// res.UserInfo = <no output>
// res.SubDomain = hello
// res.Domain = xn--rhqv96g
// res.Suffix = com
// res.RegisteredDomain = xn--rhqv96g.com
// res.Port = <no output>
// res.Path = <no output>

res = extractor.Extract(fasttld.URLParams{URL: url, ConvertURLToPunyCode: false})

// res.Scheme = https://
// res.UserInfo = <no output>
// res.SubDomain = hello
// res.Domain = 世界
// res.Suffix = com
// res.RegisteredDomain = 世界.com
// res.Port = <no output>
// res.Path = <no output>
```

## Testing

```sh
go test -v -coverprofile=test_coverage.out && go tool cover -html=test_coverage.out -o test_coverage.html
```

## Benchmarks

```sh
go test -bench=. -benchmem -cpu 1
```

### Modules used

| Benchmark Name                | Source                           |
|-------------------------------|----------------------------------|
| BenchmarkGoFastTld            | go-fasttld (this module)         |
| BenchmarkJPilloraGoTld        | github.com/jpillora/go-tld       |
| BenchmarkJoeGuoTldExtract     | github.com/joeguo/tldextract     |
| BenchmarkMjd2021USATldExtract | github.com/mjd2021usa/tldextract |
| BenchmarkM507Tlde             | github.com/M507/tlde             |

### Results

Benchmarks performed on AMD Ryzen 7 5800X, Manjaro Linux.

**go-fasttld** performs especially well on longer URLs.

---

#### #1

<code>https://news.google.com</code>

| Benchmark Name                | Iterations | ns/op       | B/op     | allocs/op   | Fastest            |
|-------------------------------|------------|-------------|----------|-------------|--------------------|
| BenchmarkGoFastTld            | 2519748    | 478.4 ns/op | 176 B/op | 4 allocs/op |                    |
| BenchmarkJPilloraGoTld        | 2695350    | 453.6 ns/op | 224 B/op | 2 allocs/op | :heavy_check_mark: |
| BenchmarkJoeGuoTldExtract     | 2485053    | 509.0 ns/op | 160 B/op | 5 allocs/op |                    |
| BenchmarkMjd2021USATldExtract | 1451647    | 825.2 ns/op | 208 B/op | 7 allocs/op |                    |
| BenchmarkM507Tlde             | 2396223    | 484.3 ns/op | 160 B/op | 5 allocs/op |                    |

---

#### #2

<code>https://iupac.org/iupac-announces-the-2021-top-ten-emerging-technologies-in-chemistry/</code>

| Benchmark Name                | Iterations | ns/op       | B/op     | allocs/op   | Fastest            |
|-------------------------------|------------|-------------|----------|-------------|--------------------|
| BenchmarkGoFastTld            | 2368650    | 505.6 ns/op | 304 B/op | 4 allocs/op | :heavy_check_mark: |
| BenchmarkJPilloraGoTld        | 1889172    | 634.5 ns/op | 224 B/op | 2 allocs/op |                    |
| BenchmarkJoeGuoTldExtract     | 2242525    | 524.4 ns/op | 272 B/op | 5 allocs/op |                    |
| BenchmarkMjd2021USATldExtract | 1525376    | 782.3 ns/op | 288 B/op | 6 allocs/op |                    |
| BenchmarkM507Tlde             | 2310541    | 518.6 ns/op | 272 B/op | 5 allocs/op |                    |

---

#### #3

<code>https://www.google.com/maps/dir/Parliament+Place,+Parliament+House+Of+Singapore,+Singapore/Parliament+St,+London,+UK/@25.2440033,33.6721455,4z/data=!3m1!4b1!4m14!4m13!1m5!1m1!1s0x31da19a0abd4d71d:0xeda26636dc4ea1dc!2m2!1d103.8504863!2d1.2891543!1m5!1m1!1s0x487604c5aaa7da5b:0xf13a2197d7e7dd26!2m2!1d-0.1260826!2d51.5017061!3e4</code>

| Benchmark Name                | Iterations | ns/op       | B/op      | allocs/op   | Fastest            |
|-------------------------------|------------|-------------|-----------|-------------|--------------------|
| BenchmarkGoFastTld            | 1725812    | 711.7 ns/op | 784 B/op  | 4 allocs/op | :heavy_check_mark: |
| BenchmarkJPilloraGoTld        | 447218     | 2532 ns/op  | 928 B/op  | 4 allocs/op |                    |
| BenchmarkJoeGuoTldExtract     | 836458     | 1337 ns/op  | 1120 B/op | 6 allocs/op |                    |
| BenchmarkMjd2021USATldExtract | 927424     | 1215 ns/op  | 1120 B/op | 6 allocs/op |                    |
| BenchmarkM507Tlde             | 880028     | 1253 ns/op  | 1120 B/op | 6 allocs/op |                    |

---

## Acknowledgements

- [fasttld (Python)](https://github.com/jophy/fasttld)
- [tldextract (Python)](https://github.com/john-kurkowski/tldextract)
- [tldextract (Go)](https://github.com/mjd2021usa/tldextract)
- [IETF RFC 2396](https://www.ietf.org/rfc/rfc2396.txt)
