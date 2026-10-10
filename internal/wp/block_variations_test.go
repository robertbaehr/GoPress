package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestBlockVariationRegistryRegisterGetUnregister(t *testing.T) {
	r := wp.NewBlockVariationRegistry()
	if !r.RegisterBlockVariation("core/group", wp.BlockVariation{Name: "row", Title: "Row"}) {
		t.Fatal("register failed")
	}
	if !r.IsRegistered("core/group", "row") {
		t.Fatal("expected row registered")
	}
	got, ok := r.Get("core/group", "row")
	if !ok || got.Title != "Row" {
		t.Fatalf("get: %#v ok=%v", got, ok)
	}
	if r.RegisterBlockVariation("core/group", wp.BlockVariation{Name: "has space"}) {
		t.Fatal("spaces in variation name must fail")
	}
	if !r.UnregisterBlockVariation("core/group", "row") {
		t.Fatal("unregister failed")
	}
	if r.IsRegistered("core/group", "row") {
		t.Fatal("expected unregistered")
	}
}

func TestContextRegisterBlockVariationAndDefaults(t *testing.T) {
	c := bootWP(t)
	if c.BlockVariations == nil {
		t.Fatal("Boot should wire BlockVariations")
	}
	if !c.BlockVariations.IsRegistered("core/group", "row") {
		t.Fatal("default group/row missing")
	}
	if !c.RegisterBlockVariation("core/embed", wp.BlockVariation{Name: "youtube", Title: "YouTube"}) {
		t.Fatal("context register failed")
	}
	vars := c.BlockVariations.ForBlock("core/embed")
	if len(vars) != 1 || vars[0].Name != "youtube" {
		t.Fatalf("ForBlock: %#v", vars)
	}
	if !c.UnregisterBlockVariation("core/embed", "youtube") {
		t.Fatal("context unregister failed")
	}
	all := c.BlockVariations.All()
	if len(all["core/group"]) < 2 {
		t.Fatalf("expected default group variations, got %#v", all["core/group"])
	}
}
