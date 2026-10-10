package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestRSDXMLStub(t *testing.T) {
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodGet, "/rsd.xml", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "xml") {
		t.Fatalf("Content-Type=%q", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`xmlns="http://archipelago.phrasewise.com/rsd"`,
		"<engineName>GoPress</engineName>",
		"<apis>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
	// Must not fall through to front 501.
	if strings.Contains(body, "not implemented") {
		t.Fatalf("unexpected front 501: %q", body)
	}
}

func TestXMLRPCRsdStillGone(t *testing.T) {
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodGet, "/xmlrpc.php?rsd", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}
