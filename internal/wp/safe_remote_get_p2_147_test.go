package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.147: SafeRemoteGet rejects private/unsafe URLs before dialing.
func TestSafeRemoteGetRejectsUnsafeP2147(t *testing.T) {
	c := testBootQuery(t)

	for _, bad := range []string{
		"",
		"javascript:alert(1)",
		"http://user:pass@example.com/",
		"http://127.0.0.2/x",
		"http://192.168.1.1/x",
		"http://example.com:9999/a",
	} {
		resp, err := wp.SafeRemoteGet(c, bad)
		if resp != nil {
			t.Fatalf("%q: expected nil response", bad)
		}
		he, ok := err.(*wp.HTTPError)
		if !ok {
			t.Fatalf("%q: err type %T: %v", bad, err, err)
		}
		if he.Code != "http_request_failed" {
			t.Fatalf("%q: code=%q", bad, he.Code)
		}
		if !strings.Contains(he.Message, "valid URL") {
			t.Fatalf("%q: message=%q", bad, he.Message)
		}
	}
}

// P2.147: SafeRemoteGet still honors BlockExternal after validation.
func TestSafeRemoteGetBlockExternalP2147(t *testing.T) {
	c := testBootQuery(t)
	if c.Config == nil {
		c.Config = &config.Config{}
	}
	c.Config.BlockExternal = true

	resp, err := wp.SafeRemoteGet(c, "https://example.com/")
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

// P2.147: validated public URL proceeds past validation (may fail to connect).
func TestSafeRemoteGetValidatedProceedsP2147(t *testing.T) {
	c := testBootQuery(t)
	if c.Config == nil {
		c.Config = &config.Config{}
	}
	c.Config.BlockExternal = false

	// Port 8080 is in the safe-ports allowlist; dial should fail without hitting validation.
	_, err := wp.SafeRemoteGet(c, "http://8.8.8.8:8080/")
	if err == nil {
		t.Fatal("expected connection error after validation")
	}
	he, ok := err.(*wp.HTTPError)
	if !ok {
		t.Fatalf("err type %T", err)
	}
	if he.Code == "http_request_not_executed" {
		t.Fatalf("should not be blocked after validation")
	}
	if strings.Contains(he.Message, "valid URL") {
		t.Fatalf("validated URL should not fail validation: %q", he.Message)
	}
}
