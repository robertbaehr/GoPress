package wp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestMkdirPCreatesNested(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "a", "b", "c")
	if !wp.MkdirP(target) {
		t.Fatal("MkdirP returned false")
	}
	fi, err := os.Stat(target)
	if err != nil || !fi.IsDir() {
		t.Fatalf("stat: %v fi=%v", err, fi)
	}
	if !wp.MkdirP(target) {
		t.Fatal("second MkdirP should succeed")
	}
	// Trailing slash + collapsed slashes.
	if !wp.MkdirP(base + "/d//e/") {
		t.Fatal("slash-normalized MkdirP failed")
	}
}

func TestMkdirPRejectsTraversal(t *testing.T) {
	if wp.MkdirP("foo/../../bar") {
		t.Fatal("expected traversal rejection")
	}
	if wp.MkdirP("..") {
		t.Fatal("expected .. rejection")
	}
	if wp.MkdirP("../outside") {
		t.Fatal("expected ../outside rejection")
	}
}

func TestMkdirPExistingFileIsFalse(t *testing.T) {
	base := t.TempDir()
	file := filepath.Join(base, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if wp.MkdirP(file) {
		t.Fatal("MkdirP on existing file should be false")
	}
}

func TestMkdirPEmptyFalse(t *testing.T) {
	if wp.MkdirP("") {
		t.Fatal("empty should be false")
	}
}
