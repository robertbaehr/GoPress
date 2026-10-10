package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestOPMLGone(t *testing.T) {
	s := &server.Server{}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/wp-links-opml.php", nil)
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s status=%d body=%q", method, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "disabled") {
			t.Fatalf("%s body=%q", method, rec.Body.String())
		}
	}
}
