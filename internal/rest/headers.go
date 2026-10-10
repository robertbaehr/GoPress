package rest

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// setAPILinkHeader sets the WordPress discovery Link header (rel=https://api.w.org/).
// Collection pagination may overwrite Link later via setCollectionHeaders.
func setAPILinkHeader(w http.ResponseWriter, c *wp.Context) {
	root := restRootURL(c)
	if root == "" {
		return
	}
	w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="https://api.w.org/"`, root))
}

func restRootURL(c *wp.Context) string {
	siteURL := strings.TrimRight(optionString(c, "siteurl", ""), "/")
	home := strings.TrimRight(optionString(c, "home", siteURL), "/")
	if home == "" {
		home = siteURL
	}
	if home == "" {
		return ""
	}
	return home + "/wp-json/"
}

// setCollectionHeaders adds X-WP-Total, X-WP-TotalPages, and Link pagination headers.
func setCollectionHeaders(w http.ResponseWriter, r *http.Request, c *wp.Context, total, totalPages, page int) {
	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	w.Header().Set("Access-Control-Expose-Headers", "X-WP-Total, X-WP-TotalPages, Link")

	base := collectionURL(r, c)
	var links []string
	if page > 1 {
		links = append(links, fmt.Sprintf(`<%s>; rel="prev"`, withPage(base, page-1)))
	}
	if totalPages > 0 && page < totalPages {
		links = append(links, fmt.Sprintf(`<%s>; rel="next"`, withPage(base, page+1)))
	}
	if len(links) > 0 {
		w.Header().Set("Link", strings.Join(links, ", "))
	}
}

func collectionURL(r *http.Request, c *wp.Context) *url.URL {
	u := *r.URL
	// Prefer site home/url (matches WP get_rest_url host) over the request Host,
	// so Go on :8082 still emits Link targets against the configured site URL.
	if root := restRootURL(c); root != "" {
		if parsed, err := url.Parse(root); err == nil && parsed.Host != "" {
			u.Scheme = parsed.Scheme
			if u.Scheme == "" {
				u.Scheme = "http"
			}
			u.Host = parsed.Host
		}
	} else if r.Host != "" {
		u.Scheme = "http"
		if r.TLS != nil {
			u.Scheme = "https"
		}
		u.Host = r.Host
	}
	// Preserve original query param order; drop page so withPage can append it.
	u.RawQuery = stripQueryParam(u.RawQuery, "page")
	return &u
}

func stripQueryParam(raw, key string) string {
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, "&")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		k, _, _ := strings.Cut(p, "=")
		if k == key {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, "&")
}

func withPage(base *url.URL, page int) string {
	u := *base
	raw := stripQueryParam(u.RawQuery, "page")
	if page > 1 {
		pageQ := "page=" + strconv.Itoa(page)
		if raw == "" {
			raw = pageQ
		} else {
			raw = raw + "&" + pageQ
		}
	}
	u.RawQuery = raw
	return u.String()
}
