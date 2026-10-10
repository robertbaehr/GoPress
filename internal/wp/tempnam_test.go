package wp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestTempNamCreatesUniqueFile(t *testing.T) {
	dir := t.TempDir()
	path := wp.TempNam("photo.jpg", dir)
	if path == "" {
		t.Fatal("empty path")
	}
	if !strings.HasPrefix(path, dir) {
		t.Fatalf("path %q not under %q", path, dir)
	}
	if !strings.HasSuffix(path, ".tmp") {
		t.Fatalf("expected .tmp suffix: %q", path)
	}
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "photo-") {
		t.Fatalf("expected photo- prefix: %q", base)
	}
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		t.Fatalf("expected created file: %v %#v", err, fi)
	}
	// Second call should not collide.
	path2 := wp.TempNam("photo.jpg", dir)
	if path2 == path {
		t.Fatalf("collision: %q", path)
	}
	_ = os.Remove(path)
	_ = os.Remove(path2)
}

func TestTempNamEmptyFilename(t *testing.T) {
	dir := t.TempDir()
	path := wp.TempNam("", dir)
	if path == "" || !strings.HasSuffix(path, ".tmp") {
		t.Fatalf("got %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(path)
}

func TestTempNamDefaultDir(t *testing.T) {
	path := wp.TempNam("x.txt", "")
	if path == "" {
		t.Fatal("empty")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(path)
}

func TestGetTempDirTrailingSlash(t *testing.T) {
	d := wp.GetTempDir()
	if d == "" {
		t.Fatal("empty")
	}
	if !strings.HasSuffix(d, "/") && !strings.HasSuffix(d, string(os.PathSeparator)) {
		t.Fatalf("expected trailing slash: %q", d)
	}
}
