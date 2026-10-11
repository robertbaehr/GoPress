package wp

import (
	"net/http"
	"strings"
)

// ClearPingbackHeaders removes X-Pingback and Link rel=pingback headers (P3.131).
// GoPress keeps XML-RPC/pingbacks out of scope (P3.41 / P3.98), so front and
// xmlrpc responses must never advertise pingback discovery.
func ClearPingbackHeaders(w http.ResponseWriter) {
	if w == nil {
		return
	}
	w.Header().Del("X-Pingback")
	vals := w.Header().Values("Link")
	if len(vals) == 0 {
		return
	}
	kept := make([]string, 0, len(vals))
	for _, v := range vals {
		lower := strings.ToLower(v)
		if strings.Contains(lower, `rel="pingback"`) || strings.Contains(lower, "rel=pingback") ||
			strings.Contains(lower, "rel='pingback'") {
			continue
		}
		kept = append(kept, v)
	}
	w.Header().Del("Link")
	for _, v := range kept {
		w.Header().Add("Link", v)
	}
}
