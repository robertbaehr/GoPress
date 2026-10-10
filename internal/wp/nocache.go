// nocache_headers / wp_get_nocache_headers (P6.27) — subset of wp-includes/functions.php.
package wp

import "net/http"

// NoCacheHeaders sets Expires + Cache-Control to prevent caching (nocache_headers).
// Matches WP's wp_get_nocache_headers defaults (no Last-Modified).
func NoCacheHeaders(w http.ResponseWriter) {
	if w == nil {
		return
	}
	h := w.Header()
	h.Set("Expires", "Wed, 11 Jan 1984 05:00:00 GMT")
	h.Set("Cache-Control", "no-cache, must-revalidate, max-age=0, no-store, private")
	h.Del("Last-Modified")
}
