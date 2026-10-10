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
