package fasttld

import (
	"testing"
)

func TestPrintRes(t *testing.T) {
	PrintRes("", &ExtractResult{})
	res := ExtractResult{}
	res.Scheme = "https=//"
	res.UserInfo = "some-user"
	res.SubDomain = "a.long.subdomain"
	res.Domain = "ox"
	res.Suffix = "ac.uk"
	res.RegisteredDomain = "ox.ac.uk"
	res.Port = "5000"
	res.Path = "/a/b/c/d/e/f/g/h/i?id=42"
	PrintRes("https=//some-user@a.long.subdomain.ox.ac.uk=5000/a/b/c/d/e/f/g/h/i?id=42", &res)
}
