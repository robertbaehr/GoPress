package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestAllowedHTMLUserDescriptionAddsRelTarget(t *testing.T) {
	ud := wp.AllowedHTMLUserDescription()
	a := ud["a"]
	if a == nil {
		t.Fatal("expected a tag")
	}
	if a["rel"] != true || a["target"] != true {
		t.Fatalf("expected a[rel]/a[target]: %#v", a)
	}
	// Mutating user_description must not mutate data context.
	data := wp.AllowedHTMLData()
	if data["a"] != nil && data["a"]["rel"] == true {
		t.Fatal("data context should not gain rel from user_description clone")
	}
}

func TestAllowedHTMLFilteredPreUserDescription(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	for _, ctx := range []string{"user_description", "pre_user_description", "pre_term_description"} {
		got := wp.AllowedHTMLFiltered(c, ctx)
		if got["a"]["rel"] != true || got["a"]["target"] != true {
			t.Fatalf("%s missing rel/target: %#v", ctx, got["a"])
		}
	}
}

func TestAllowedHTMLStrip(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	got := wp.AllowedHTMLFiltered(c, "strip")
	if len(got) != 0 {
		t.Fatalf("strip should be empty: %#v", got)
	}
	cleaned := wp.Kses(`<p>Hi <strong>there</strong></p>`, got, nil)
	if strings.Contains(cleaned, "<") {
		t.Fatalf("strip kses should remove tags: %q", cleaned)
	}
}

func TestKsesAllowedHTMLEntities(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	ents, ok := wp.KsesAllowedHTML(c, "entities").([]string)
	if !ok || len(ents) < 10 {
		t.Fatalf("entities: %#v", ents)
	}
	found := false
	for _, e := range ents {
		if e == "nbsp" || e == "amp" || e == "lt" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected common entities in %v…", ents[:min(10, len(ents))])
	}
}

func TestKsesAllowedHTMLExplicitMap(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_kses_allowed_html", func(value any, args ...any) any {
		if len(args) > 0 && args[0] == "explicit" {
			m := value.(wp.AllowedHTML)
			out := make(wp.AllowedHTML, len(m)+1)
			for k, v := range m {
				out[k] = v
			}
			out["mark"] = wp.AttrRule{"class": true}
			return out
		}
		return value
	}, 10, 2)
	base := wp.AllowedHTML{"em": wp.AttrRule{}}
	got, ok := wp.KsesAllowedHTML(c, base).(wp.AllowedHTML)
	if !ok || got["mark"] == nil || got["em"] == nil {
		t.Fatalf("explicit: %#v", got)
	}
}

func TestAllowedHTMLFilteredRestoresFormWhenInputAllowed(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_kses_allowed_html", func(value any, args ...any) any {
		// Drop form but keep input (CUSTOM_TAGS-style filter).
		m := value.(wp.AllowedHTML)
		out := make(wp.AllowedHTML, len(m))
		for k, v := range m {
			if k == "form" {
				continue
			}
			out[k] = v
		}
		out["input"] = wp.AttrRule{"type": true, "name": true}
		return out
	}, 10, 2)
	got := wp.AllowedHTMLFiltered(c, "post")
	if got["form"] == nil {
		t.Fatal("expected form restored when input present")
	}
	if got["form"]["action"] != true || got["form"]["method"] != true {
		t.Fatalf("form attrs: %#v", got["form"])
	}
}
