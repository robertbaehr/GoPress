package wp

import "strings"

// WPStripAllTags mirrors wp_strip_all_tags (P2.31).
//
// Removes <script>/<style> blocks (including their contents), then all HTML
// tags. When removeBreaks is true, collapses runs of whitespace to a single
// space (WP style-engine / safecss path).
func WPStripAllTags(text string, removeBreaks bool) string {
	text = strings.ReplaceAll(text, "\x00", "")
	// PHP: preg_replace( '@<(script|style)[^>]*?>.*?</\\1>@si', '', $text )
	text = reScriptOrStyleBlock.ReplaceAllString(text, "")
	text = reTags.ReplaceAllString(text, "")
	if removeBreaks {
		text = reWhitespace.ReplaceAllString(text, " ")
	}
	return strings.TrimSpace(text)
}
