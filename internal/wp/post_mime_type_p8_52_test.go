package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.52: GetPostMimeType for attachment fixture and non-attachment post.
func TestGetPostMimeTypeP852(t *testing.T) {
	c := mediaBoot(t)

	mime, ok := wp.GetPostMimeType(c, 5)
	if !ok || mime == "" {
		t.Fatalf("attachment 5 mime=%q ok=%v", mime, ok)
	}
	if !strings.HasPrefix(mime, "image/") {
		t.Fatalf("mime=%q want image/*", mime)
	}

	postMime, ok := wp.GetPostMimeType(c, 1)
	if !ok {
		t.Fatal("post 1 should load")
	}
	if postMime != "" {
		t.Fatalf("hello-world mime=%q want empty", postMime)
	}

	if _, ok := wp.GetPostMimeType(c, 0); ok {
		t.Fatal("zero id")
	}
	if _, ok := wp.GetPostMimeType(nil, 5); ok {
		t.Fatal("nil context")
	}
	if _, ok := wp.GetPostMimeType(c, 99999999); ok {
		t.Fatal("missing post")
	}
}
