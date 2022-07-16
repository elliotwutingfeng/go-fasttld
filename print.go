package fasttld

import (
	"github.com/fatih/color"
)

// PrintRes pretty-prints URL components from ExtractResult
func PrintRes(url string, res *ExtractResult) {
	var leftAttrsFilled = []color.Attribute{color.FgHiYellow, color.Bold}
	var leftAttrsBlank = []color.Attribute{color.FgHiBlack}
	var rightAttrs = []color.Attribute{color.FgHiWhite}

	if len(url) != 0 {
		color.New(leftAttrsFilled...).Print("              url: ")
	} else {
		color.New(leftAttrsBlank...).Print("              url: ")
	}
	color.New(rightAttrs...).Println(url)

	if len(res.Scheme) != 0 {
		color.New(leftAttrsFilled...).Print("           scheme: ")
	} else {
		color.New(leftAttrsBlank...).Print("           scheme: ")
	}
	color.New(rightAttrs...).Println(res.Scheme)

	if len(res.UserInfo) != 0 {
		color.New(leftAttrsFilled...).Print("         userinfo: ")
	} else {
		color.New(leftAttrsBlank...).Print("         userinfo: ")
	}
	color.New(rightAttrs...).Println(res.UserInfo)

	if len(res.SubDomain) != 0 {
		color.New(leftAttrsFilled...).Print("        subdomain: ")
	} else {
		color.New(leftAttrsBlank...).Print("        subdomain: ")
	}
	color.New(rightAttrs...).Println(res.SubDomain)

	if len(res.Domain) != 0 {
		color.New(leftAttrsFilled...).Print("           domain: ")
	} else {
		color.New(leftAttrsBlank...).Print("           domain: ")
	}
	color.New(rightAttrs...).Println(res.Domain)

	if len(res.Suffix) != 0 {
		color.New(leftAttrsFilled...).Print("           suffix: ")
	} else {
		color.New(leftAttrsBlank...).Print("           suffix: ")
	}
	color.New(rightAttrs...).Println(res.Suffix)

	if len(res.RegisteredDomain) != 0 {
		color.New(leftAttrsFilled...).Print("registered domain: ")
	} else {
		color.New(leftAttrsBlank...).Print("registered domain: ")
	}
	color.New(rightAttrs...).Println(res.RegisteredDomain)

	if len(res.Port) != 0 {
		color.New(leftAttrsFilled...).Print("             port: ")
	} else {
		color.New(leftAttrsBlank...).Print("             port: ")
	}
	color.New(rightAttrs...).Println(res.Port)

	if len(res.Path) != 0 {
		color.New(leftAttrsFilled...).Print("             path: ")
	} else {
		color.New(leftAttrsBlank...).Print("             path: ")
	}
	color.New(rightAttrs...).Println(res.Path)

	color.New().Println("")
}
