package wp

import (
	"html"
	"regexp"
	"strings"
)

var reKsesXHTMLSlash = regexp.MustCompile(`(?s)\s*/\s*$`)

// KsesAttr rebuilds an opening HTML tag with only allowed attributes
// (wp_kses_attr, P2.128). allowed may be an AllowedHTML map or a context
// name resolved via KsesAllowedHTML (nil context → post).
func KsesAttr(element, attr string, allowed any, protocols []string) string {
	loadKsesAllowed()
	allowMap := resolveKsesAttrAllowed(allowed)
	element = strings.TrimSpace(element)
	if element == "" {
		return ""
	}

	xhtmlSlash := ""
	if reKsesXHTMLSlash.MatchString(attr) {
		xhtmlSlash = " /"
	}

	elementLow := strings.ToLower(element)
	rules, ok := allowMap[elementLow]
	if !ok || rules == nil {
		// Tag not listed, or listed with true/empty meaning no attributes.
		return "<" + element + xhtmlSlash + ">"
	}

	parsed := KsesHair(attr, protocols)
	required := map[string]bool{}
	for name, limits := range rules {
		if lim, ok := limits.(map[string]any); ok {
			if req, ok := lim["required"].(bool); ok && req {
				required[strings.ToLower(name)] = true
			}
		}
	}

	stripped := ""
	if xhtmlSlash == "" {
		stripped = "<" + element + ">"
	}

	var attr2 strings.Builder
	for _, a := range parsed {
		nameLow := strings.ToLower(a.Name)
		okAttr, whole := KsesAttrCheck(a.Name, a.Value, a.Whole, a.Vless, element, allowMap)
		if okAttr {
			attr2.WriteByte(' ')
			attr2.WriteString(whole)
			delete(required, nameLow)
			continue
		}
		if required[nameLow] {
			return stripped
		}
	}
	if len(required) > 0 {
		return stripped
	}

	cleaned := strings.Map(func(r rune) rune {
		if r == '<' || r == '>' {
			return -1
		}
		return r
	}, attr2.String())

	return "<" + element + cleaned + xhtmlSlash + ">"
}

// KsesAttrCheck reports whether an attribute is allowed (wp_kses_attr_check, P2.128).
// On rejection it returns false and empty whole; on success whole is the
// (possibly style-filtered) attribute fragment.
func KsesAttrCheck(name, value, whole, vless, element string, allowed AllowedHTML) (bool, string) {
	loadKsesAllowed()
	if allowed == nil {
		return false, ""
	}
	nameLow := strings.ToLower(name)
	elementLow := strings.ToLower(element)
	rules, ok := allowed[elementLow]
	if !ok || rules == nil {
		return false, ""
	}

	limits, has := rules[nameLow]
	if !has || limits == nil || limits == "" {
		// data-* wildcard
		if strings.HasPrefix(nameLow, "data-") {
			if _, ok := rules["data-*"]; ok && ksesDataAttrNameOK(nameLow) {
				has = true
				limits = rules["data-*"]
			}
		}
		if !has {
			return false, ""
		}
	}

	if nameLow == "style" {
		decoded := html.UnescapeString(value)
		filtered := safecssFilterAttr(decoded)
		if filtered == "" {
			return false, ""
		}
		if filtered != decoded {
			encoded := ksesHairRecodeValue(filtered)
			whole = name + `="` + encoded + `"`
			value = encoded
		}
	}

	if lim, ok := limits.(map[string]any); ok {
		for key, curr := range lim {
			if key == "required" {
				continue
			}
			if !ksesCheckAttrVal(value, vless, key, curr) {
				return false, ""
			}
		}
	}
	return true, whole
}

func resolveKsesAttrAllowed(allowed any) AllowedHTML {
	switch t := allowed.(type) {
	case AllowedHTML:
		return t
	case map[string]AttrRule:
		return AllowedHTML(t)
	case string:
		v := KsesAllowedHTML(nil, t)
		if m, ok := v.(AllowedHTML); ok {
			return m
		}
	case nil:
		v := KsesAllowedHTML(nil, "post")
		if m, ok := v.(AllowedHTML); ok {
			return m
		}
	}
	return AllowedHTML{}
}

func ksesDataAttrNameOK(name string) bool {
	if !strings.HasPrefix(name, "data-") {
		return false
	}
	for _, r := range name[5:] {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

// ksesCheckAttrVal mirrors wp_kses_check_attr_val for common constraint keys.
func ksesCheckAttrVal(value, vless, key string, curr any) bool {
	switch key {
	case "maxlen":
		n, ok := asIntish(curr)
		return ok && len(value) <= n
	case "maxval":
		n, ok := asIntish(curr)
		if !ok {
			return false
		}
		iv, ok := asIntish(value)
		return ok && iv <= n
	case "minlen":
		n, ok := asIntish(curr)
		return ok && len(value) >= n
	case "minval":
		n, ok := asIntish(curr)
		if !ok {
			return false
		}
		iv, ok := asIntish(value)
		return ok && iv >= n
	case "valueless":
		want, _ := curr.(string)
		return vless == want
	default:
		return true
	}
}

func asIntish(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case string:
		n := 0
		for _, r := range t {
			if r < '0' || r > '9' {
				return 0, false
			}
			n = n*10 + int(r-'0')
		}
		return n, t != ""
	default:
		return 0, false
	}
}
