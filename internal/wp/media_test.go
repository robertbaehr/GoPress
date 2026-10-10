package wp_test

import (
	"context"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func mediaBoot(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH: "/workspace/",
		AuthKey: "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	return c
}

func TestRegisteredImageSizesFromOptions(t *testing.T) {
	c := mediaBoot(t)
	sizes := wp.GetRegisteredImageSizes(c)
	for _, name := range []string{"thumbnail", "medium", "medium_large", "large"} {
		s, ok := sizes[name]
		if !ok {
			t.Fatalf("missing size %q", name)
		}
		if s.Width <= 0 && s.Height <= 0 {
			t.Fatalf("%s has no dimensions: %+v", name, s)
		}
	}
	if sizes["thumbnail"].Width != 150 || sizes["medium"].Width != 300 {
		t.Fatalf("unexpected defaults from fixture options: thumb=%+v medium=%+v", sizes["thumbnail"], sizes["medium"])
	}
}

func TestAddImageSize(t *testing.T) {
	c := mediaBoot(t)
	wp.AddImageSize("gopress-test", 120, 80, true)
	t.Cleanup(func() { wp.RemoveImageSize("gopress-test") })
	s, ok := wp.GetImageSize(c, "gopress-test")
	if !ok || s.Width != 120 || s.Height != 80 || !s.Crop {
		t.Fatalf("got %+v ok=%v", s, ok)
	}
}

func TestGetAttachmentURL(t *testing.T) {
	c := mediaBoot(t)
	url := wp.GetAttachmentURL(c, 5)
	if url == "" {
		t.Fatal("empty attachment URL")
	}
	if !strings.Contains(url, "sample-photo-a") {
		t.Fatalf("url=%q", url)
	}
	if !strings.Contains(url, "wp-content/uploads") {
		t.Fatalf("expected uploads path in %q", url)
	}
}

func TestGetAttachedFile(t *testing.T) {
	c := mediaBoot(t)
	file := wp.GetAttachedFile(c, 5)
	if !strings.HasSuffix(file, "2024/06/sample-photo-a.jpg") {
		t.Fatalf("file=%q", file)
	}
}

func TestResizeImageStub(t *testing.T) {
	_, err := wp.ResizeImage("/tmp/x.jpg", wp.ImageSize{Name: "thumbnail", Width: 150, Height: 150})
	if err == nil {
		t.Fatal("expected not-implemented error")
	}
}
