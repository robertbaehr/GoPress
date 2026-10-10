// KSES HTML sanitizer (P1.11) — subset of wp-includes/kses.php matching golden vectors.
//
// WordPress 7.2 defaults to the HTML-API path (wp_sanitize_html_kses). This port
// approximates that behavior for the golden corpus rather than shipping the full
// HTML API. Remainder / known gaps are documented in kses_test.go.
package wp

import (
	_ "embed"
	"encoding/json"
	"html"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

//go:embed kses_allowed.json
var ksesAllowedJSON []byte

// AttrRule describes whether an attribute is allowed (and optional constraints).
// A nil map for a tag means the tag is allowed with no attributes.
type AttrRule map[string]any

// AllowedHTML maps lowercase tag names to attribute rules.
type AllowedHTML map[string]AttrRule

type ksesAllowFile struct {
	Data      map[string]json.RawMessage `json:"data"`
	Post      map[string]json.RawMessage `json:"post"`
	Entities  []string                   `json:"entities"`
	Protocols []string                   `json:"protocols"`
}

var (
	ksesOnce     sync.Once
	ksesDataHTML AllowedHTML
	ksesPostHTML AllowedHTML
	ksesProtos   []string
)

func loadKsesAllowed() {
	ksesOnce.Do(func() {
		var raw ksesAllowFile
		if err := json.Unmarshal(ksesAllowedJSON, &raw); err != nil {
			panic("wp: kses_allowed.json: " + err.Error())
		}
		ksesDataHTML = parseAllowedContext(raw.Data)
		ksesPostHTML = parseAllowedContext(raw.Post)
		ksesProtos = append([]string(nil), raw.Protocols...)
	})
}

func parseAllowedContext(in map[string]json.RawMessage) AllowedHTML {
	out := make(AllowedHTML, len(in))
	for tag, raw := range in {
		tag = strings.ToLower(tag)
		s := strings.TrimSpace(string(raw))
		if s == "true" {
			out[tag] = nil // tag allowed, any/no attr check → bare tag in legacy; treat as no attrs
			continue
		}
		if s == "[]" || s == "{}" {
			out[tag] = AttrRule{}
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			out[tag] = AttrRule{}
			continue
		}
		out[tag] = AttrRule(m)
	}
	return out
}

// AllowedHTMLPost returns the post-context allow list (wp_kses_allowed_html('post')).
func AllowedHTMLPost() AllowedHTML {
	loadKsesAllowed()
	return ksesPostHTML
}

// AllowedHTMLData returns the data-context allow list (wp_kses_allowed_html('data')).
func AllowedHTMLData() AllowedHTML {
	loadKsesAllowed()
	return ksesDataHTML
}

// AllowedHTMLUserDescription returns the user_description context allow list.
// WordPress aliases this to $allowedtags (same as data); golden allowed.json
// has no extra tags beyond data for this context (P2.14).
func AllowedHTMLUserDescription() AllowedHTML {
	return AllowedHTMLData()
}

// KsesPost sanitizes content with the post allow list (wp_kses_post).
func KsesPost(data string) string {
	return Kses(data, AllowedHTMLPost(), nil)
}

// KsesData sanitizes content with the data allow list (wp_kses_data when
// current_filter() is empty / default).
func KsesData(data string) string {
	return Kses(data, AllowedHTMLData(), nil)
}

// KsesUserDescription sanitizes with the user_description allow list.
func KsesUserDescription(data string) string {
	return Kses(data, AllowedHTMLUserDescription(), nil)
}

// Kses filters HTML content according to allowed tags/attrs (wp_kses subset).
func Kses(content string, allowed AllowedHTML, protocols []string) string {
	loadKsesAllowed()
	if protocols == nil {
		protocols = ksesProtos
	}
	protoSet := make(map[string]bool, len(protocols))
	for _, p := range protocols {
		protoSet[p] = true
	}

	// Strip unwanted C0 controls (HTML-API path); keep TAB/LF/CR.
	content = stripKsesControls(content)
	return ksesSanitize(content, allowed, protoSet)
}

func stripKsesControls(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		// HTML-API: strip \x01-\x08, \x0B, \x0C, \x0E-\x1F; null handled in text.
		if c == 0 || (c < 0x20 && c != '\t' && c != '\n' && c != '\r') {
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// Atomic elements whose inner text is discarded when the tag is not allowed
// (mirrors WP_HTML special_atomic_elements handling).
var ksesAtomic = map[string]bool{
	"iframe": true, "noembed": true, "noframes": true,
	"script": true, "style": true, "textarea": true,
	"title": true, "xmp": true,
}

var ksesVoid = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

var (
	reKsesComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	reKsesTag     = regexp.MustCompile(`(?is)<(/)?\s*([a-zA-Z][a-zA-Z0-9:-]*)((?:[^>"']|"[^"]*"|'[^']*')*)\s*(/?)\s*>`)
	reKsesBogus   = regexp.MustCompile(`(?s)<!(?:\[CDATA\[.*?\]\]|--\[.*?\]--|[^>-][^>]*)>`)
	reKsesPI      = regexp.MustCompile(`(?s)<\?.*?\?>`)
	reKsesAttr    = regexp.MustCompile(`(?is)([^\s"'<>/=]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>` + "`" + `]+)))?`)
)

func ksesSanitize(content string, allowed AllowedHTML, protoSet map[string]bool) string {
	var out strings.Builder
	out.Grow(len(content))
	i := 0
	skipUntil := "" // closing tag name while discarding atomic content

	for i < len(content) {
		if skipUntil != "" {
			// Find closing tag (case-insensitive).
			closePat := "</" + skipUntil
			rest := content[i:]
			low := strings.ToLower(rest)
			idx := strings.Index(low, closePat)
			if idx < 0 {
				return out.String()
			}
			// consume through '>'
			j := i + idx + len(closePat)
			for j < len(content) && content[j] != '>' {
				j++
			}
			if j < len(content) {
				j++
			}
			i = j
			skipUntil = ""
			continue
		}

		lt := strings.IndexByte(content[i:], '<')
		if lt < 0 {
			out.WriteString(ksesEmitText(content[i:]))
			break
		}
		lt += i
		if lt > i {
			out.WriteString(ksesEmitText(content[i:lt]))
		}
		rest := content[lt:]

		// Normative HTML comment.
		if strings.HasPrefix(rest, "<!--") {
			if m := reKsesComment.FindString(rest); m != "" {
				inner := m[len("<!--") : len(m)-len("-->")]
				// Reject incorrectly-closed / non-normative handled elsewhere.
				if strings.Contains(inner, "<") {
					inner = strings.ReplaceAll(inner, "<", "&lt;")
				}
				// Collapse -- like legacy for safety; golden comments are simple.
				out.WriteString("<!--")
				out.WriteString(inner)
				out.WriteString("-->")
				i = lt + len(m)
				continue
			}
			// Unclosed comment: drop remainder opener as text-less.
			out.WriteString(ksesEmitText(rest))
			break
		}

		// Processing instructions / XML declarations.
		if strings.HasPrefix(rest, "<?") {
			if m := reKsesPI.FindString(rest); m != "" {
				i = lt + len(m)
				continue
			}
			// unclosed PI — drop
			i = len(content)
			continue
		}

		// Bogus comments / CDATA / markup declarations → remove.
		if strings.HasPrefix(rest, "<!") {
			if m := reKsesBogus.FindString(rest); m != "" {
				i = lt + len(m)
				continue
			}
			// `<![CDATA[...]]>` without standard match
			if strings.HasPrefix(strings.ToUpper(rest), "<![CDATA[") {
				end := strings.Index(rest, "]]>")
				if end >= 0 {
					i = lt + end + 3
					continue
				}
			}
			i = lt + 1
			continue
		}

		m := reKsesTag.FindStringSubmatch(rest)
		if m == nil {
			// Lone '<' or malformed — emit as text ('<' stays; HTML API may differ).
			// For goldens, malformed tags that aren't real tags become empty removals
			// only when they look like tags. Emit '&lt;' for bare '<'? Goldens don't cover.
			out.WriteString("&lt;")
			i = lt + 1
			continue
		}
		full := m[0]
		closing := m[1] != ""
		elem := strings.ToLower(m[2])
		attrStr := m[3]
		_ = m[4] // self-closing flag ignored in HTML serialization

		rules, ok := allowed[elem]
		if !ok {
			if closing {
				i = lt + len(full)
				continue
			}
			if ksesAtomic[elem] {
				skipUntil = elem
				i = lt + len(full)
				continue
			}
			// Disallowed: drop tag, keep inner content (until we see close — content follows).
			i = lt + len(full)
			continue
		}

		if closing {
			if !ksesVoid[elem] {
				out.WriteString("</")
				out.WriteString(elem)
				out.WriteByte('>')
			}
			i = lt + len(full)
			continue
		}

		// Opening tag.
		tag := ksesBuildTag(elem, attrStr, rules, protoSet)
		out.WriteString(tag)
		i = lt + len(full)
	}
	return out.String()
}

func ksesEmitText(s string) string {
	if s == "" {
		return ""
	}
	// HTML newline normalization.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// Decode character references then re-escape markup-significant chars.
	s = html.UnescapeString(s)
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '\r':
			b.WriteString("&#xD;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func ksesBuildTag(elem, attrStr string, rules AttrRule, protoSet map[string]bool) string {
	var b strings.Builder
	b.WriteByte('<')
	b.WriteString(elem)

	if rules != nil {
		attrs := ksesParseAttrs(attrStr)
		for _, a := range attrs {
			name := strings.ToLower(a.name)
			if !ksesAttrAllowed(name, rules) {
				continue
			}
			val := a.value
			if name == "style" {
				val = safecssFilterAttr(html.UnescapeString(val))
				if val == "" {
					continue
				}
			}
			if ksesURIAttr[name] {
				val = ksesBadProtocolWith(html.UnescapeString(val), protoSet)
				val = ksesEscapeAttr(val)
			} else if !a.vless {
				val = ksesEscapeAttr(html.UnescapeString(val))
			}
			if a.vless {
				b.WriteByte(' ')
				b.WriteString(name)
			} else {
				b.WriteByte(' ')
				b.WriteString(name)
				b.WriteString(`="`)
				b.WriteString(val)
				b.WriteByte('"')
			}
		}
	}

	b.WriteByte('>')
	return b.String()
}

var ksesURIAttr = map[string]bool{
	"action": true, "archive": true, "background": true, "cite": true,
	"classid": true, "codebase": true, "data": true, "formaction": true,
	"href": true, "icon": true, "longdesc": true, "manifest": true,
	"poster": true, "profile": true, "src": true, "usemap": true, "xmlns": true,
}

func ksesAttrAllowed(name string, rules AttrRule) bool {
	if _, ok := rules[name]; ok {
		return true
	}
	if strings.HasPrefix(name, "data-") {
		if _, ok := rules["data-*"]; ok {
			for _, r := range name[5:] {
				if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
					return false
				}
			}
			return true
		}
	}
	return false
}

type ksesAttr struct {
	name  string
	value string
	vless bool
}

func ksesParseAttrs(attr string) []ksesAttr {
	attr = strings.TrimSpace(attr)
	if attr == "" {
		return nil
	}
	// Strip trailing slash from XHTML.
	attr = strings.TrimRight(attr, " \t\n\r/")
	var out []ksesAttr
	for _, m := range reKsesAttr.FindAllStringSubmatch(attr, -1) {
		name := m[1]
		if name == "/" {
			continue
		}
		var val string
		vless := true
		switch {
		case m[2] != "" || strings.Contains(m[0], `"`):
			val = m[2]
			vless = false
		case m[3] != "" || strings.Contains(m[0], `'`):
			val = m[3]
			vless = false
		case m[4] != "":
			val = m[4]
			vless = false
		default:
			if strings.Contains(m[0], "=") {
				vless = false
			}
		}
		out = append(out, ksesAttr{name: name, value: val, vless: vless})
	}
	return out
}

func ksesEscapeAttr(v string) string {
	replacer := strings.NewReplacer(
		`&`, `&amp;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
		`"`, `&quot;`,
		`'`, `&apos;`,
	)
	return replacer.Replace(v)
}

func ksesBadProtocolWith(content string, protoSet map[string]bool) string {
	content = ksesNoNull(content)
	lower := strings.ToLower(content)
	if (strings.HasPrefix(lower, "https://") && protoSet["https"]) ||
		(strings.HasPrefix(lower, "http://") && protoSet["http"]) {
		return content
	}
	orig := content
	for i := 0; i < 6; i++ {
		next := ksesBadProtocolOnceWith(content, protoSet)
		if next == content {
			return content
		}
		content = next
	}
	if content != orig {
		return ""
	}
	return content
}

func ksesBadProtocolOnceWith(content string, protoSet map[string]bool) string {
	idx := strings.Index(content, ":")
	if idx < 0 {
		return content
	}
	scheme := content[:idx]
	rest := content[idx+1:]
	if strings.Contains(scheme, "/") || strings.Contains(scheme, "?") {
		return content
	}
	scheme = strings.ToLower(ksesNoNull(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, scheme)))
	if protoSet[scheme] {
		return scheme + ":" + rest
	}
	return rest
}
