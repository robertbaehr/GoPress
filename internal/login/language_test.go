package login

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteLanguageSwitcherEmpty(t *testing.T) {
	var b strings.Builder
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	writeLanguageSwitcher(&b, req, nil, nil)
	if b.Len() != 0 {
		t.Fatalf("expected no markup without languages, got %q", b.String())
	}
}

func TestWriteLanguageSwitcherWithLocales(t *testing.T) {
	var b strings.Builder
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?redirect_to=%2Fwp-admin%2F", nil)
	writeLanguageSwitcher(&b, req, nil, []string{"de_DE", "fr_FR"})
	out := b.String()
	if !strings.Contains(out, `class="language-switcher"`) {
		t.Fatalf("missing language-switcher: %q", out)
	}
	if !strings.Contains(out, `name="wp_lang"`) || !strings.Contains(out, `id="language-switcher-locales"`) {
		t.Fatalf("missing select: %q", out)
	}
	if !strings.Contains(out, "English (United States)") {
		t.Fatalf("missing en_US option: %q", out)
	}
	if !strings.Contains(out, `value="de_DE"`) || !strings.Contains(out, `value="fr_FR"`) {
		t.Fatalf("missing locale options: %q", out)
	}
	if !strings.Contains(out, `name="redirect_to"`) {
		t.Fatalf("missing redirect_to hidden: %q", out)
	}
	if !strings.Contains(out, `value="Change"`) {
		t.Fatalf("missing Change button: %q", out)
	}
}
