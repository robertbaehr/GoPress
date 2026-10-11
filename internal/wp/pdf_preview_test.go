package wp_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGenerateAttachmentMetadataPDFPreview(t *testing.T) {
	// P8.34: PDF uploads get a *-pdf.jpg preview and fallback sizes.
	if _, err := exec.LookPath("gs"); err != nil {
		t.Skip("ghostscript (gs) not available")
	}
	c := mediaBoot(t)
	src := filepath.Join("..", "..", "tests", "fixtures", "media", "sample-doc.pdf")
	if _, err := os.Stat(src); err != nil {
		src = filepath.Join("tests", "fixtures", "media", "sample-doc.pdf")
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture pdf: %v", err)
	}
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "note.pdf")
	if err := os.WriteFile(pdfPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	var fallbackSeen bool
	c.Hooks.AddFilter("fallback_intermediate_image_sizes", func(value any, args ...any) any {
		fallbackSeen = true
		return []string{"thumbnail", "medium"}
	}, 10, 2)

	meta := wp.GenerateAttachmentMetadata(c, 42, pdfPath)
	if !fallbackSeen {
		t.Fatal("expected fallback_intermediate_image_sizes filter")
	}
	if meta["filesize"] == nil {
		t.Fatalf("expected pdf filesize: %#v", meta)
	}
	sizes, ok := meta["sizes"].(map[string]any)
	if !ok || sizes["full"] == nil {
		t.Fatalf("expected sizes.full preview: %#v", meta["sizes"])
	}
	full, _ := sizes["full"].(map[string]any)
	file, _ := full["file"].(string)
	if file == "" || filepath.Ext(file) != ".jpg" {
		t.Fatalf("preview file=%q", file)
	}
	if _, err := os.Stat(filepath.Join(dir, file)); err != nil {
		t.Fatalf("preview missing: %v", err)
	}
	if sizes["thumbnail"] == nil || sizes["medium"] == nil {
		t.Fatalf("expected thumbnail+medium: %#v", sizes)
	}
	if sizes["large"] != nil {
		t.Fatalf("filter omitted large; got %#v", sizes["large"])
	}
	w, _ := full["width"].(int)
	h, _ := full["height"].(int)
	if w < 100 || h < 50 {
		t.Fatalf("unexpected preview dimensions %dx%d", w, h)
	}
}
