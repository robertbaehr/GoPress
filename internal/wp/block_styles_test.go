package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestBlockStyleRegistryRegisterGetUnregister(t *testing.T) {
	r := wp.NewBlockStyleRegistry()
	if !r.RegisterBlockStyle("core/separator", wp.BlockStyle{Name: "wide", Label: "Wide Line"}) {
		t.Fatal("register failed")
	}
	if !r.IsRegistered("core/separator", "wide") {
		t.Fatal("expected wide registered")
	}
	got, ok := r.Get("core/separator", "wide")
	if !ok || got.Label != "Wide Line" {
		t.Fatalf("get: %#v ok=%v", got, ok)
	}
	if r.RegisterBlockStyle("core/separator", wp.BlockStyle{Name: "has space"}) {
		t.Fatal("spaces in style name must fail")
	}
	if !r.UnregisterBlockStyle("core/separator", "wide") {
		t.Fatal("unregister failed")
	}
	if r.IsRegistered("core/separator", "wide") {
		t.Fatal("expected unregistered")
	}
}

func TestContextRegisterBlockStyleAndDefaults(t *testing.T) {
	c := bootWP(t)
	if c.BlockStyles == nil {
		t.Fatal("Boot should wire BlockStyles")
	}
	if !c.BlockStyles.IsRegistered("core/separator", "wide") {
		t.Fatal("default separator/wide missing")
	}
	if !c.RegisterBlockStyle("core/group", wp.BlockStyle{Name: "shadow", Label: "Shadow"}) {
		t.Fatal("context register failed")
	}
	styles := c.BlockStyles.ForBlock("core/group")
	if len(styles) != 1 || styles[0].Name != "shadow" {
		t.Fatalf("ForBlock: %#v", styles)
	}
	if !c.UnregisterBlockStyle("core/group", "shadow") {
		t.Fatal("context unregister failed")
	}
	all := c.BlockStyles.All()
	if len(all["core/separator"]) < 2 {
		t.Fatalf("expected default separator styles, got %#v", all["core/separator"])
	}
}

func TestRegisterBlockStylesMulti(t *testing.T) {
	r := wp.NewBlockStyleRegistry()
	ok := r.RegisterBlockStyles([]string{"core/quote", "core/pullquote"}, wp.BlockStyle{
		Name:  "plain",
		Label: "Plain",
	})
	if !ok {
		t.Fatal("multi register failed")
	}
	if !r.IsRegistered("core/quote", "plain") || !r.IsRegistered("core/pullquote", "plain") {
		t.Fatal("expected style on both blocks")
	}
}
