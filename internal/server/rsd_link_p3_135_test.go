package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.135: front head uses RsdLink (EditURI → xmlrpc.php?rsd).
func TestFrontHeadRsdLinkP3135(t *testing.T) {
	c := testBoot(t)
	s := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `rel="EditURI"`) || !strings.Contains(body, "xmlrpc.php?rsd") {
		t.Fatalf("front head missing RSD link in: %q", body[:min(1200, len(body))])
	}
	if got := wp.RsdLink(c); !strings.Contains(body, strings.TrimSpace(got)) && !strings.Contains(body, "xmlrpc.php?rsd") {
		t.Fatalf("RsdLink output not reflected in head: helper=%q", got)
	}
}
