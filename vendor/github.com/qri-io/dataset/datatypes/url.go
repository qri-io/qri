package datatypes

import (
	"regexp"
)

var urlRegexp = regexp.MustCompile(`/((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.‌​-]+(:[0-9]+)?|(?:www‌​.|[-;:&=\+\$,\w]+@)[‌​A-Za-z0-9.-]+)((?:\/‌​[\+~%\/.\w-_]*)?\??(‌​?:[-\+=&;%@.\w_]*)#?‌​(?:[\w]*))?)/`)

// taken from: https://github.com/mvdan/xurls
const (
	otherScheme = `ftp|file|ssh`

	letter    = `\p{L}`
	mark      = `\p{M}`
	number    = `\p{N}`
	iriChar   = letter + mark + number
	currency  = `\p{Sc}`
	otherSymb = `\p{So}`
	endChar   = iriChar + `/\-+_&~*%=#` + currency + otherSymb
	midChar   = endChar + `@.,:;'?!|`
	wellParen = `\([` + midChar + `]*(\([` + midChar + `]*\)[` + midChar + `]*)*\)`
	wellBrack = `\[[` + midChar + `]*(\[[` + midChar + `]*\][` + midChar + `]*)*\]`
	wellBrace = `\{[` + midChar + `]*(\{[` + midChar + `]*\}[` + midChar + `]*)*\}`
	wellAll   = wellParen + `|` + wellBrack + `|` + wellBrace
	pathCont  = `([` + midChar + `]*(` + wellAll + `|[` + endChar + `])+)+`
	comScheme = `[a-zA-Z][a-zA-Z.\-+]*://`
	scheme    = `(` + comScheme + `|` + otherScheme + `)`

	iri      = `[` + iriChar + `]([` + iriChar + `\-]*[` + iriChar + `])?`
	domain   = `(` + iri + `\.)+`
	octet    = `(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9][0-9]|[0-9])`
	ipv4Addr = `\b` + octet + `\.` + octet + `\.` + octet + `\.` + octet + `\b`
	ipv6Addr = `([0-9a-fA-F]{1,4}:([0-9a-fA-F]{1,4}:([0-9a-fA-F]{1,4}:([0-9a-fA-F]{1,4}:([0-9a-fA-F]{1,4}:[0-9a-fA-F]{0,4}|:[0-9a-fA-F]{1,4})?|(:[0-9a-fA-F]{1,4}){0,2})|(:[0-9a-fA-F]{1,4}){0,3})|(:[0-9a-fA-F]{1,4}){0,4})|:(:[0-9a-fA-F]{1,4}){0,5})((:[0-9a-fA-F]{1,4}){2}|:(25[0-5]|(2[0-4]|1[0-9]|[1-9])?[0-9])(\.(25[0-5]|(2[0-4]|1[0-9]|[1-9])?[0-9])){3})|(([0-9a-fA-F]{1,4}:){1,6}|:):[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){7}:`
	ipAddr   = `(` + ipv4Addr + `|` + ipv6Addr + `)`
	// site     = domain + gtld
	site     = domain
	hostName = `(` + site + `|` + ipAddr + `)`
	port     = `(:[0-9]*)?`
	path     = `(/|/` + pathCont + `?|\b|$)`
	webURL   = hostName + port + path

	strict  = `(\b` + scheme + pathCont + `)`
	relaxed = `(` + strict + `|` + webURL + `)`
)

var Relaxed = regexp.MustCompile(relaxed)
