package server

import (
	"net/http"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveFavicon redirects /favicon.ico like WordPress do_favicon (P3.31).
func serveFavicon(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	target := wp.SiteIconRedirectURL(c)
	if target == "" {
		target = wp.DefaultSiteIconPath
	}
	w.Header().Set("X-Redirect-By", "WordPress")
	http.Redirect(w, r, target, http.StatusFound)
}
