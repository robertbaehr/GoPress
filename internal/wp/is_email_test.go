package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsEmailValidCases(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"a@b.co", "a@b.co"},
		{"user@example.com", "user@example.com"},
		{"first.last+tag@example.co.uk", "first.last+tag@example.co.uk"},
		{"", ""},
		{"a@b.c", ""}, // too short (< 6)
		{"no-at-sign", ""},
		{"@example.com", ""},
		{"user@", ""},
		{"user@exam ple.com", ""},
		{"user@exa..mple.com", ""},
		{".user@example.com", ".user@example.com"}, // local charset allows leading dot
		{"user@-example.com", ""},
		{"user@example.com-", ""},
		{"user@exam_ple.com", ""},
	}
	for _, tc := range cases {
		got := wp.IsEmail(tc.in)
		if got != tc.want {
			t.Fatalf("IsEmail(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	if !wp.IsEmailValid("admin@example.com") {
		t.Fatal("expected valid")
	}
	if wp.IsEmailValid("not-an-email") {
		t.Fatal("expected invalid")
	}
}

func TestIsEmailFiltered(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	if got := wp.IsEmailFiltered(c, "admin@example.com"); got != "admin@example.com" {
		t.Fatalf("got %q", got)
	}
	c.Hooks.AddFilter("is_email", func(value any, args ...any) any {
		return false
	}, 10, 3)
	if got := wp.IsEmailFiltered(c, "admin@example.com"); got != "" {
		t.Fatalf("filter should reject, got %q", got)
	}

	c2 := &wp.Context{Hooks: wp.NewHooks()}
	c2.Hooks.AddFilter("is_email", func(value any, args ...any) any {
		// Force-accept short addresses when context is email_too_short.
		// ApplyFilters("is_email", false, email, context) → args[0]=email, args[1]=context.
		if len(args) >= 2 {
			if ctx, ok := args[1].(string); ok && ctx == "email_too_short" {
				if email, ok := args[0].(string); ok {
					return email
				}
			}
		}
		return value
	}, 10, 3)
	if got := wp.IsEmailFiltered(c2, "a@b.c"); got != "a@b.c" {
		t.Fatalf("filter override got %q", got)
	}
}
