package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.145: deeper wp_parse_url parity — IPv6 brackets, empty URL path, translate helper.
func TestParseURLDeepIPv6AndEmptyP2145(t *testing.T) {
	p, ok := wp.ParseURL("http://[::1]:8080/path")
	if !ok {
		t.Fatal("expected parse ok")
	}
	m := p.ToMap()
	if m["host"] != "[::1]" {
		t.Fatalf("IPv6 host want [::1] got %#v", m["host"])
	}
	if m["port"] != 8080 || m["path"] != "/path" {
		t.Fatalf("IPv6 parts %#v", m)
	}

	p, ok = wp.ParseURL("")
	if !ok {
		t.Fatal("empty URL should parse")
	}
	m = p.ToMap()
	if path, has := m["path"]; !has || path != "" {
		t.Fatalf("empty URL want path:\"\" got %#v", m)
	}
	comp, ok := wp.ParseURLComponent("", wp.PHPURLPath)
	if !ok || comp != "" {
		t.Fatalf("PHP_URL_PATH on empty: %v ok=%v", comp, ok)
	}
}

func TestParseURLDeepColonQueryAndUserinfoP2145(t *testing.T) {
	p, ok := wp.ParseURL("/path?a=b:c")
	if !ok {
		t.Fatal("expected parse ok")
	}
	m := p.ToMap()
	if m["path"] != "/path" || m["query"] != "a=b:c" {
		t.Fatalf("colon-in-query %#v", m)
	}

	p, ok = wp.ParseURL("//user:pass@example.com/x")
	if !ok {
		t.Fatal("expected parse ok")
	}
	m = p.ToMap()
	if m["user"] != "user" || m["pass"] != "pass" || m["host"] != "example.com" {
		t.Fatalf("protocol-relative userinfo %#v", m)
	}
	if _, has := m["scheme"]; has {
		t.Fatalf("scheme should be unset: %#v", m)
	}

	p, ok = wp.ParseURL("http://example.com:0/")
	if !ok {
		t.Fatal("expected parse ok")
	}
	m = p.ToMap()
	if m["port"] != 0 {
		t.Fatalf("port 0 %#v", m)
	}
}

func TestTranslatePHPURLConstantToKeyP2145(t *testing.T) {
	if got := wp.TranslatePHPURLConstantToKey(wp.PHPURLHost); got != "host" {
		t.Fatalf("host=%q", got)
	}
	if got := wp.TranslatePHPURLConstantToKey(wp.PHPURLFragment); got != "fragment" {
		t.Fatalf("fragment=%q", got)
	}
	if got := wp.TranslatePHPURLConstantToKey(-1); got != "" {
		t.Fatalf("unknown=%q", got)
	}
	if got := wp.TranslatePHPURLConstantToKey(99); got != "" {
		t.Fatalf("out-of-range=%q", got)
	}
}
