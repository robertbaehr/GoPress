package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestXMLRPCGone(t *testing.T) {
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodGet, "/xmlrpc.php", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("GET status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "disabled") {
		t.Fatalf("GET body=%q", rec.Body.String())
	}
	if rec.Header().Get("Allow") == "" {
		t.Fatal("expected Allow header")
	}
	if rec.Header().Get("X-Robots-Tag") != "noindex" {
		t.Fatalf("X-Robots-Tag=%q", rec.Header().Get("X-Robots-Tag"))
	}
}

func TestXMLRPCGonePostFault(t *testing.T) {
	// P3.94: POST returns XML-RPC fault methodResponse.
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodPost, "/xmlrpc.php", strings.NewReader(`<?xml version="1.0"?><methodCall><methodName>system.listMethods</methodName></methodCall>`))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST status=%d body=%q", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/xml") {
		t.Fatalf("Content-Type=%q", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<methodResponse>",
		"<fault>",
		"<name>faultCode</name>",
		"<name>faultString</name>",
		"disabled",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}

func TestXMLRPCGoneHead(t *testing.T) {
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodHead, "/xmlrpc.php", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("HEAD status=%d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD body should be empty: %q", rec.Body.String())
	}
}

func TestXMLRPCPingbackPingFault(t *testing.T) {
	// P3.98: pingback.ping POSTs get an explicit pingback-disabled fault.
	s := &server.Server{}
	payload := `<?xml version="1.0"?><methodCall><methodName>pingback.ping</methodName><params><param><value><string>http://example.com/</string></value></param></params></methodCall>`
	req := httptest.NewRequest(http.MethodPost, "/xmlrpc.php", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Pingback"); got != "" {
		t.Fatalf("X-Pingback must be absent, got %q", got)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<methodResponse>",
		"<fault>",
		"Pingbacks are disabled",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
	if strings.Contains(body, "XML-RPC services are disabled") {
		t.Fatalf("expected pingback-specific fault, got generic: %q", body)
	}
}
