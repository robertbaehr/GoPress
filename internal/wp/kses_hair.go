package wp

import (
	"html"
	"strings"
)

// KsesHairAttr is one attribute from wp_kses_hair (P2.127).
type KsesHairAttr struct {
	Name  string // attribute name
	Value string // normalized value (empty when valueless)
	Whole string // reconstructed attribute HTML fragment
	Vless string // "y" when boolean/valueless, else "n"
}

// KsesHair parses an HTML attribute list into a name→attr map (wp_kses_hair).
// URI attributes are passed through kses_bad_protocol using protocols
// (nil/empty → default allowedProtocols). Values are syntax-escaped like PHP's
// strtr of &, <, >, ', ".
func KsesHair(attr string, protocols []string) map[string]KsesHairAttr {
	out := map[string]KsesHairAttr{}
	parsed := ksesParseAttrs(attr)
	if len(parsed) == 0 {
		return out
	}
	protoSet := map[string]bool{}
	if len(protocols) == 0 {
		for p := range allowedProtocols {
			protoSet[p] = true
		}
	} else {
		for _, p := range protocols {
			protoSet[strings.ToLower(p)] = true
		}
	}
	for _, a := range parsed {
		name := a.name
		isBool := a.vless
		val := a.value
		if !isBool && ksesURIAttr[strings.ToLower(name)] {
			val = ksesBadProtocolWith(html.UnescapeString(val), protoSet)
		}
		recoded := ""
		whole := name
		vless := "y"
		if !isBool {
			recoded = ksesHairRecodeValue(val)
			whole = name + `="` + recoded + `"`
			vless = "n"
		}
		out[name] = KsesHairAttr{
			Name:  name,
			Value: recoded,
			Whole: whole,
			Vless: vless,
		}
	}
	return out
}

// ksesHairRecodeValue mirrors the syntax_characters strtr in wp_kses_hair.
func ksesHairRecodeValue(v string) string {
	replacer := strings.NewReplacer(
		`&`, `&amp;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
		`'`, `&apos;`,
		`"`, `&quot;`,
	)
	return replacer.Replace(v)
}
