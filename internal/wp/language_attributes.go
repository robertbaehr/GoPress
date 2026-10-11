package wp

import (
	"html"
	"strings"
)

// IsRTL reports whether the current locale is right-to-left (is_rtl stub, P3.120).
// Without a full WP_Locale, known RTL locale prefixes return true.
func IsRTL(c *Context) bool {
	code := ""
	if c != nil && c.Locale != nil {
		code = strings.ToLower(strings.TrimSpace(c.Locale.Code))
	}
	if code == "" {
		return false
	}
	// Common WordPress RTL locales (subset).
	switch {
	case strings.HasPrefix(code, "ar"),
		strings.HasPrefix(code, "he"),
		strings.HasPrefix(code, "fa"),
		strings.HasPrefix(code, "ur"),
		code == "ckb",
		strings.HasPrefix(code, "ps"),
		strings.HasPrefix(code, "yi"):
		return true
	default:
		return false
	}
}

// GetLanguageAttributes builds language attributes for the html tag
// (get_language_attributes, P3.120). doctype is "html" (default) or "xhtml".
func GetLanguageAttributes(c *Context, doctype string) string {
	if doctype == "" {
		doctype = "html"
	}
	var attrs []string
	if IsRTL(c) {
		attrs = append(attrs, `dir="rtl"`)
	}
	lang := BlogInfoLanguage(c)
	htmlType := "text/html"
	if c != nil && c.Options != nil {
		if v := c.Options.GetOption("html_type", "text/html"); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				htmlType = strings.TrimSpace(s)
			}
		}
	}
	if lang != "" {
		esc := html.EscapeString(lang)
		if htmlType == "text/html" || doctype == "html" {
			attrs = append(attrs, `lang="`+esc+`"`)
		}
		if (htmlType != "" && htmlType != "text/html") || doctype == "xhtml" {
			attrs = append(attrs, `xml:lang="`+esc+`"`)
		}
	}
	out := strings.Join(attrs, " ")
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("language_attributes", out, doctype)
		if s, ok := filtered.(string); ok {
			out = s
		}
	}
	return out
}

// LanguageAttributesHTML returns ` lang="…"` (with leading space) for embedding
// in an opening <html> tag, or empty when attributes are blank (P3.120).
func LanguageAttributesHTML(c *Context) string {
	attrs := GetLanguageAttributes(c, "html")
	if attrs == "" {
		return ""
	}
	return " " + attrs
}
