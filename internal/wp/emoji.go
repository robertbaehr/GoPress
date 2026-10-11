// Emoji settings JSON (P3.49 / P3.136) + print stubs (P3.12).
package wp

import "encoding/json"

const (
	emojiPNGBase = "https://s.w.org/images/core/emoji/17.0.2/72x72/"
	emojiSVGBase = "https://s.w.org/images/core/emoji/17.0.2/svg/"
)

// EmojiSettings returns the wp-emoji-settings object without filters (compat).
func EmojiSettings() map[string]any {
	return EmojiSettingsCtx(nil)
}

// EmojiSettingsCtx returns settings with emoji_url / emoji_ext / emoji_svg_url /
// emoji_svg_ext filters applied (_print_emoji_detection_script, P3.136).
// GoPress does not ship the emoji loader module; source URLs stay empty.
func EmojiSettingsCtx(c *Context) map[string]any {
	baseURL := emojiPNGBase
	ext := ".png"
	svgURL := emojiSVGBase
	svgExt := ".svg"
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("emoji_url", baseURL); v != nil {
			if s, ok := v.(string); ok && s != "" {
				baseURL = s
			}
		}
		if v := c.Hooks.ApplyFilters("emoji_ext", ext); v != nil {
			if s, ok := v.(string); ok && s != "" {
				ext = s
			}
		}
		if v := c.Hooks.ApplyFilters("emoji_svg_url", svgURL); v != nil {
			if s, ok := v.(string); ok && s != "" {
				svgURL = s
			}
		}
		if v := c.Hooks.ApplyFilters("emoji_svg_ext", svgExt); v != nil {
			if s, ok := v.(string); ok && s != "" {
				svgExt = s
			}
		}
	}
	return map[string]any{
		"baseUrl": baseURL,
		"ext":     ext,
		"svgUrl":  svgURL,
		"svgExt":  svgExt,
		"source": map[string]any{
			"concatemoji": "",
			"wpemoji":     "",
			"twemoji":     "",
		},
	}
}

// ShouldPrintEmoji reports whether emoji detection scripts should be printed.
// Settings JSON is printed (P3.49); the loader module remains skipped (P3.12).
func ShouldPrintEmoji() bool {
	return false
}

// PrintEmojiSettingsJSON returns settings JSON without context filters.
func PrintEmojiSettingsJSON() string {
	return PrintEmojiSettingsJSONCtx(nil)
}

// PrintEmojiSettingsJSONCtx returns the application/json script tag with id
// wp-emoji-settings (P3.49 / P3.136 / _print_emoji_detection_script settings half).
func PrintEmojiSettingsJSONCtx(c *Context) string {
	raw, err := json.Marshal(EmojiSettingsCtx(c))
	if err != nil {
		return ""
	}
	return `<script type="application/json" id="wp-emoji-settings">` + string(raw) + `</script>`
}

// PrintEmojiDetectionScript is a no-op stand-in for the emoji loader module.
// WP prints this as a module in the footer; GoPress keeps it empty (P3.12 / P3.136).
func PrintEmojiDetectionScript() string {
	return ""
}

// PrintEmojiStyles is a no-op stand-in for print_emoji_styles.
func PrintEmojiStyles() string {
	return ""
}

// EmojiHeadHTML returns head markup without context filters.
func EmojiHeadHTML() string {
	return EmojiHeadHTMLCtx(nil)
}

// EmojiHeadHTMLCtx returns head markup: filtered settings JSON only (no loader/styles).
// P3.136: applies emoji_* URL/ext filters via EmojiSettingsCtx.
func EmojiHeadHTMLCtx(c *Context) string {
	return PrintEmojiSettingsJSONCtx(c)
}
