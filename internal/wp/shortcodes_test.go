package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestDoShortcodeCaption(t *testing.T) {
	s := wp.NewShortcodes()
	wp.RegisterDefaultShortcodes(s)
	in := `[caption id="attachment_1" align="alignnone" width="300"]<img src="x.jpg" alt="" /> Hello[/caption]`
	out := s.DoShortcode(in)
	if !strings.Contains(out, `<figure`) || !strings.Contains(out, `wp-caption`) {
		t.Fatalf("got %q", out)
	}
	if !strings.Contains(out, `Hello`) || !strings.Contains(out, `figcaption`) {
		t.Fatalf("missing caption text: %q", out)
	}
	if strings.Contains(out, "[caption") {
		t.Fatalf("shortcode not expanded: %q", out)
	}
}

func TestDoShortcodeGallery(t *testing.T) {
	s := wp.NewShortcodes()
	wp.RegisterDefaultShortcodes(s)
	out := s.DoShortcode(`[gallery ids="10,20" columns="2"]`)
	if !strings.Contains(out, `gallery-columns-2`) {
		t.Fatalf("got %q", out)
	}
	if !strings.Contains(out, `attachment:10`) || !strings.Contains(out, `attachment:20`) {
		t.Fatalf("got %q", out)
	}
}

func TestDoShortcodeEmbed(t *testing.T) {
	s := wp.NewShortcodes()
	wp.RegisterDefaultShortcodes(s)
	out := s.DoShortcode(`[embed]https://example.com/watch?v=1[/embed]`)
	if !strings.Contains(out, `https://example.com/watch?v=1`) {
		t.Fatalf("got %q", out)
	}
	if !strings.Contains(out, `wp-block-embed`) {
		t.Fatalf("got %q", out)
	}
}

func TestDoShortcodeCustomRegister(t *testing.T) {
	s := wp.NewShortcodes()
	s.AddShortcode("hello", func(attrs map[string]string, content string) string {
		return "Hi " + attrs["name"]
	})
	out := s.DoShortcode(`Say [hello name="Bob"] please`)
	if out != `Say Hi Bob please` {
		t.Fatalf("got %q", out)
	}
}

func TestParseShortcodeAtts(t *testing.T) {
	attrs := wp.ParseShortcodeAtts(` ids="1,2" columns='3' size=thumbnail`)
	if attrs["ids"] != "1,2" || attrs["columns"] != "3" || attrs["size"] != "thumbnail" {
		t.Fatalf("%#v", attrs)
	}
}

func TestBootRegistersDefaultShortcodes(t *testing.T) {
	c := bootWP(t)
	if c.Shortcodes == nil || !c.Shortcodes.HasShortcode("gallery") {
		t.Fatal("gallery shortcode not registered on Boot")
	}
}

func TestShortcodeAttsDefaultsAndOverride(t *testing.T) {
	// P9.22: shortcode_atts merges defaults.
	pairs := map[string]string{"ids": "", "columns": "3", "size": "thumbnail"}
	atts := map[string]string{"ids": "1,2", "columns": "2"}
	out := wp.ShortcodeAtts(pairs, atts)
	if out["ids"] != "1,2" || out["columns"] != "2" || out["size"] != "thumbnail" {
		t.Fatalf("%#v", out)
	}
	if _, ok := out["unknown"]; ok {
		t.Fatal("unexpected key")
	}
}

func TestShortcodeAttsFiltered(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("shortcode_atts_gallery", func(value any, args ...any) any {
		out, _ := value.(map[string]string)
		if out == nil {
			out = map[string]string{}
		}
		out["columns"] = "4"
		return out
	}, 10, 4)
	pairs := map[string]string{"columns": "3", "ids": ""}
	out := wp.ShortcodeAttsFiltered(c, pairs, map[string]string{"ids": "9"}, "gallery")
	if out["columns"] != "4" || out["ids"] != "9" {
		t.Fatalf("%#v", out)
	}
}

func TestStripShortcodes(t *testing.T) {
	// P9.22: strip_shortcodes removes registered tags, keeps unknown.
	s := wp.NewShortcodes()
	wp.RegisterDefaultShortcodes(s)
	in := `Before [gallery ids="1"] mid [caption]x[/caption] after [unknown]y[/unknown]`
	out := s.StripShortcodes(in)
	if strings.Contains(out, "[gallery") || strings.Contains(out, "[caption") {
		t.Fatalf("registered tags remain: %q", out)
	}
	if !strings.Contains(out, "[unknown]") || !strings.Contains(out, "Before") || !strings.Contains(out, "after") {
		t.Fatalf("got %q", out)
	}
}

func TestStripShortcodesTagNamesFilter(t *testing.T) {
	c := bootWP(t)
	c.Hooks.AddFilter("strip_shortcodes_tagnames", func(value any, args ...any) any {
		return []string{"gallery"}
	}, 10, 2)
	in := `[gallery ids="1"] [caption]keep[/caption]`
	out := wp.StripShortcodes(c, in)
	if strings.Contains(out, "[gallery") {
		t.Fatalf("gallery not stripped: %q", out)
	}
	if !strings.Contains(out, "[caption]") {
		t.Fatalf("caption should remain when filtered out of tagnames: %q", out)
	}
}
