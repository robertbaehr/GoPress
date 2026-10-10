package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestEmojiPrintScriptsSkip(t *testing.T) {
	if wp.ShouldPrintEmoji() {
		t.Fatal("emoji print should be disabled")
	}
	if s := wp.PrintEmojiDetectionScript(); s != "" {
		t.Fatalf("detection script should be empty, got %q", s)
	}
	if s := wp.PrintEmojiStyles(); s != "" {
		t.Fatalf("styles should be empty, got %q", s)
	}
	if s := wp.EmojiHeadHTML(); s != "" {
		t.Fatalf("head html should be empty, got %q", s)
	}
	settings := wp.EmojiSettings()
	if settings["baseUrl"] != "" || settings["ext"] != "" {
		t.Fatalf("expected empty CDN settings, got %#v", settings)
	}
	src, _ := settings["source"].(map[string]any)
	if src == nil || src["wpemoji"] != "" {
		t.Fatalf("expected empty source, got %#v", settings)
	}
}
