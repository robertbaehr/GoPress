package wp

import (
	"html"
	"regexp"
	"strings"
)

var (
	reKsesOneAttrLead  = regexp.MustCompile(`^\s*`)
	reKsesOneAttrTrail = regexp.MustCompile(`\s*$`)
	reKsesOneAttrSplit = regexp.MustCompile(`\s*=\s*`)
)

// KsesOneAttr filters one HTML attribute (wp_kses_one_attr, P2.129).
// Preserves leading/trailing whitespace; rejects disallowed attributes as "".
func KsesOneAttr(attr, element string) string {
	loadKsesAllowed()
	attr = ksesNoNullKeepSlash(attr)

	lead := reKsesOneAttrLead.FindString(attr)
	trail := reKsesOneAttrTrail.FindString(attr)
	if trail == "" {
		attr = attr[len(lead):]
	} else {
		attr = attr[len(lead) : len(attr)-len(trail)]
	}

	parts := reKsesOneAttrSplit.Split(attr, 2)
	name := parts[0]
	value := ""
	vless := "y"
	whole := name
	if len(parts) == 2 {
		value = parts[1]
		quote := ""
		if value != "" {
			quote = string(value[0])
		}
		if quote == `"` || quote == `'` {
			if !strings.HasSuffix(value, quote) {
				return ""
			}
			value = value[1 : len(value)-1]
		} else {
			quote = `"`
		}
		value = EscAttr(value)
		if ksesURIAttr[strings.ToLower(name)] {
			value = ksesBadProtocol(html.UnescapeString(value))
			value = EscAttr(value)
		}
		whole = name + "=" + quote + value + quote
		vless = "n"
		attr = whole
	}

	allow := resolveKsesAttrAllowed("post")
	ok, filtered := KsesAttrCheck(name, value, whole, vless, element, allow)
	if !ok {
		attr = ""
	} else if filtered != "" {
		attr = filtered
	}
	return lead + attr + trail
}

// ksesNoNullKeepSlash mirrors wp_kses_no_null(…, array('slash_zero'=>'keep')).
func ksesNoNullKeepSlash(content string) string {
	// Keep literal "\0" sequences; strip other NUL-like control chars like ksesNoNull.
	var b strings.Builder
	b.Grow(len(content))
	for i := 0; i < len(content); i++ {
		c := content[i]
		if c == 0 {
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
