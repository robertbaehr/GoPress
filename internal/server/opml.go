package server

import (
	"net/http"
)

// serveOPMLGone rejects wp-links-opml.php as out of scope (P3.30).
// The classic Links Manager OPML export is unavailable with GoPress.
func serveOPMLGone(w http.ResponseWriter, r *http.Request) {
	_ = r
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte("OPML link export is disabled on this site.\n"))
}
