package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestWLWManifestStub(t *testing.T) {
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodGet, "/wp-includes/wlwmanifest.xml", nil)
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
		`xmlns="http://schemas.microsoft.com/wlw/manifest/weblog"`,
		"<clientType>WordPress</clientType>",
		"<serviceName>GoPress</serviceName>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}
