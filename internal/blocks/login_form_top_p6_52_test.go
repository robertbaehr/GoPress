package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestLoginoutFormTopMiddleBottomFilters(t *testing.T) {
	in := `<!-- wp:loginout {"displayLoginAsForm":true} /-->`
	env := &blocks.Env{
		IsLoggedIn: func() bool { return false },
		ApplyFilter: func(tag string, value any, args ...any) any {
			switch tag {
			case "login_form_top":
				return `<!--TOP-->`
			case "login_form_middle":
				return `<!--MID-->`
			case "login_form_bottom":
				return `<!--BOT-->`
			default:
				return value
			}
		},
	}
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, "<!--TOP-->") {
		t.Fatalf("missing login_form_top: %q", out)
	}
	if !strings.Contains(out, "<!--MID-->") {
		t.Fatalf("missing login_form_middle: %q", out)
	}
	if !strings.Contains(out, "<!--BOT-->") {
		t.Fatalf("missing login_form_bottom: %q", out)
	}
	// Order: top after <form>, middle after password, bottom before </form>.
	top := strings.Index(out, "<!--TOP-->")
	user := strings.Index(out, `name="log"`)
	mid := strings.Index(out, "<!--MID-->")
	pwd := strings.Index(out, `name="pwd"`)
	bot := strings.Index(out, "<!--BOT-->")
	end := strings.Index(out, "</form>")
	if !(top < user && pwd < mid && mid < bot && bot < end) {
		t.Fatalf("filter order wrong: top=%d user=%d pwd=%d mid=%d bot=%d end=%d", top, user, pwd, mid, bot, end)
	}
}

func TestLoginoutFormTopAbsentWithoutFilter(t *testing.T) {
	in := `<!-- wp:loginout {"displayLoginAsForm":true} /-->`
	out := blocks.RenderWith(&blocks.Env{IsLoggedIn: func() bool { return false }}, in)
	if !strings.Contains(out, `id="loginform"`) {
		t.Fatalf("expected login form: %q", out)
	}
	if strings.Contains(out, "<!--TOP-->") {
		t.Fatal("unexpected top injection")
	}
}
