package wp_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestEmojiPrintScriptsSkipLoader(t *testing.T) {
	if wp.ShouldPrintEmoji() {
		t.Fatal("emoji loader print should stay disabled")
	}
	if s := wp.PrintEmojiDetectionScript(); s != "" {
		t.Fatalf("detection/loader script should be empty, got %q", s)
	}
	if s := wp.PrintEmojiStyles(); s != "" {
		t.Fatalf("styles should be empty, got %q", s)
	}
}

func TestEmojiSettingsJSON(t *testing.T) {
	html := wp.PrintEmojiSettingsJSON()
	if !strings.Contains(html, `id="wp-emoji-settings"`) {
		t.Fatalf("missing settings id: %q", html)
	}
	if !strings.Contains(html, `type="application/json"`) {
		t.Fatalf("missing type: %q", html)
	}
	start := strings.Index(html, ">")
	end := strings.LastIndex(html, "</script>")
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("malformed script tag: %q", html)
	}
	payload := html[start+1 : end]
	var settings map[string]any
	if err := json.Unmarshal([]byte(payload), &settings); err != nil {
		t.Fatalf("json: %v payload=%q", err, payload)
	}
	if settings["baseUrl"] == "" || settings["ext"] != ".png" {
		t.Fatalf("unexpected settings: %#v", settings)
	}
	if settings["svgUrl"] == "" || settings["svgExt"] != ".svg" {
		t.Fatalf("unexpected svg settings: %#v", settings)
	}
	head := wp.EmojiHeadHTML()
	if !strings.Contains(head, `id="wp-emoji-settings"`) {
		t.Fatalf("EmojiHeadHTML should include settings JSON: %q", head)
	}
}
