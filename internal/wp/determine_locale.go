package wp

import (
	"net/http"
	"regexp"
	"strings"
)

var reSanitizeLocaleName = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// SanitizeLocaleName keeps A–Z, a–z, 0–9, _, - (sanitize_locale_name, P6.61).
func SanitizeLocaleName(localeName string) string {
	return reSanitizeLocaleName.ReplaceAllString(localeName, "")
}

// GetLocale returns the site locale (get_locale, P6.61).
// WPLANG option when set; otherwise en_US. Honors the locale filter.
func GetLocale(c *Context) string {
	locale := "en_US"
	if c != nil && c.Options != nil {
		if v := strings.TrimSpace(optionValueString(c.Options.GetOption("WPLANG", ""))); v != "" {
			locale = v
		}
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("locale", locale); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return locale
}

// DetermineLocale resolves the request locale (determine_locale, P6.61).
// Login subset: pre_determine_locale → ?wp_lang= / wp_lang cookie → get_locale →
// determine_locale filter. Callers on wp-login.php pass the request; admin/user
// locale branches land in a later task.
func DetermineLocale(c *Context, r *http.Request) string {
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("pre_determine_locale", nil); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}

	determined := ""
	if r != nil {
		if q := strings.TrimSpace(r.URL.Query().Get("wp_lang")); q != "" {
			determined = SanitizeLocaleName(q)
		} else if ck, err := r.Cookie("wp_lang"); err == nil && ck != nil && strings.TrimSpace(ck.Value) != "" {
			determined = SanitizeLocaleName(ck.Value)
		}
	}
	if determined == "" {
		determined = GetLocale(c)
	}

	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("determine_locale", determined); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return determined
}
