// Emoji settings JSON (P3.49) + print stubs (P3.12).
package wp

import "encoding/json"

const (
	emojiPNGBase = "https://s.w.org/images/core/emoji/17.0.2/72x72/"
	emojiSVGBase = "https://s.w.org/images/core/emoji/17.0.2/svg/"
)

// EmojiSettings returns the wp-emoji-settings object (_print_emoji_detection_script).
// GoPress does not ship the emoji loader module; source URLs stay empty.
func EmojiSettings() map[string]any {
	return map[string]any{
		"baseUrl": emojiPNGBase,
		"ext":     ".png",
		"svgUrl":  emojiSVGBase,
		"svgExt":  ".svg",
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

// PrintEmojiSettingsJSON returns the application/json script tag with id
// wp-emoji-settings (P3.49 / _print_emoji_detection_script settings half).
func PrintEmojiSettingsJSON() string {
	raw, err := json.Marshal(EmojiSettings())
	if err != nil {
		return ""
	}
	return `<script type="application/json" id="wp-emoji-settings">` + string(raw) + `</script>`
}

// PrintEmojiDetectionScript is a no-op stand-in for the emoji loader module.
func PrintEmojiDetectionScript() string {
	return ""
}

// PrintEmojiStyles is a no-op stand-in for print_emoji_styles.
func PrintEmojiStyles() string {
	return ""
}

// EmojiHeadHTML returns head markup: settings JSON only (no loader/styles).
func EmojiHeadHTML() string {
	return PrintEmojiSettingsJSON()
}
