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

func TestParseURLEdgeCasesP279(t *testing.T) {
	// PHP parse_url("example.com:80/path") → host, port, path
	p, ok := wp.ParseURL("example.com:80/path")
	if !ok {
		t.Fatal("expected parse ok")
	}
	m := p.ToMap()
	if m["host"] != "example.com" || m["port"] != 80 || m["path"] != "/path" {
		t.Fatalf("schemeless host:port => %#v", m)
	}

	// Empty password retained (PHP returns pass: "").
	p, ok = wp.ParseURL("http://user:@example.com/x")
	if !ok {
		t.Fatal("expected parse ok")
	}
	m = p.ToMap()
	if m["user"] != "user" {
		t.Fatalf("user=%v", m["user"])
	}
	if _, has := m["pass"]; !has {
		t.Fatalf("expected empty pass key: %#v", m)
	}

	// Bare scheme fails like PHP false.
	if _, ok := wp.ParseURL("http://"); ok {
		t.Fatal("http:// should fail")
	}
	if _, ok := wp.ParseURL("http:///path"); ok {
		t.Fatal("http:///path should fail")
	}

	// Query / fragment only.
	p, ok = wp.ParseURL("?foo=bar")
	if !ok || p.ToMap()["query"] != "foo=bar" {
		t.Fatalf("query-only: ok=%v %#v", ok, p.ToMap())
	}
	p, ok = wp.ParseURL("#frag")
	if !ok || p.ToMap()["fragment"] != "frag" {
		t.Fatalf("frag-only: ok=%v %#v", ok, p.ToMap())
	}
}

func TestParseURLNonHTTPSchemes(t *testing.T) {
	p, ok := wp.ParseURL("file:///tmp/x")
	if !ok {
		t.Fatal("file:/// should parse")
	}
	if p.Scheme != "file" || p.Path != "/tmp/x" {
		t.Fatalf("%+v", p)
	}
	p, ok = wp.ParseURL("mailto:a@b.c")
	if !ok || p.Scheme != "mailto" {
		t.Fatalf("mailto: %+v ok=%v", p, ok)
	}
}
