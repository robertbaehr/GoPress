package wp

import (
	"net/http"
	"net/url"
	"strings"
)

// GetRawReferer returns the unvalidated referer from `_wp_http_referer`
// (query or POST body) or the HTTP Referer header (wp_get_raw_referer, P2.74).
//
// Returns "" when neither is present (PHP false). Do not use for redirects —
// callers that need a safe URL should use GetReferer / ValidateRedirect.
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

// GetReferer returns a validated referer URL (wp_get_referer, P2.75).
//
// Rejects the current request URI (and home_url + REQUEST_URI) so self-referers
// are not used for redirects. Returns "" when missing/invalid (PHP false).
func GetReferer(c *Context) string {
	ref := GetRawReferer(c)
	if ref == "" || c == nil || c.Req == nil {
		return ""
	}
	reqURI := ""
	if c.Req.URL != nil {
		reqURI = c.Req.URL.RequestURI()
	}
	if reqURI == "" {
		reqURI = c.Req.URL.Path
	}
	reqURI = UnslashString(reqURI)
	if ref == reqURI {
		return ""
	}
	home := strings.TrimRight(HomeURL(c, "", ""), "/")
	if home != "" && ref == home+reqURI {
		return ""
	}
	return ValidateRedirect(c, ref, "")
}

// ValidateRedirect mirrors wp_validate_redirect (P2.75 / P6.20 subset).
// Relative same-origin paths are kept; absolute URLs must match home's host
// (plus allowed_redirect_hosts filter). Invalid locations return fallback.
func ValidateRedirect(c *Context, location, fallback string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return fallback
	}
	// Browsers treat //host as scheme-relative; force http: like core before parse.
	if strings.HasPrefix(location, "//") {
		location = "http:" + location
	}

	cut := strings.IndexByte(location, '?')
	test := location
	if cut >= 0 {
		test = location[:cut]
	}
	lp, err := url.Parse(test)
	if err != nil {
		return fallback
	}

	if lp.Scheme != "" && lp.Scheme != "http" && lp.Scheme != "https" {
		return fallback
	}
	// Reject scheme-only / malformed hosts (https:evil.com).
	if lp.Host == "" && (lp.Scheme != "" || lp.User != nil || lp.Opaque != "") {
		return fallback
	}

	// Relative path — allow.
	if lp.Host == "" {
		if strings.HasPrefix(location, "/") && !strings.HasPrefix(location, "//") {
			return location
		}
		return fallback
	}

	homeHost := ""
	home := HomeURL(c, "", "")
	if hu, err := url.Parse(home); err == nil && hu != nil {
		homeHost = strings.ToLower(hu.Hostname())
	}
	allowed := []string{}
	if homeHost != "" {
		allowed = append(allowed, homeHost)
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("allowed_redirect_hosts", allowed, lp.Hostname()); v != nil {
			switch t := v.(type) {
			case []string:
				allowed = t
			case []any:
				allowed = nil
				for _, x := range t {
					if s, ok := x.(string); ok {
						allowed = append(allowed, s)
					}
				}
			}
		}
	}
	host := strings.ToLower(lp.Hostname())
	ok := strings.EqualFold(host, homeHost)
	if !ok {
		for _, h := range allowed {
			if strings.EqualFold(host, h) {
				ok = true
				break
			}
		}
	}
	if !ok {
		return fallback
	}
	return location
}
