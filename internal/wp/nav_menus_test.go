package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestNavMenuLocationsRoundTrip(t *testing.T) {
	c := testBootQuery(t)
	t.Cleanup(func() {
		_ = wp.SetNavMenuLocation(c, "primary", 0)
		_ = wp.SetNavMenuLocation(c, "footer", 0)
	})

	registered := wp.RegisteredNavMenus(c)
	if registered["primary"] == "" || registered["footer"] == "" {
		t.Fatalf("registered=%v", registered)
	}
	if err := wp.SetNavMenuLocation(c, "primary", 37); err != nil {
		t.Fatal(err)
	}
	locs := wp.GetNavMenuLocations(c)
	if locs["primary"] != 37 {
		t.Fatalf("primary=%v", locs)
	}
	if err := wp.SetNavMenuLocation(c, "primary", 0); err != nil {
		t.Fatal(err)
	}
	locs = wp.GetNavMenuLocations(c)
	if locs["primary"] != 0 {
		t.Fatalf("cleared primary=%v", locs)
	}
	if err := wp.SetNavMenuLocation(c, "nope", 1); err == nil {
		t.Fatal("expected invalid location error")
	}
}
