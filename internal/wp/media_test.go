package wp_test

import (
	"context"
	"fmt"
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
	t.Cleanup(func() { _ = wp.RemoveImageSize("gopress-test") })
	s, ok := wp.GetImageSize(c, "gopress-test")
	if !ok || s.Width != 120 || s.Height != 80 || !s.Crop {
		t.Fatalf("got %+v ok=%v", s, ok)
	}
	if !wp.HasImageSize("gopress-test") {
		t.Fatal("HasImageSize expected true for additional size")
	}
	if wp.HasImageSize("thumbnail") {
		t.Fatal("HasImageSize is additional-only; thumbnail should be false")
	}
}

func TestIntermediateImageSizesAndNamesChoose(t *testing.T) {
	c := mediaBoot(t)
	wp.AddImageSize("gopress-editor", 200, 100, false)
	t.Cleanup(func() { _ = wp.RemoveImageSize("gopress-editor") })

	names := wp.GetIntermediateImageSizes(c)
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"thumbnail", "medium", "medium_large", "large", "gopress-editor"} {
		if !found[want] {
			t.Fatalf("missing intermediate size %q in %v", want, names)
		}
	}

	c.Hooks.AddFilter("intermediate_image_sizes", func(value any, args ...any) any {
		return []string{"thumbnail", "gopress-editor"}
	}, 10, 1)
	filtered := wp.GetIntermediateImageSizes(c)
	if len(filtered) != 2 || filtered[0] != "thumbnail" || filtered[1] != "gopress-editor" {
		t.Fatalf("filter not applied: %v", filtered)
	}

	labels := wp.ImageSizeNamesChoose(c)
	if labels["thumbnail"] != "Thumbnail" || labels["full"] != "Full Size" {
		t.Fatalf("labels=%v", labels)
	}
	c.Hooks.AddFilter("image_size_names_choose", func(value any, args ...any) any {
		return map[string]string{"thumbnail": "Tiny", "full": "Full Size"}
	}, 10, 1)
	labels = wp.ImageSizeNamesChoose(c)
	if labels["thumbnail"] != "Tiny" {
		t.Fatalf("names_choose filter: %v", labels)
	}

	if !wp.RemoveImageSize("gopress-editor") {
		t.Fatal("expected RemoveImageSize true")
	}
	if wp.RemoveImageSize("gopress-editor") {
		t.Fatal("expected RemoveImageSize false when missing")
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

func TestPostThumbnailHelpers(t *testing.T) {
	c := mediaBoot(t)
	postID := int64(13)
	prev := fmt.Sprint(c.Meta.GetMetadata(wp.MetaPost, postID, "_thumbnail_id", true))
	if prev == "<nil>" {
		prev = ""
	}
	t.Cleanup(func() {
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "_thumbnail_id", "")
		if prev != "" {
			_, _, _ = c.Meta.AddMetadata(wp.MetaPost, postID, "_thumbnail_id", prev, true)
		}
	})
	_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "_thumbnail_id", "")
	if _, _, err := c.Meta.AddMetadata(wp.MetaPost, postID, "_thumbnail_id", "5", true); err != nil {
		t.Fatalf("set thumbnail: %v", err)
	}
	if got := wp.GetPostThumbnailID(c, postID); got != 5 {
		t.Fatalf("thumbnail id=%d want 5", got)
	}
	url := wp.GetPostThumbnailURL(c, postID)
	if !strings.Contains(url, "sample-photo-a") {
		t.Fatalf("thumbnail url=%q", url)
	}
}

func TestResizeImageStub(t *testing.T) {
	_, err := wp.ResizeImage("/tmp/x.jpg", wp.ImageSize{Name: "thumbnail", Width: 150, Height: 150})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestAttachmentImageSrcAndHTML(t *testing.T) {
	// P8.32: wp_get_attachment_image_src / wp_get_attachment_image.
	c := mediaBoot(t)
	src, ok := wp.GetAttachmentImageSrc(c, 5, "thumbnail", false)
	if !ok {
		t.Fatal("expected thumbnail src")
	}
	if !strings.Contains(src.URL, "sample-photo-a-150x150") || src.Width != 150 || src.Height != 150 || !src.Resized {
		t.Fatalf("src=%+v", src)
	}
	full, ok := wp.GetAttachmentImageSrc(c, 5, "full", false)
	if !ok || !strings.Contains(full.URL, "sample-photo-a.jpg") || full.Resized {
		t.Fatalf("full=%+v", full)
	}
	if u := wp.GetAttachmentImageURL(c, 5, "medium", false); !strings.Contains(u, "300x225") {
		t.Fatalf("medium url=%q", u)
	}
	html := wp.GetAttachmentImage(c, 5, "thumbnail", false, nil)
	for _, want := range []string{
		`<img src="`,
		"sample-photo-a-150x150",
		`class="attachment-thumbnail size-thumbnail"`,
		`width="150"`,
		`height="150"`,
		`loading="lazy"`,
		`decoding="async"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
	html = wp.GetAttachmentImage(c, 5, "thumbnail", false, map[string]string{
		"class": "custom", "alt": "A", "loading": "eager",
	})
	if !strings.Contains(html, `class="custom"`) || !strings.Contains(html, `alt="A"`) || !strings.Contains(html, `loading="eager"`) {
		t.Fatalf("attr html=%q", html)
	}
}

func TestPrependAttachmentAndPagesOption(t *testing.T) {
	c := mediaBoot(t)
	if wp.AttachmentPagesEnabled(c) {
		t.Fatal("fixture should have wp_attachment_pages_enabled=0")
	}
	html := wp.PrependAttachment(c, 5, "")
	if !strings.Contains(html, `class="attachment"`) {
		t.Fatalf("missing attachment wrapper: %q", html)
	}
	if !strings.Contains(html, "<img ") || !strings.Contains(html, "sample-photo-a") {
		t.Fatalf("expected medium img link: %q", html)
	}
	if !strings.Contains(html, `<a href="`) {
		t.Fatalf("expected file link: %q", html)
	}
	withBody := wp.PrependAttachment(c, 5, "<p>caption text</p>")
	if !strings.Contains(withBody, "caption text") || !strings.HasPrefix(strings.TrimSpace(withBody), `<p class="attachment">`) {
		t.Fatalf("prepend+content=%q", withBody)
	}
}

func TestGetAttachmentMetadata(t *testing.T) {
	c := mediaBoot(t)
	meta, ok := wp.GetAttachmentMetadata(c, 5)
	if !ok {
		t.Fatal("expected metadata for attachment 5")
	}
	if fmt.Sprint(meta["file"]) != "2024/06/sample-photo-a.jpg" {
		t.Fatalf("file=%v", meta["file"])
	}
	if meta["width"] == nil || meta["height"] == nil {
		t.Fatalf("missing dimensions: %#v", meta)
	}
	sizes, ok := meta["sizes"].(map[string]any)
	if !ok || len(sizes) == 0 {
		// sizes may be map[any]any from unserialize path
		if _, ok2 := meta["sizes"].(map[any]any); !ok2 {
			t.Fatalf("sizes=%T %#v", meta["sizes"], meta["sizes"])
		}
	}

	c.Hooks.AddFilter("wp_get_attachment_metadata", func(value any, args ...any) any {
		m, _ := value.(map[string]any)
		if m == nil {
			return value
		}
		m["gopress_filtered"] = true
		return m
	}, 10, 2)
	meta, ok = wp.GetAttachmentMetadata(c, 5)
	if !ok || meta["gopress_filtered"] != true {
		t.Fatalf("filter not applied: ok=%v %#v", ok, meta)
	}
	unf, ok := wp.GetAttachmentMetadataUnfiltered(c, 5)
	if !ok || unf["gopress_filtered"] == true {
		t.Fatalf("unfiltered should skip filter: %#v", unf)
	}
	if _, ok := wp.GetAttachmentMetadata(c, 999999); ok {
		t.Fatal("missing attachment should be false")
	}
}
