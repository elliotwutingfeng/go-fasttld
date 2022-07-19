package fasttld

import (
	"testing"
)

func TestPrintRes(t *testing.T) {
	PrintRes("", &ExtractResult{})
	res := ExtractResult{}
	res.Scheme = "https://"
	res.UserInfo = "user"
	res.SubDomain = "a.subdomain"
	res.Domain = "example"
	res.Suffix = "ac.uk"
	res.RegisteredDomain = "example.ac.uk"
	res.Port = "5000"
	res.Path = "/a/b?id=42"
	res.HostType = HostName
	PrintRes("https://user@a.subdomain.example.ac.uk:5000/a/b?id=42", &res)
	res = ExtractResult{}
	res.HostType = IPv4
	PrintRes("1.1.1.1", &res)
	res.HostType = IPv6
	PrintRes("[aBcD:ef01:2345:6789:aBcD:ef01:2345:6789]", &res)
}
