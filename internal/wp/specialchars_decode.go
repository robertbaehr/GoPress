package wp

import (
	"regexp"
	"strings"
)

// Quote-style constants matching PHP ENT_* used by wp_specialchars_decode (P2.78).
const (
	ENTNoQuotes = 0 // ENT_NOQUOTES
	ENTCompat   = 2 // ENT_COMPAT / "double"
	ENTQuotes   = 3 // ENT_QUOTES
)

var (
	reEntityPad39  = regexp.MustCompile(`&#0*39;`)
	reEntityPadX27 = regexp.MustCompile(`(?i)&#x0*27;`)
	reEntityPad34  = regexp.MustCompile(`&#0*34;`)
	reEntityPadX22 = regexp.MustCompile(`(?i)&#x0*22;`)
	reEntityPad60  = regexp.MustCompile(`&#0*60;`)
	reEntityPad62  = regexp.MustCompile(`&#0*62;`)
	reEntityPad38  = regexp.MustCompile(`&#0*38;`)
	reEntityPadX26 = regexp.MustCompile(`(?i)&#x0*26;`)
)

// SpecialcharsDecode mirrors wp_specialchars_decode (P2.78).
// quoteStyle accepts ENTNoQuotes/ENTCompat/ENTQuotes or the strings
// "single" / "double"; unrecognized non-empty values become ENTQuotes.
func SpecialcharsDecode(text string, quoteStyle any) string {
	if text == "" {
		return ""
	}
	if !strings.Contains(text, "&") {
		return text
	}
	style := normalizeQuoteStyle(quoteStyle)

	single := map[string]string{"&#039;": "'", "&#x27;": "'"}
	double := map[string]string{"&quot;": `"`, "&#034;": `"`, "&#x22;": `"`}
	others := map[string]string{
		"&lt;": "<", "&#060;": "<",
		"&gt;": ">", "&#062;": ">",
		"&amp;": "&", "&#038;": "&", "&#x26;": "&",
	}

	var translation map[string]string
	switch style {
	case ENTQuotes:
		translation = mergeMaps(single, double, others)
		text = padSingle(text)
		text = padDouble(text)
		text = padOthers(text)
	case ENTCompat:
		translation = mergeMaps(double, others)
		text = padDouble(text)
		text = padOthers(text)
	case "single":
		translation = mergeMaps(single, others)
		text = padSingle(text)
		text = padOthers(text)
	default: // ENTNoQuotes
		translation = others
		text = padOthers(text)
	}
	return strtr(text, translation)
}

func normalizeQuoteStyle(quoteStyle any) any {
	switch v := quoteStyle.(type) {
	case nil:
		return ENTNoQuotes
	case int:
		switch v {
		case ENTNoQuotes, ENTCompat, ENTQuotes:
			return v
		default:
			return ENTQuotes
		}
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "", "0":
			return ENTNoQuotes
		case "single":
			return "single"
		case "double", "2":
			return ENTCompat
		case "3", "quotes":
			return ENTQuotes
		default:
			return ENTQuotes
		}
	default:
		return ENTQuotes
	}
}

func padSingle(text string) string {
	text = reEntityPad39.ReplaceAllString(text, "&#039;")
	return reEntityPadX27.ReplaceAllString(text, "&#x27;")
}

func padDouble(text string) string {
	text = reEntityPad34.ReplaceAllString(text, "&#034;")
	return reEntityPadX22.ReplaceAllString(text, "&#x22;")
}

func padOthers(text string) string {
	text = reEntityPad60.ReplaceAllString(text, "&#060;")
	text = reEntityPad62.ReplaceAllString(text, "&#062;")
	text = reEntityPad38.ReplaceAllString(text, "&#038;")
	return reEntityPadX26.ReplaceAllString(text, "&#x26;")
}

func mergeMaps(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// strtr performs longest-key-first replacement like PHP strtr($text, $map).
func strtr(text string, m map[string]string) string {
	if len(m) == 0 || text == "" {
		return text
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Longest first so &#038; wins over &amp; when both present as keys.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if len(keys[j]) > len(keys[i]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(text); {
		matched := false
		for _, k := range keys {
			if strings.HasPrefix(text[i:], k) {
				b.WriteString(m[k])
				i += len(k)
				matched = true
				break
			}
		}
		if !matched {
			b.WriteByte(text[i])
			i++
		}
	}
	return b.String()
}
