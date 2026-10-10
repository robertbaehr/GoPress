// Shortcode API (P2.11) — register / do_shortcode subset.
package wp

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// ShortcodeFunc renders a shortcode. attrs may be empty; content is inner text
// (empty for self-closing tags).
type ShortcodeFunc func(attrs map[string]string, content string) string

// Shortcodes is a process-wide shortcode tag registry.
type Shortcodes struct {
	mu   sync.RWMutex
	tags map[string]ShortcodeFunc
}

// NewShortcodes returns an empty registry.
func NewShortcodes() *Shortcodes {
	return &Shortcodes{tags: map[string]ShortcodeFunc{}}
}

// AddShortcode registers tag (last writer wins).
func (s *Shortcodes) AddShortcode(tag string, fn ShortcodeFunc) {
	if s == nil || fn == nil {
		return
	}
	tag = strings.TrimSpace(tag)
	if tag == "" || !validShortcodeTag(tag) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tags == nil {
		s.tags = map[string]ShortcodeFunc{}
	}
	s.tags[tag] = fn
}

// RemoveShortcode unregisters tag.
func (s *Shortcodes) RemoveShortcode(tag string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tags, tag)
}

// HasShortcode reports whether tag is registered.
func (s *Shortcodes) HasShortcode(tag string) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.tags[tag]
	return ok
}

var shortcodeFind = regexp.MustCompile(`\[([^<>&/\[\]\x00-\x20=]+)`)

// DoShortcode expands registered shortcodes in content.
func (s *Shortcodes) DoShortcode(content string) string {
	if s == nil || content == "" || !strings.Contains(content, "[") {
		return content
	}
	s.mu.RLock()
	tags := make(map[string]ShortcodeFunc, len(s.tags))
	for k, v := range s.tags {
		tags[k] = v
	}
	s.mu.RUnlock()
	if len(tags) == 0 {
		return content
	}
	found := shortcodeFind.FindAllStringSubmatch(content, -1)
	needed := map[string]bool{}
	for _, m := range found {
		if len(m) > 1 {
			if _, ok := tags[m[1]]; ok {
				needed[m[1]] = true
			}
		}
	}
	if len(needed) == 0 {
		return content
	}
	// Expand enclosing forms first, then self-closing / bare tags.
	for tag, fn := range tags {
		if !needed[tag] {
			continue
		}
		q := regexp.QuoteMeta(tag)
		enclosing := regexp.MustCompile(`(?s)\[` + q + `(\s[^\]]*)?\](.*?)\[/` + q + `\]`)
		content = enclosing.ReplaceAllStringFunc(content, func(match string) string {
			m := enclosing.FindStringSubmatch(match)
			attrText, inner := "", ""
			if len(m) > 1 {
				attrText = m[1]
			}
			if len(m) > 2 {
				inner = m[2]
			}
			return fn(ParseShortcodeAtts(attrText), inner)
		})
		selfClose := regexp.MustCompile(`\[` + q + `(\s[^\]]*)?/\s*\]`)
		content = selfClose.ReplaceAllStringFunc(content, func(match string) string {
			m := selfClose.FindStringSubmatch(match)
			attrText := ""
			if len(m) > 1 {
				attrText = m[1]
			}
			return fn(ParseShortcodeAtts(attrText), "")
		})
		bare := regexp.MustCompile(`\[` + q + `(\s[^\]]*)?\]`)
		content = bare.ReplaceAllStringFunc(content, func(match string) string {
			m := bare.FindStringSubmatch(match)
			attrText := ""
			if len(m) > 1 {
				attrText = m[1]
			}
			return fn(ParseShortcodeAtts(attrText), "")
		})
	}
	return content
}

// DoShortcode expands shortcodes using c.Shortcodes (nil-safe).
func DoShortcode(c *Context, content string) string {
	if c == nil || c.Shortcodes == nil {
		return content
	}
	return c.Shortcodes.DoShortcode(content)
}

// ParseShortcodeAtts parses shortcode attribute text into a map.
func ParseShortcodeAtts(text string) map[string]string {
	out := map[string]string{}
	text = strings.TrimSpace(text)
	if text == "" {
		return out
	}
	re := regexp.MustCompile(`([\w-]+)\s*=\s*"([^"]*)"|([\w-]+)\s*=\s*'([^']*)'|([\w-]+)\s*=\s*([^\s"'\]]+)`)
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		switch {
		case m[1] != "":
			out[strings.ToLower(m[1])] = m[2]
		case m[3] != "":
			out[strings.ToLower(m[3])] = m[4]
		case m[5] != "":
			out[strings.ToLower(m[5])] = m[6]
		}
	}
	return out
}

func validShortcodeTag(tag string) bool {
	for i := 0; i < len(tag); i++ {
		c := tag[i]
		if c == '<' || c == '>' || c == '&' || c == '/' || c == '[' || c == ']' || c <= ' ' || c == '=' {
			return false
		}
	}
	return true
}

// RegisterDefaultShortcodes installs gallery, caption, and embed handlers.
func RegisterDefaultShortcodes(s *Shortcodes) {
	if s == nil {
		return
	}
	s.AddShortcode("caption", shortcodeCaption)
	s.AddShortcode("wp_caption", shortcodeCaption)
	s.AddShortcode("gallery", shortcodeGallery)
	s.AddShortcode("embed", shortcodeEmbed)
}

func shortcodeCaption(attrs map[string]string, content string) string {
	caption := attrs["caption"]
	if caption == "" {
		// New-style: trailing text after <img> (optionally wrapped in <a>).
		re := regexp.MustCompile(`(?is)^((?:<a\b[^>]*>\s*)?<img\b[^>]*>(?:\s*</a>)?)(.*)$`)
		if m := re.FindStringSubmatch(content); len(m) == 3 {
			content = strings.TrimSpace(m[1])
			caption = strings.TrimSpace(m[2])
		}
	}
	align := attrs["align"]
	if align == "" {
		align = "alignnone"
	}
	width := attrs["width"]
	class := strings.TrimSpace("wp-caption " + align + " " + attrs["class"])
	idAttr := ""
	if id := attrs["id"]; id != "" {
		idAttr = ` id="` + html.EscapeString(id) + `"`
	}
	style := ""
	if w, err := strconv.Atoi(width); err == nil && w > 0 {
		style = fmt.Sprintf(` style="width: %dpx"`, w)
	}
	if caption == "" {
		return content
	}
	return fmt.Sprintf(`<figure%s%s class="%s">%s<figcaption class="wp-caption-text">%s</figcaption></figure>`,
		idAttr, style, html.EscapeString(class), content, html.EscapeString(caption))
}

func shortcodeGallery(attrs map[string]string, content string) string {
	ids := attrs["ids"]
	if ids == "" {
		ids = attrs["include"]
	}
	columns := attrs["columns"]
	if columns == "" {
		columns = "3"
	}
	var b strings.Builder
	b.WriteString(`<div class="gallery gallery-columns-`)
	b.WriteString(html.EscapeString(columns))
	b.WriteString(`">`)
	if ids != "" {
		for _, part := range strings.Split(ids, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			b.WriteString(`<figure class="gallery-item"><div class="gallery-icon">`)
			b.WriteString(`<!-- attachment:`)
			b.WriteString(html.EscapeString(part))
			b.WriteString(` -->`)
			b.WriteString(`</div></figure>`)
		}
	}
	b.WriteString(`</div>`)
	return b.String()
}

func shortcodeEmbed(attrs map[string]string, content string) string {
	src := strings.TrimSpace(content)
	if src == "" {
		src = strings.TrimSpace(attrs["src"])
	}
	if src == "" {
		return ""
	}
	// Basic stub: link wrapper (full oEmbed fetch is out of scope for P2.11).
	width := attrs["width"]
	if width == "" {
		width = "600"
	}
	return fmt.Sprintf(
		`<div class="wp-block-embed"><div class="wp-block-embed__wrapper"><a class="wp-embedded-content" href="%s" data-width="%s">%s</a></div></div>`,
		html.EscapeString(src), html.EscapeString(width), html.EscapeString(src),
	)
}
