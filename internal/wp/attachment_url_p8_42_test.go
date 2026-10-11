package wp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.42: SSL front forces https; empty filter result clears URL.
func TestGetAttachmentURLSSLandFilterP842(t *testing.T) {
	c := mediaBoot(t)

	req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
	req.TLS = nil
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)

	url := wp.GetAttachmentURL(c, 5)
	if url == "" || !strings.HasPrefix(url, "https://") {
		t.Fatalf("expected https attachment URL, got %q", url)
	}
	if !strings.Contains(url, "sample-photo-a") {
		t.Fatalf("url=%q", url)
	}

	c.Hooks.AddFilter("wp_get_attachment_url", func(value any, _ ...any) any {
		return ""
	}, 10, 2)
	if got := wp.GetAttachmentURL(c, 5); got != "" {
		t.Fatalf("empty filter should clear URL, got %q", got)
	}
}
