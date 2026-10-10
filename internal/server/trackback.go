package server

import (
	"net/http"
	"strings"
)

// serveTrackbackGone rejects wp-trackback.php as out of scope (P3.19 / P3.93).
// Responds with a trackback_response-shaped XML error (trackbacks are closed).
// Singular HTML also omits classic trackback_rdf() markup (P3.46).
func serveTrackbackGone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Allow", "GET, POST")
	// P3.93: XML body matches trackback_response(1, …) shape.
	body := trackbackResponseXML(1, "Sorry, trackbacks are closed for this item.")
	w.WriteHeader(http.StatusForbidden)
	if r != nil && r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(body))
}

// trackbackResponseXML builds a minimal trackback_response payload (P3.93).
func trackbackResponseXML(errCode int, message string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	b.WriteString("<response>\n")
	b.WriteString("<error>")
	if errCode != 0 {
		b.WriteString("1")
	} else {
		b.WriteString("0")
	}
	b.WriteString("</error>\n")
	if errCode != 0 && message != "" {
		b.WriteString("<message>")
		b.WriteString(xmlEscapeTrackback(message))
		b.WriteString("</message>\n")
	}
	b.WriteString("</response>\n")
	return b.String()
}

func xmlEscapeTrackback(s string) string {
	r := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
		`'`, "&apos;",
	)
	return r.Replace(s)
}
