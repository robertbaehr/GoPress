package wp

import (
	"strings"
)

// protectedExcerptMessage matches get_the_excerpt for password-protected posts (P3.124).
const protectedExcerptMessage = "There is no excerpt because this is a protected post."

// HasExcerpt is has_excerpt() (P3.124).
func HasExcerpt(post *Post) bool {
	return post != nil && strings.TrimSpace(post.Excerpt) != ""
}

// GetTheExcerpt is get_the_excerpt() (P3.124 / P2.140).
// Password-protected posts return the protected message. Applies get_the_excerpt
// filter, then wp_trim_excerpt (P2.140) which generates from content when empty.
func GetTheExcerpt(c *Context, post *Post) string {
	if post == nil {
		return ""
	}
	if c != nil && PostPasswordRequired(c, c.Req, post) {
		return protectedExcerptMessage
	}
	excerpt := post.Excerpt
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_the_excerpt", excerpt, post); v != nil {
			if s, ok := v.(string); ok {
				excerpt = s
			}
		}
	}
	// P2.140: default WP wires wp_trim_excerpt onto get_the_excerpt @10.
	return WPTrimExcerpt(c, excerpt, post)
}

// TheExcerpt is the_excerpt() display path (P3.124): GetTheExcerpt → wpautop →
// the_excerpt filter.
func TheExcerpt(c *Context, post *Post) string {
	out := GetTheExcerpt(c, post)
	if strings.TrimSpace(out) == "" {
		return ""
	}
	// Default WP the_excerpt filter includes wpautop at priority 10.
	out = WpAutop(out, true)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("the_excerpt", out); v != nil {
			if s, ok := v.(string); ok {
				out = s
			}
		}
	}
	return out
}

// TrimExcerptFromContent is a wp_trim_excerpt subset for empty custom excerpts (P3.124).
// Strips tags / shortcodes / block comments lightly and keeps numWords (default 55).
// Prefer WPTrimExcerpt (P2.140) for filter-aware generation.
func TrimExcerptFromContent(content string, numWords int) string {
	if numWords <= 0 {
		numWords = 55
	}
	text := stripBlockComments(content)
	text = StripShortcodes(nil, text)
	text = stripTagsBasic(text)
	text = strings.Join(strings.Fields(text), " ")
	words := strings.Fields(text)
	if len(words) <= numWords {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:numWords], " ") + "…"
}

// stripTagsBasic removes HTML tags without entity decoding (excerpt tease).
func stripTagsBasic(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}
