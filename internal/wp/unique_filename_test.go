package wp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestUniqueFilenameBasic(t *testing.T) {
	dir := t.TempDir()
	got := wp.UniqueFilename(dir, "photo.jpg")
	if got != "photo.jpg" {
		t.Fatalf("got %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = wp.UniqueFilename(dir, "photo.jpg")
	if got != "photo-1.jpg" {
		t.Fatalf("got %q want photo-1.jpg", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "photo-1.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = wp.UniqueFilename(dir, "photo.jpg")
	if got != "photo-2.jpg" {
		t.Fatalf("got %q want photo-2.jpg", got)
	}
}

func TestUniqueFilenameSubsizeSuffix(t *testing.T) {
	dir := t.TempDir()
	got := wp.UniqueFilename(dir, "image-150x150.png")
	if got != "image-150x150-1.png" {
		t.Fatalf("got %q want image-150x150-1.png", got)
	}
	got = wp.UniqueFilename(dir, "hero-scaled.jpg")
	if got != "hero-scaled-1.jpg" {
		t.Fatalf("got %q want hero-scaled-1.jpg", got)
	}
	got = wp.UniqueFilename(dir, "shot-rotated.webp")
	if got != "shot-rotated-1.webp" {
		t.Fatalf("got %q want shot-rotated-1.webp", got)
	}
}

func TestUniqueFilenameLowercasesExt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "doc.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := wp.UniqueFilename(dir, "doc.PDF")
	if got != "doc-1.pdf" {
		t.Fatalf("got %q want doc-1.pdf", got)
	}
}

func TestUniqueFilenameSanitizes(t *testing.T) {
	dir := t.TempDir()
	got := wp.UniqueFilename(dir, "My Photo (1).JPG")
	if got == "" || got == "My Photo (1).JPG" {
		t.Fatalf("expected sanitized name, got %q", got)
	}
	if filepath.Ext(got) != ".jpg" {
		t.Fatalf("expected lowercase ext, got %q", got)
	}
}
