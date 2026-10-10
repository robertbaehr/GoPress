package rest

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// setCollectionHeaders adds X-WP-Total, X-WP-TotalPages, and Link pagination headers.
func setCollectionHeaders(w http.ResponseWriter, r *http.Request, total, totalPages, page int) {
	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	w.Header().Set("Access-Control-Expose-Headers", "X-WP-Total, X-WP-TotalPages, Link")

	base := collectionURL(r)
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

func collectionURL(r *http.Request) *url.URL {
	u := *r.URL
	// Prefer absolute URL when Host is present (httptest may leave it empty).
	if r.Host != "" {
		u.Scheme = "http"
		if r.TLS != nil {
			u.Scheme = "https"
		}
		u.Host = r.Host
	}
	q := u.Query()
	// page is rewritten per link; drop existing so withPage can set it.
	q.Del("page")
	u.RawQuery = q.Encode()
	return &u
}

func withPage(base *url.URL, page int) string {
	u := *base
	q := u.Query()
	if page <= 1 {
		q.Del("page")
	} else {
		q.Set("page", strconv.Itoa(page))
	}
	u.RawQuery = q.Encode()
	return u.String()
}
