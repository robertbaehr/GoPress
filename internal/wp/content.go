// the_content filter pipeline (P5.11) — blocks, shortcodes, texturize.
package wp

import (
	"strings"
	"unicode/utf8"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// TheContent runs the display pipeline for post_content (the_content subset):
// do_blocks → do_shortcode → wptexturize, then the "the_content" filter.
func TheContent(c *Context, raw string) string {
	out := ApplyContentFilters(c, raw)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("the_content", out); v != nil {
			if s, ok := v.(string); ok {
				out = s
			}
		}
	}
	return out
}

// ApplyContentFilters runs blocks + shortcodes + texturize without the hook.
func ApplyContentFilters(c *Context, raw string) string {
	if raw == "" {
		return ""
	}
	out := blocks.Render(raw)
	out = DoShortcode(c, out)
	out = Texturize(out)
	return out
}

// Texturize is a small wptexturize subset used by REST and front the_content.
func Texturize(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s) + 16)
	inTag := false
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '<' {
			inTag = true
			b.WriteRune(r)
			i += size
			continue
		}
		if r == '>' && inTag {
			inTag = false
			b.WriteRune(r)
			i += size
			continue
		}
		if inTag {
			b.WriteRune(r)
			i += size
			continue
		}
		if r == '.' && i+2 < len(s) && s[i:i+3] == "..." {
			b.WriteString("&#8230;")
			i += 3
			continue
		}
		if r == '\'' {
			b.WriteString("&#8217;")
			i += size
			continue
		}
		b.WriteRune(r)
		i += size
	}
	return b.String()
}
