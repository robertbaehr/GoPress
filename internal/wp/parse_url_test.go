package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestParseURLAbsolute(t *testing.T) {
	p, ok := wp.ParseURL("https://user:pass@example.com:8443/path?q=1#frag")
	if !ok || p == nil {
		t.Fatal("parse failed")
	}
	if p.Scheme != "https" || p.Host != "example.com" || p.Port != 8443 {
		t.Fatalf("%+v", p)
	}
	if p.User != "user" || p.Pass != "pass" || p.Path != "/path" || p.Query != "q=1" || p.Fragment != "frag" {
		t.Fatalf("%+v", p)
	}
	m := p.ToMap()
	if m["port"] != 8443 {
		t.Fatalf("port map=%v", m)
	}
}

func TestParseURLProtocolRelative(t *testing.T) {
	p, ok := wp.ParseURL("//example.com/foo")
	if !ok || p == nil {
		t.Fatal("parse failed")
	}
	if p.Scheme != "" {
		t.Fatalf("scheme should be unset, got %q", p.Scheme)
	}
	if p.Host != "example.com" || p.Path != "/foo" {
		t.Fatalf("%+v", p)
	}
}

func TestParseURLPathOnly(t *testing.T) {
	p, ok := wp.ParseURL("/wp-admin/options.php?page=1")
	if !ok || p == nil {
		t.Fatal("parse failed")
	}
	if p.Scheme != "" || p.Host != "" {
		t.Fatalf("scheme/host should be unset: %+v", p)
	}
	if p.Path != "/wp-admin/options.php" || p.Query != "page=1" {
		t.Fatalf("%+v", p)
	}
}

func TestParseURLComponent(t *testing.T) {
	host, ok := wp.ParseURLComponent("https://example.com/a", wp.PHPURLHost)
	if !ok || host != "example.com" {
		t.Fatalf("host=%v ok=%v", host, ok)
	}
	path, ok := wp.ParseURLComponent("https://example.com/a", wp.PHPURLPath)
	if !ok || path != "/a" {
		t.Fatalf("path=%v", path)
	}
	missing, ok := wp.ParseURLComponent("https://example.com/a", wp.PHPURLPort)
	if !ok || missing != nil {
		t.Fatalf("port=%v ok=%v", missing, ok)
	}
	all, ok := wp.ParseURLComponent("//cdn.example/x", -1)
	if !ok {
		t.Fatal("expected ok")
	}
	m, ok := all.(map[string]any)
	if !ok || m["host"] != "cdn.example" {
		t.Fatalf("%T %v", all, all)
	}
	if _, has := m["scheme"]; has {
		t.Fatalf("scheme should be absent: %v", m)
	}
}
