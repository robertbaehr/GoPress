package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestHeadEnqueueScriptsHooks(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	wp.RegisterDefaultHeadHooks(c)

	var order []string
	c.Hooks.AddAction("wp_enqueue_scripts", func(args ...any) {
		order = append(order, "enqueue")
	}, 10, 0)
	c.Hooks.AddAction("wp_print_styles", func(args ...any) {
		order = append(order, "styles")
	}, 10, 0)
	c.Hooks.AddAction("wp_print_head_scripts", func(args ...any) {
		order = append(order, "head_scripts")
	}, 10, 0)
	c.Hooks.AddAction("wp_head", func(args ...any) {
		order = append(order, "head_mid")
	}, 5, 0)

	wp.Head(c)
	want := []string{"enqueue", "head_mid", "styles", "head_scripts"}
	if len(order) != len(want) {
		t.Fatalf("order=%v want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order=%v want %v", order, want)
		}
	}
}

func TestRegisterDefaultHeadHooksNilSafe(t *testing.T) {
	wp.RegisterDefaultHeadHooks(nil)
	wp.Head(nil)
	wp.EnqueueScripts(nil)
	wp.PrintStyles(nil)
	wp.PrintHeadScripts(nil)
}
