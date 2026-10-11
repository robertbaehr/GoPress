// get_body_class / body_class helpers (P3.114) — subset of wp-includes/post-template.php.
package wp

import (
	"strings"
)

// GetBodyClass appends shared front body classes and runs the body_class filter (P3.114).
// Base classes (home/blog/singular/…) are supplied by the caller; this adds
// logged-in, admin-bar, no-customize-support, and wp-custom-logo when applicable.
func GetBodyClass(c *Context, classes string) string {
	parts := splitBodyClasses(classes)
	if c != nil && c.CurrentUser != nil {
		parts = appendBodyClassUnique(parts, "logged-in")
		// P7.128: admin-bar class follows is_admin_bar_showing().
		if IsAdminBarShowing(c) {
			parts = appendBodyClassUnique(parts, "admin-bar")
			parts = appendBodyClassUnique(parts, "no-customize-support")
		}
	}
	if HasCustomLogo(c) {
		parts = appendBodyClassUnique(parts, "wp-custom-logo")
	}
	// P7.153: custom-background when theme supports + non-default color/image.
	if HasCustomBackground(c) {
		parts = appendBodyClassUnique(parts, "custom-background")
	}
	// P7.154: custom-header when theme supports + header image / non-default text color.
	if HasCustomHeader(c) {
		parts = appendBodyClassUnique(parts, "custom-header")
	}
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("body_class", parts, []string{})
		switch t := filtered.(type) {
		case []string:
			parts = t
		case []any:
			parts = make([]string, 0, len(t))
			for _, v := range t {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					parts = append(parts, strings.TrimSpace(s))
				}
			}
		}
	}
	return strings.Join(dedupeBodyClasses(parts), " ")
}

// BodyClassWithAdminBar appends logged-in / admin-bar body classes when a user is present.
// P3.114: delegates to GetBodyClass (also wp-custom-logo + body_class filter).
func BodyClassWithAdminBar(c *Context, classes string) string {
	return GetBodyClass(c, classes)
}

func splitBodyClasses(classes string) []string {
	classes = strings.TrimSpace(classes)
	if classes == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Fields(classes) {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func appendBodyClassUnique(parts []string, class string) []string {
	class = strings.TrimSpace(class)
	if class == "" {
		return parts
	}
	for _, p := range parts {
		if p == class {
			return parts
		}
	}
	return append(parts, class)
}

func dedupeBodyClasses(parts []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
