package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

// P3.142: wp_robots filter can return map[string]string directive values.
func TestFrontRobotsMetaStringMapFilterP3142(t *testing.T) {
	c := testBoot(t)

	c.Hooks.AddFilter("wp_robots", func(value any, _ ...any) any {
		return map[string]string{
			"max-image-preview": "standard",
			"nosnippet":         "",
		}
	}, 20, 1)

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `max-image-preview:standard`) {
		t.Fatalf("expected string-map robots directives: %q", body[:min(600, len(body))])
	}
	// Empty string values are omitted by formatRobotsDirectives.
	if strings.Contains(body, "nosnippet") {
		t.Fatalf("empty string directive should be omitted: %q", body[:min(600, len(body))])
	}
}

func TestFrontRobotsSensitivePageFilterP3142(t *testing.T) {
	c := testBoot(t)

	c.Hooks.AddFilter("wp_robots", func(value any, _ ...any) any {
		m, _ := value.(map[string]any)
		if m == nil {
			m = map[string]any{}
		}
		// Mirror wp_robots_sensitive_page (P3.142).
		m["noindex"] = true
		m["noarchive"] = true
		return m
	}, 20, 1)

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{"noindex", "noarchive", "max-image-preview:large"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body[:min(600, len(body))])
		}
	}
}

func TestFrontRobotsSearchStillNoindexFollowP3142(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?s=robots", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	want := `content='noindex, follow, max-image-preview:large'`
	if !strings.Contains(body, want) {
		t.Fatalf("expected search robots %q in %q", want, body[:min(600, len(body))])
	}
}
