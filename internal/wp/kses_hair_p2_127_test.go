package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestKsesHairBasic(t *testing.T) {
	attrs := wp.KsesHair(`class="is-wide" inert data-lazy='&lt;img&#00062'`, nil)
	if attrs["class"].Value != "is-wide" || attrs["class"].Vless != "n" {
		t.Fatalf("class=%+v", attrs["class"])
	}
	if attrs["class"].Whole != `class="is-wide"` {
		t.Fatalf("class.whole=%q", attrs["class"].Whole)
	}
	if attrs["inert"].Vless != "y" || attrs["inert"].Value != "" || attrs["inert"].Whole != "inert" {
		t.Fatalf("inert=%+v", attrs["inert"])
	}
	lazy := attrs["data-lazy"]
	if lazy.Vless != "n" {
		t.Fatalf("data-lazy=%+v", lazy)
	}
	// Value is syntax-escaped; entity decode happens before URI check only for URI attrs.
	if lazy.Value == "" {
		t.Fatalf("data-lazy empty: %+v", lazy)
	}
}

func TestKsesHairURIProtocol(t *testing.T) {
	attrs := wp.KsesHair(`href="javascript:alert(1)" src="https://example.com/a.png"`, []string{"http", "https"})
	href := attrs["href"]
	if href.Vless != "n" {
		t.Fatalf("href=%+v", href)
	}
	// Bad protocol stripped — value should not retain javascript:
	if containsFold(href.Value, "javascript") {
		t.Fatalf("href kept javascript: %q", href.Value)
	}
	src := attrs["src"]
	if src.Value != "https://example.com/a.png" {
		t.Fatalf("src=%q", src.Value)
	}
}

func TestKsesHairEmpty(t *testing.T) {
	if got := wp.KsesHair("", nil); len(got) != 0 {
		t.Fatalf("empty → %#v", got)
	}
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (stringIndexFold(s, sub) >= 0)))
}

func stringIndexFold(s, sub string) int {
	ls, lsub := len(s), len(sub)
	for i := 0; i+lsub <= ls; i++ {
		ok := true
		for j := 0; j < lsub; j++ {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}
