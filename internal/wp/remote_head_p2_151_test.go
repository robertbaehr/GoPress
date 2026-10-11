package wp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.151: RemoteHead uses HEAD; blocked when BlockExternal.
func TestRemoteHeadBlockExternalP2151(t *testing.T) {
	c := testBootQuery(t)
	if c.Config == nil {
		c.Config = &config.Config{}
	}
	c.Config.BlockExternal = true

	resp, err := wp.RemoteHead(c, "https://example.com/")
	if resp != nil {
		t.Fatalf("expected nil response")
	}
	he, ok := err.(*wp.HTTPError)
	if !ok {
		t.Fatalf("err type %T: %v", err, err)
	}
	if he.Code != "http_request_not_executed" {
		t.Fatalf("code=%q", he.Code)
	}
}

// P2.151: RemoteHead hits a local test server with HEAD.
func TestRemoteHeadMethodP2151(t *testing.T) {
	c := testBootQuery(t)
	if c.Config == nil {
		c.Config = &config.Config{}
	}
	c.Config.BlockExternal = false

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("X-Test", "1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	resp, err := wp.RemoteHead(c, srv.URL)
	if err != nil {
		t.Fatalf("RemoteHead: %v", err)
	}
	if gotMethod != http.MethodHead {
		t.Fatalf("method=%q", gotMethod)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if resp.Headers.Get("X-Test") != "1" {
		t.Fatalf("headers=%v", resp.Headers)
	}
}

// P2.151: SafeRemoteHead rejects unsafe URLs.
func TestSafeRemoteHeadRejectsUnsafeP2151(t *testing.T) {
	c := testBootQuery(t)
	resp, err := wp.SafeRemoteHead(c, "http://127.0.0.2/x")
	if resp != nil {
		t.Fatalf("expected nil response")
	}
	he, ok := err.(*wp.HTTPError)
	if !ok {
		t.Fatalf("err type %T: %v", err, err)
	}
	if he.Code != "http_request_failed" || !strings.Contains(he.Message, "valid URL") {
		t.Fatalf("code=%q msg=%q", he.Code, he.Message)
	}
}

// P2.151: SafeRemoteHead honors BlockExternal after validation.
func TestSafeRemoteHeadBlockExternalP2151(t *testing.T) {
	c := testBootQuery(t)
	if c.Config == nil {
		c.Config = &config.Config{}
	}
	c.Config.BlockExternal = true
	resp, err := wp.SafeRemoteHead(c, "https://example.com/")
	if resp != nil {
		t.Fatalf("expected nil")
	}
	he, ok := err.(*wp.HTTPError)
	if !ok || he.Code != "http_request_not_executed" {
		t.Fatalf("err=%v", err)
	}
}
