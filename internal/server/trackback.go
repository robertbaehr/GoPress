package server

import (
	"net/http"
)

// serveTrackbackGone rejects wp-trackback.php as out of scope (P3.19).
func serveTrackbackGone(w http.ResponseWriter, r *http.Request) {
	_ = r
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte("Trackback and pingback services are disabled on this site.\n"))
}
