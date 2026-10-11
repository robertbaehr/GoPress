package wp_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.55: UpdateAttachedFile stores relative path under uploads basedir.
func TestUpdateAttachedFileP855(t *testing.T) {
	c := testBootQuery(t)
	uploads := wp.GetUploadDir(c)
	abs := filepath.Join(uploads.Basedir, "2024/06/updated-p855.jpg")
	prev := wp.GetAttachedFile(c, 5)
	t.Cleanup(func() {
		if prev != "" {
			_ = wp.UpdateAttachedFile(c, 5, prev)
		}
	})

	if !wp.UpdateAttachedFile(c, 5, abs) {
		t.Fatal("update failed")
	}
	got, _ := c.Meta.GetMetadata(wp.MetaPost, 5, "_wp_attached_file", true).(string)
	if got != "2024/06/updated-p855.jpg" {
		t.Fatalf("meta=%q", got)
	}
	if path := wp.GetAttachedFile(c, 5); !strings.HasSuffix(path, "updated-p855.jpg") {
		t.Fatalf("path=%q", path)
	}

	c.Hooks.AddFilter("update_attached_file", func(value any, _ ...any) any {
		return filepath.Join(uploads.Basedir, "filtered-p855.png")
	}, 10, 2)
	if !wp.UpdateAttachedFile(c, 5, abs) {
		t.Fatal("filtered update failed")
	}
	got, _ = c.Meta.GetMetadata(wp.MetaPost, 5, "_wp_attached_file", true).(string)
	if got != "filtered-p855.png" {
		t.Fatalf("filtered meta=%q", got)
	}
}

// P8.55: missing attachment / empty path guards.
func TestUpdateAttachedFileGuardsP855(t *testing.T) {
	c := testBootQuery(t)
	if wp.UpdateAttachedFile(c, 999999, "/tmp/x.jpg") {
		t.Fatal("missing attachment should fail")
	}
	if wp.UpdateAttachedFile(nil, 5, "/tmp/x.jpg") {
		t.Fatal("nil context")
	}
	if wp.RelativeUploadPath(c, "") != "" {
		t.Fatal("empty relative")
	}
}
