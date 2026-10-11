package server

import (
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

var reXMLRPCMethodName = regexp.MustCompile(`(?is)<methodName>\s*([^<\s]+)\s*</methodName>`)

// serveXMLRPCGone rejects XML-RPC as out of scope (P3.10 / P3.94 / P3.98).
// POST returns an XML-RPC fault payload; GET/HEAD keep a plain disabled notice.
// pingback.ping POSTs get a pingback-specific fault string (P3.98).
// xmlrpc.php?rsd stays 403 (RSD lives at /rsd.xml, P3.32).
func serveXMLRPCGone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Allow", "GET, POST")
	// P3.131 / P3.98 / P3.41: never advertise pingback discovery from xmlrpc.php.
	wp.ClearPingbackHeaders(w)
	method := http.MethodGet
	if r != nil {
		method = r.Method
	}
	switch method {
	case http.MethodPost:
		w.Header().Set("Content-Type", "text/xml; charset=UTF-8")
		faultCode := 405
		faultMsg := "XML-RPC services are disabled on this site."
		if name := xmlrpcMethodName(r); strings.EqualFold(name, "pingback.ping") {
			// P3.98: pingbacks ride XML-RPC; reject with an explicit pingback fault.
			faultCode = 405
			faultMsg = "Pingbacks are disabled on this site."
		}
		body := xmlrpcFaultResponse(faultCode, faultMsg)
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

// xmlrpcMethodName extracts <methodName> from a limited POST body (P3.98).
func xmlrpcMethodName(r *http.Request) string {
	if r == nil || r.Body == nil {
		return ""
	}
	// Bound read: methodName appears near the top of typical methodCall payloads.
	limited := io.LimitReader(r.Body, 4096)
	raw, err := io.ReadAll(limited)
	if err != nil || len(raw) == 0 {
		return ""
	}
	m := reXMLRPCMethodName.FindSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(string(m[1]))
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
