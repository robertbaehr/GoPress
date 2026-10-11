package wp_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestHttpValidateURLBasicsP2146(t *testing.T) {
	c := testBootQuery(t)

	if got := wp.HttpValidateURL(c, ""); got != "" {
		t.Fatalf("empty → %q", got)
	}
	if got := wp.HttpValidateURL(c, "12345"); got != "" {
		t.Fatalf("numeric → %q", got)
	}
	if got := wp.HttpValidateURL(c, "javascript:alert(1)"); got != "" {
		t.Fatalf("js → %q", got)
	}
	if got := wp.HttpValidateURL(c, "http://user:pass@example.com/"); got != "" {
		t.Fatalf("userinfo → %q", got)
	}
	// Fixture home is often 127.0.0.1 — use a distinct loopback / private host.
	if got := wp.HttpValidateURL(c, "http://127.0.0.2/x"); got != "" {
		t.Fatalf("loopback → %q", got)
	}
	if got := wp.HttpValidateURL(c, "http://192.168.1.1/x"); got != "" {
		t.Fatalf("private → %q", got)
	}
	if got := wp.HttpValidateURL(c, "http://8.8.8.8/x"); got != "http://8.8.8.8/x" {
		t.Fatalf("public IP → %q", got)
	}
	if got := wp.HttpValidateURL(c, "https://example.com/a"); got != "https://example.com/a" {
		t.Fatalf("https → %q", got)
	}
	if got := wp.HttpValidateURL(c, "http://example.com:8080/a"); got != "http://example.com:8080/a" {
		t.Fatalf("safe port → %q", got)
	}
	if got := wp.HttpValidateURL(c, "http://example.com:9999/a"); got != "" {
		t.Fatalf("unsafe port → %q", got)
	}
}

func TestHttpValidateURLSameHostAndExternalFilterP2146(t *testing.T) {
	c := testBootQuery(t)
	home := ""
	if c.Options != nil {
		home = strings.TrimSpace(fmt.Sprint(c.Options.GetOption("home", "")))
	}
	if home == "" {
		t.Skip("no home option")
	}
	parsed, ok := wp.ParseURL(home)
	if !ok || parsed == nil || parsed.Host == "" {
		t.Skip("unparseable home")
	}
	// Same host should skip private-IP checks even for loopback home in some fixtures.
	same := "http://" + parsed.Host + "/path"
	if got := wp.HttpValidateURL(c, same); got != same {
		t.Fatalf("same host → %q want %q", got, same)
	}

	c.Hooks.AddFilter("http_request_host_is_external", func(value any, _ ...any) any { return true }, 10, 3)
	if got := wp.HttpValidateURL(c, "http://10.0.0.5/x"); got != "http://10.0.0.5/x" {
		t.Fatalf("external filter allow → %q", got)
	}
}
