package wp

import (
	"net/http"
	"strings"
)

// GetRawReferer returns the unvalidated referer from `_wp_http_referer`
// (query or POST body) or the HTTP Referer header (wp_get_raw_referer, P2.74).
//
// Returns "" when neither is present (PHP false). Do not use for redirects —
// callers that need a safe URL should validate separately (wp_get_referer /
// wp_validate_redirect).
func GetRawReferer(c *Context) string {
	if c == nil || c.Req == nil {
		return ""
	}
	return getRawRefererFromRequest(c.Req)
}

func getRawRefererFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	// $_REQUEST['_wp_http_referer']: query string first, then POST body.
	if v := strings.TrimSpace(r.URL.Query().Get("_wp_http_referer")); v != "" {
		return UnslashString(v)
	}
	if r.PostForm != nil {
		if v := strings.TrimSpace(r.PostForm.Get("_wp_http_referer")); v != "" {
			return UnslashString(v)
		}
	} else if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		_ = r.ParseForm()
		if v := strings.TrimSpace(r.Form.Get("_wp_http_referer")); v != "" {
			return UnslashString(v)
		}
	}
	if v := strings.TrimSpace(r.Referer()); v != "" {
		return UnslashString(v)
	}
	// net/http normalizes Referer; also accept raw header if set oddly.
	if v := strings.TrimSpace(r.Header.Get("Referer")); v != "" {
		return UnslashString(v)
	}
	return ""
}
