// Emoji print stubs (P3.12) — skip wpemoji detection scripts/styles.
package wp

// EmojiSettings returns the _wpemojiSettings object with printing disabled.
// GoPress does not ship emoji CDN polyfills; front renders skip print scripts.
func EmojiSettings() map[string]any {
	return map[string]any{
		"baseUrl": "",
		"ext":     "",
		"svgUrl":  "",
		"svgExt":  "",
		"source": map[string]any{
			"concatemoji": "",
			"wpemoji":     "",
			"twemoji":     "",
		},
	}
}

// ShouldPrintEmoji reports whether emoji detection scripts should be printed.
// Always false in GoPress (print scripts skip).
func ShouldPrintEmoji() bool {
	return false
}

// PrintEmojiDetectionScript is a no-op stand-in for print_emoji_detection_script.
func PrintEmojiDetectionScript() string {
	return ""
}

// PrintEmojiStyles is a no-op stand-in for print_emoji_styles.
func PrintEmojiStyles() string {
	return ""
}

// EmojiHeadHTML returns head markup for emoji (empty when printing is skipped).
func EmojiHeadHTML() string {
	if !ShouldPrintEmoji() {
		return ""
	}
	return PrintEmojiStyles() + PrintEmojiDetectionScript()
}
