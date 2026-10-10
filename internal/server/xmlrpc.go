package server

import (
	"net/http"
)

// serveXMLRPCGone rejects XML-RPC as out of scope (P3.10).
func serveXMLRPCGone(w http.ResponseWriter, r *http.Request) {
	_ = r
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte("XML-RPC services are disabled on this site.\n"))
}
