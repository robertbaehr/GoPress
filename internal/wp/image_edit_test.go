package wp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestLoadImageToEditPathFull(t *testing.T) {
	c := mediaBoot(t)
	path := wp.LoadImageToEditPath(c, 5, "full")
	if !strings.HasSuffix(path, "2024/06/sample-photo-a.jpg") {
		t.Fatalf("full path=%q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing: %v", err)
	}
}

func TestLoadImageToEditPathFilter(t *testing.T) {
	c := mediaBoot(t)
	c.Hooks.AddFilter("load_image_to_edit_path", func(value any, _ ...any) any {
		return "/tmp/edited-override.jpg"
	}, 10, 3)
	path := wp.LoadImageToEditPath(c, 5, "full")
	if path != "/tmp/edited-override.jpg" {
		t.Fatalf("filter path=%q", path)
	}
}

func TestRecordImageEditBackupSizes(t *testing.T) {
	c := mediaBoot(t)
	t.Cleanup(func() {
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, 5, "_wp_attachment_backup_sizes", "")
	})
	backup, err := wp.RecordImageEditBackupSizes(c, 5)
	if err != nil {
		t.Fatal(err)
	}
	full, ok := backup["full-orig"].(map[string]any)
	if !ok {
		t.Fatalf("full-orig: %#v", backup)
	}
	if filepath.Base(fmt.Sprint(full["file"])) != "sample-photo-a.jpg" {
		t.Fatalf("file=%v", full["file"])
	}
	got := wp.GetAttachmentBackupSizes(c, 5)
	if got == nil || got["full-orig"] == nil {
		t.Fatalf("persisted backup missing: %#v", got)
	}
	// Idempotent: second call keeps existing full-orig.
	again, err := wp.RecordImageEditBackupSizes(c, 5)
	if err != nil {
		t.Fatal(err)
	}
	if again["full-orig"] == nil {
		t.Fatal("expected full-orig retained")
	}
}
