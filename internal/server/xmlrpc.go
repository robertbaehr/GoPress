package server

import (
	"net/http"
	"strconv"
	"strings"
)

// serveXMLRPCGone rejects XML-RPC as out of scope (P3.10 / P3.94).
// POST returns an XML-RPC fault payload; GET/HEAD keep a plain disabled notice.
// xmlrpc.php?rsd stays 403 (RSD lives at /rsd.xml, P3.32).
func serveXMLRPCGone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Allow", "GET, POST")
	method := http.MethodGet
	if r != nil {
		method = r.Method
	}
	switch method {
	case http.MethodPost:
		// P3.94: methodResponse fault shape for disabled XML-RPC.
		w.Header().Set("Content-Type", "text/xml; charset=UTF-8")
		body := xmlrpcFaultResponse(405, "XML-RPC services are disabled on this site.")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(body))
	case http.MethodHead:
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.WriteHeader(http.StatusForbidden)
	default:
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("XML-RPC services are disabled on this site.\n"))
	}
}

// xmlrpcFaultResponse builds a minimal XML-RPC methodResponse fault (P3.94).
func xmlrpcFaultResponse(code int, message string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString("<methodResponse>\n")
	b.WriteString("<fault>\n")
	b.WriteString("<value>\n")
	b.WriteString("<struct>\n")
	b.WriteString("<member><name>faultCode</name><value><int>")
	b.WriteString(strconv.Itoa(code))
	b.WriteString("</int></value></member>\n")
	b.WriteString("<member><name>faultString</name><value><string>")
	b.WriteString(xmlEscapeTrackback(message))
	b.WriteString("</string></value></member>\n")
	b.WriteString("</struct>\n")
	b.WriteString("</value>\n")
	b.WriteString("</fault>\n")
	b.WriteString("</methodResponse>\n")
	return b.String()
}
