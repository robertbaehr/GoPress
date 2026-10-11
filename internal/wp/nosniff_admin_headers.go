package wp

import (
	"net/http"
	"strings"
)

// SendNosniffHeader sets X-Content-Type-Options: nosniff (wp_send_nosniff_header, P2.159).
func SendNosniffHeader(w http.ResponseWriter) {
	if w == nil {
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// AdminHeaders sets the admin Referrer-Policy header (wp_admin_headers, P2.159).
// Default policy is strict-origin-when-cross-origin; filter admin_referrer_policy may replace it.
// Empty filtered policy skips the header.
func AdminHeaders(w http.ResponseWriter, c *Context) {
	if w == nil {
		return
	}
	policy := "strict-origin-when-cross-origin"
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("admin_referrer_policy", policy); v != nil {
			if s, ok := v.(string); ok {
				policy = s
			}
		}
	}
	policy = strings.TrimSpace(policy)
	if policy == "" {
		return
	}
	w.Header().Set("Referrer-Policy", policy)
}
