package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestKsesAttrKeepsAllowed(t *testing.T) {
	out := wp.KsesAttr("a", `href="https://example.com" class="btn" onclick="alert(1)"`, "post", nil)
	if !strings.HasPrefix(out, "<a ") || !strings.HasSuffix(out, ">") {
		t.Fatalf("tag shape: %q", out)
	}
	if !strings.Contains(out, `href="https://example.com"`) {
		t.Fatalf("expected href kept: %q", out)
	}
	if !strings.Contains(out, `class="btn"`) {
		t.Fatalf("expected class kept: %q", out)
	}
	if strings.Contains(out, "onclick") {
		t.Fatalf("onclick should be stripped: %q", out)
	}
}

func TestKsesAttrXHTMLSlash(t *testing.T) {
	out := wp.KsesAttr("img", `src="https://example.com/a.png" alt="x" /`, "post", nil)
	if !strings.HasSuffix(out, " />") && !strings.Contains(out, " /") {
		t.Fatalf("expected xhtml slash: %q", out)
	}
	if !strings.Contains(out, `src="https://example.com/a.png"`) {
		t.Fatalf("expected src: %q", out)
	}
}

func TestKsesAttrDisallowedElement(t *testing.T) {
	out := wp.KsesAttr("script", `src="https://evil.test/x.js"`, "post", nil)
	if out != "<script>" {
		t.Fatalf("disallowed element should drop attrs: %q", out)
	}
}

func TestKsesAttrCheckRejectsUnknown(t *testing.T) {
	allow, _ := wp.KsesAllowedHTML(nil, "post").(wp.AllowedHTML)
	ok, whole := wp.KsesAttrCheck("onclick", "alert(1)", `onclick="alert(1)"`, "n", "a", allow)
	if ok || whole != "" {
		t.Fatalf("onclick should fail: ok=%v whole=%q", ok, whole)
	}
	ok, whole = wp.KsesAttrCheck("href", "https://example.com", `href="https://example.com"`, "n", "a", allow)
	if !ok || !strings.Contains(whole, "href=") {
		t.Fatalf("href should pass: ok=%v whole=%q", ok, whole)
	}
}
