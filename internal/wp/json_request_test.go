package wp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsJSONMediaType(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/ld+json", true},
		{"application/json+oembed", true},
		{"text/html, application/json", true},
		{"text/html", false},
		{"application/xml", false},
		{"", false},
	} {
		if got := wp.IsJSONMediaType(tc.in); got != tc.want {
			t.Fatalf("IsJSONMediaType(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsJSONRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if wp.IsJSONRequest(req) {
		t.Fatal("plain GET should not be JSON")
	}
	req.Header.Set("Accept", "application/json")
	if !wp.IsJSONRequest(req) {
		t.Fatal("Accept application/json")
	}
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set("Content-Type", "application/json; charset=utf-8")
	if !wp.IsJSONRequest(req2) {
		t.Fatal("Content-Type application/json")
	}
	if wp.IsJSONRequest(nil) {
		t.Fatal("nil request")
	}
}
