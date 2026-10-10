package wp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetImageMimeMagic(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"a.jpg", []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10}, "image/jpeg"},
		{"a.png", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}, "image/png"},
		{"a.gif", []byte("GIF89a......"), "image/gif"},
		{"a.webp", append([]byte("RIFF\x00\x00\x00\x00WEBP"), 0, 0, 0, 0), "image/webp"},
		{"a.avif", []byte{0, 0, 0, 0x14, 'f', 't', 'y', 'p', 'a', 'v', 'i', 'f', 0, 0, 0, 0}, "image/avif"},
		{"a.heic", []byte{0, 0, 0, 0x14, 'f', 't', 'y', 'p', 'h', 'e', 'i', 'c', 0, 0, 0, 0}, "image/heic"},
		{"a.txt", []byte("not an image"), ""},
	}
	for _, tc := range cases {
		path := filepath.Join(dir, tc.name)
		if err := os.WriteFile(path, tc.data, 0o644); err != nil {
			t.Fatal(err)
		}
		got := wp.GetImageMime(path)
		if got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
	if got := wp.GetImageMime(filepath.Join(dir, "missing.bin")); got != "" {
		t.Fatalf("missing file: got %q", got)
	}
	if got := wp.GetImageMime(""); got != "" {
		t.Fatalf("empty path: got %q", got)
	}
}
