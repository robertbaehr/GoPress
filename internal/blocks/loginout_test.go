package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderLoginoutLoggedOut(t *testing.T) {
	env := &blocks.Env{
		IsLoggedIn: func() bool { return false },
		CurrentURL: func() string { return "http://example.test/sample-page/" },
		LoginURL: func(redirect string) string {
			if redirect != "" {
				return "http://example.test/wp-login.php?redirect_to=" + redirect
			}
			return "http://example.test/wp-login.php"
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:loginout /-->`)
	if !strings.Contains(out, `class="wp-block-loginout logged-out"`) {
		t.Fatalf("expected logged-out wrapper: %q", out)
	}
	if !strings.Contains(out, `>Log in</a>`) {
		t.Fatalf("expected Log in link: %q", out)
	}
	if !strings.Contains(out, `redirect_to=http://example.test/sample-page/`) {
		t.Fatalf("expected redirectToCurrent: %q", out)
	}
}

func TestRenderLoginoutLoggedIn(t *testing.T) {
	env := &blocks.Env{
		IsLoggedIn: func() bool { return true },
		CurrentURL: func() string { return "http://example.test/" },
		LogoutURL: func(redirect string) string {
			u := "http://example.test/wp-login.php?action=logout"
			if redirect != "" {
				u += "&redirect_to=" + redirect
			}
			return u
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:loginout {"redirectToCurrent":false} /-->`)
	if !strings.Contains(out, `class="wp-block-loginout logged-in"`) {
		t.Fatalf("expected logged-in wrapper: %q", out)
	}
	if !strings.Contains(out, `>Log out</a>`) {
		t.Fatalf("expected Log out link: %q", out)
	}
	if strings.Contains(out, `redirect_to=`) {
		t.Fatalf("redirectToCurrent=false should omit redirect: %q", out)
	}
}

func TestRenderLoginoutAsForm(t *testing.T) {
	env := &blocks.Env{
		IsLoggedIn: func() bool { return false },
		CurrentURL: func() string { return "http://example.test/hello-world/" },
		LoginURL: func(redirect string) string {
			return "http://example.test/wp-login.php"
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:loginout {"displayLoginAsForm":true} /-->`)
	if !strings.Contains(out, `has-login-form`) {
		t.Fatalf("expected has-login-form class: %q", out)
	}
	if !strings.Contains(out, `id="loginform"`) || !strings.Contains(out, `name="log"`) {
		t.Fatalf("expected login form fields: %q", out)
	}
	if !strings.Contains(out, `name="redirect_to" value="http://example.test/hello-world/"`) {
		t.Fatalf("expected redirect_to hidden: %q", out)
	}
	if !strings.Contains(out, `wp-block-button__link`) {
		t.Fatalf("expected submit button class: %q", out)
	}
	if strings.Contains(out, `>Log in</a>`) {
		t.Fatalf("form mode should not emit Log in link: %q", out)
	}
}

func TestRenderLoginoutWrapperPolish(t *testing.T) {
	env := &blocks.Env{
		IsLoggedIn: func() bool { return false },
		CurrentURL: func() string { return "http://example.test/" },
		LoginURL:   func(redirect string) string { return "http://example.test/wp-login.php" },
	}
	out := blocks.RenderWith(env, `<!-- wp:loginout {"className":"is-style-outline","backgroundColor":"cyan-bluish-gray","anchor":"login-widget","style":{"elements":{"link":{"color":{"text":"var:preset|color|vivid-red"}}},"color":{"background":"#f0f0f0"}}} /-->`)
	if !strings.Contains(out, `wp-block-loginout logged-out`) {
		t.Fatalf("expected base classes: %q", out)
	}
	if !strings.Contains(out, `has-link-color`) {
		t.Fatalf("expected has-link-color: %q", out)
	}
	if !strings.Contains(out, `has-background`) || !strings.Contains(out, `has-cyan-bluish-gray-background-color`) {
		t.Fatalf("expected background classes: %q", out)
	}
	if !strings.Contains(out, `is-style-outline`) {
		t.Fatalf("expected className: %q", out)
	}
	if !strings.Contains(out, `id="login-widget"`) {
		t.Fatalf("expected anchor id: %q", out)
	}
}
