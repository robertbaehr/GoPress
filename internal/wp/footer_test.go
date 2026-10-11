package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFooterFiresWpFooterAndPrintFooterScripts(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	wp.RegisterDefaultFooterHooks(c)
	var order []string
	c.Hooks.AddAction("wp_footer", func(_ ...any) {
		order = append(order, "footer")
	}, 10, 0)
	c.Hooks.AddAction("wp_print_footer_scripts", func(_ ...any) {
		order = append(order, "print")
	}, 10, 0)

	wp.Footer(c)
	if c.Hooks.DidAction("wp_footer") != 1 {
		t.Fatalf("DidAction(wp_footer)=%d", c.Hooks.DidAction("wp_footer"))
	}
	if c.Hooks.DidAction("wp_print_footer_scripts") != 1 {
		t.Fatalf("DidAction(wp_print_footer_scripts)=%d", c.Hooks.DidAction("wp_print_footer_scripts"))
	}
	// Priority 10 footer callback runs before priority 20 PrintFooterScripts.
	if len(order) < 2 || order[0] != "footer" || order[1] != "print" {
		t.Fatalf("order=%v want footer then print", order)
	}
}

func TestFooterNilSafe(t *testing.T) {
	wp.Footer(nil)
	wp.PrintFooterScripts(nil)
	wp.RegisterDefaultFooterHooks(nil)
}
