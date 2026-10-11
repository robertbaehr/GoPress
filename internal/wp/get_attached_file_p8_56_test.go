package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.56: get_attached_file filter can replace the path.
func TestGetAttachedFileFilterP856(t *testing.T) {
	c := testBootQuery(t)
	// Restore fixture path in case a prior test mutated _wp_attached_file.
	_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaPost, 5, "_wp_attached_file", "2024/06/sample-photo-a.jpg", "")
	path := wp.GetAttachedFileUnfiltered(c, 5)
	if path == "" || !strings.Contains(path, "sample-photo-a.jpg") {
		t.Fatalf("default=%q", path)
	}
	c.Hooks.AddFilter("get_attached_file", func(value any, _ ...any) any {
		return "/filtered/path.jpg"
	}, 10, 2)
	if got := wp.GetAttachedFile(c, 5); got != "/filtered/path.jpg" {
		t.Fatalf("filtered=%q", got)
	}
	unf := wp.GetAttachedFileUnfiltered(c, 5)
	if unf == "/filtered/path.jpg" || !strings.Contains(unf, "sample-photo-a.jpg") {
		t.Fatalf("unfiltered=%q", unf)
	}
}
