// Formatting escapes and sanitizers (P1.10) — subset of wp-includes/formatting.php.
package wp

import (
	"fmt"
	"html"
	"math"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Byte size constants (wp-includes/default-constants.php).
const (
	KBInBytes = 1024
	MBInBytes = 1024 * KBInBytes
	GBInBytes = 1024 * MBInBytes
	TBInBytes = 1024 * GBInBytes
	PBInBytes = 1024 * TBInBytes
	EBInBytes = 1024 * PBInBytes
)

var (
	// Separate patterns (Go RE2 has no backrefs); mirrors PHP's script|style + \\1 pair.
	reScriptBlock   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyleBlock    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reTags          = regexp.MustCompile(`(?s)<[^>]*>`)
	reWhitespace    = regexp.MustCompile(`[\r\n\t ]+`)
	reSpaces        = regexp.MustCompile(` +`)
	rePercentOctet  = regexp.MustCompile(`(?i)%[a-f0-9]{2}`)
	reSanitizeKey   = regexp.MustCompile(`[^a-z0-9_\-]`)
	reTitleEntity   = regexp.MustCompile(`&.+?;`)
	reTitleAllowed  = regexp.MustCompile(`[^%a-z0-9 _-]`)
	reTitleSpaces   = regexp.MustCompile(`\s+`)
	reTitleDashes   = regexp.MustCompile(`-+`)
	reEscJSApos     = regexp.MustCompile(`(?i)&#(x)?0*(?:27|39);?`)
	reOctetPreserve = regexp.MustCompile(`%([a-fA-F0-9][a-fA-F0-9])`)
	reOctetRestore  = regexp.MustCompile(`---([a-fA-F0-9][a-fA-F0-9])---`)
	reCRLFOctet     = regexp.MustCompile(`(?i)%0d|%0a`)
	reNormDecEntity = regexp.MustCompile(`&amp;#(0*[1-9][0-9]{0,6});`)
	reNormHexEntity = regexp.MustCompile(`&amp;#[Xx](0*[1-9A-Fa-f][0-9A-Fa-f]{0,5});`)
	reNormNamedEnt  = regexp.MustCompile(`&amp;([A-Za-z]{2,8}[0-9]{0,2});`)
	reFileDots      = regexp.MustCompile(`\.{2,}`)
	reFileDashes    = regexp.MustCompile(`[\r\n\t -]+`)
	reFileSpaceSep  = regexp.MustCompile(`\p{Zs}`)
	reFileAlphaExt  = regexp.MustCompile(`(?i)^[a-z]{2,5}\d?$`)
)

// specialFileNameChars mirrors sanitize_file_name's $special_chars list.
var specialFileNameChars = []string{
	"?", "[", "]", "/", "\\", "=", "<", ">", ":", ";", ",", "'", "\"",
	"&", "$", "#", "*", "(", ")", "|", "~", "`", "!", "{", "}", "%", "+",
	"\u2019", "\u00ab", "\u00bb", "\u201d", "\u201c", "\x00",
}

// named entities accepted by a minimal wp_kses_normalize_entities subset.
var namedEntities = map[string]bool{
	"amp": true, "lt": true, "gt": true, "quot": true, "apos": true,
	"nbsp": true, "ndash": true, "mdash": true, "hellip": true,
}

// Allowed URL protocols (wp_allowed_protocols default list).
var allowedProtocols = map[string]bool{
	"http": true, "https": true, "ftp": true, "ftps": true, "mailto": true,
	"news": true, "irc": true, "irc6": true, "ircs": true, "gopher": true,
	"nntp": true, "feed": true, "telnet": true, "mms": true, "rtsp": true,
	"sms": true, "svn": true, "tel": true, "fax": true, "xmpp": true,
	"webcal": true, "urn": true,
}

// EscHTML escapes text for HTML body output (esc_html).
func EscHTML(text string) string {
	return wpSpecialchars(checkUTF8(text), true)
}

// EscAttr escapes text for HTML attribute output (esc_attr).
func EscAttr(text string) string {
	return wpSpecialchars(checkUTF8(text), true)
}

// EscJS escapes text for inline JavaScript (esc_js).
func EscJS(text string) string {
	safe := wpSpecialchars(checkUTF8(text), false) // ENT_COMPAT
	safe = reEscJSApos.ReplaceAllString(safe, "'")
	safe = phpStripslashes(safe)
	safe = strings.ReplaceAll(safe, "\r", "")
	safe = phpAddslashes(safe)
	safe = strings.ReplaceAll(safe, "\n", `\n`)
	return safe
}

// EscURL cleans and escapes a URL for display (esc_url subset).
func EscURL(url string) string {
	return escURLContext(url, "display")
}

// EscURLRaw cleans a URL for database or redirect use (esc_url_raw / sanitize_url).
// Unlike EscURL, it skips display-only entity encoding (P2.26).
func EscURLRaw(url string) string {
	return escURLContext(url, "db")
}

// SanitizeURL is an alias for EscURLRaw (sanitize_url).
func SanitizeURL(url string) string {
	return EscURLRaw(url)
}

func escURLContext(url, context string) string {
	if url == "" {
		return ""
	}
	url = strings.TrimLeft(url, " \t\n\r\f\v")
	url = strings.ReplaceAll(url, " ", "%20")
	url = filterURLChars(url)
	if url == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(url), "mailto:") {
		url = reCRLFOctet.ReplaceAllString(url, "")
	}
	url = strings.ReplaceAll(url, ";//", "://")

	if !strings.Contains(url, ":") && url != "" {
		first := url[0]
		if first != '/' && first != '#' && first != '?' && !looksLikePHPFile(url) {
			url = "http://" + url
		}
	}

	// Replace ampersands and single quotes only when displaying.
	if context == "display" {
		url = normalizeEntities(url)
		url = strings.ReplaceAll(url, "&amp;", "&#038;")
		url = strings.ReplaceAll(url, "'", "&#039;")
	}

	if url == "" {
		return ""
	}
	if url[0] == '/' {
		return url
	}
	cleaned := ksesBadProtocol(url)
	if !strings.EqualFold(cleaned, url) {
		return ""
	}
	return cleaned
}

// SanitizeKey lowercases and strips non [a-z0-9_-] (sanitize_key).
func SanitizeKey(key string) string {
	key = strings.ToLower(key)
	return reSanitizeKey.ReplaceAllString(key, "")
}

// SanitizeTextField sanitizes a string from user input (sanitize_text_field).
func SanitizeTextField(str string) string {
	return sanitizeTextFields(str, false)
}

// SanitizeTitle produces a slug (sanitize_title + sanitize_title_with_dashes, save context).
func SanitizeTitle(title string) string {
	title = removeAccents(title)
	return sanitizeTitleWithDashes(title)
}

// SanitizeFileName sanitizes a filename (sanitize_file_name, P2.29).
func SanitizeFileName(filename string) string {
	filename = removeAccents(filename)

	if !utf8.ValidString(filename) {
		ext := ""
		name := filename
		if i := strings.LastIndex(filename, "."); i >= 0 {
			ext = filename[i+1:]
			name = filename[:i]
		}
		filename = sanitizeTitleWithDashes(name)
		if ext != "" {
			filename = filename + "." + ext
		}
	}

	filename = reFileSpaceSep.ReplaceAllString(filename, " ")

	for _, ch := range specialFileNameChars {
		filename = strings.ReplaceAll(filename, ch, "")
	}
	filename = strings.ReplaceAll(filename, "%20", "-")
	filename = strings.ReplaceAll(filename, "+", "-")
	filename = reFileDots.ReplaceAllString(filename, ".")
	filename = reFileDashes.ReplaceAllString(filename, "-")
	filename = strings.Trim(filename, ".-_")

	if filename != "" && !strings.Contains(filename, ".") {
		if checkFiletypeExt("test."+filename) == filename {
			filename = "unnamed-file." + filename
		}
	}

	parts := strings.Split(filename, ".")
	if len(parts) <= 2 {
		return filename
	}

	base := parts[0]
	ext := parts[len(parts)-1]
	middles := parts[1 : len(parts)-1]
	for _, part := range middles {
		base += "." + part
		if reFileAlphaExt.MatchString(part) && !isAllowedFileExt(part) {
			base += "_"
		}
	}
	return base + "." + ext
}

// mimeTypeExtGroups is a subset of wp_get_mime_types / get_allowed_mime_types keys.
var mimeTypeExtGroups = []string{
	"jpg|jpeg|jpe", "gif", "png", "bmp", "tiff|tif", "webp", "avif", "ico",
	"heic", "heif", "heics", "heifs",
	"asf|asx", "wmv", "wmx", "wm", "avi", "divx", "flv", "mov|qt", "mpeg|mpg|mpe",
	"mp4|m4v", "ogv", "webm", "mkv",
	"3gp|3gpp", "3g2|3gp2",
	"txt|asc|c|cc|h|srt", "csv", "tsv", "ics", "rtx", "css", "htm|html", "vtt",
	"mp3|m4a|m4b", "aac", "ra|ram", "wav|x-wav", "ogg|oga", "mid|midi", "wma", "wax", "mka",
	"rtf", "js", "pdf", "class", "tar", "zip", "gz|gzip", "rar", "7z",
	"doc", "pot|pps|ppt", "wri", "xla|xls|xlt|xlw", "mdb", "mpp", "docx", "docm", "dotx", "dotm",
	"xlsx", "xlsm", "xlsb", "xltx", "xltm", "xlam", "pptx", "pptm", "ppsx", "ppsm", "potx", "potm", "ppam",
	"sldx", "sldm", "onetoc|onetoc2|onetmp|onepkg", "oxps", "xps",
	"odt", "odp", "ods", "odg", "odc", "odb", "odf",
	"wp|wpd", "key", "numbers", "pages",
}

func checkFiletypeExt(filename string) string {
	lower := strings.ToLower(filename)
	for _, group := range mimeTypeExtGroups {
		for _, ext := range strings.Split(group, "|") {
			if strings.HasSuffix(lower, "."+ext) {
				return ext
			}
		}
	}
	return ""
}

func isAllowedFileExt(part string) bool {
	part = strings.ToLower(part)
	for _, group := range mimeTypeExtGroups {
		for _, ext := range strings.Split(group, "|") {
			if part == ext {
				return true
			}
		}
	}
	// get_allowed_mime_types unsets swf/exe and often htm|html/js — treat as disallowed.
	return false
}

func sanitizeTextFields(str string, keepNewlines bool) string {
	filtered := checkUTF8(str)
	if strings.Contains(filtered, "<") {
		filtered = preKsesLessThan(filtered)
		filtered = WPStripAllTags(filtered, false)
		filtered = strings.ReplaceAll(filtered, "<\n", "&lt;\n")
	}
	if !keepNewlines {
		filtered = reWhitespace.ReplaceAllString(filtered, " ")
	}
	filtered = strings.TrimSpace(filtered)
	for {
		loc := rePercentOctet.FindStringIndex(filtered)
		if loc == nil {
			break
		}
		filtered = filtered[:loc[0]] + filtered[loc[1]:]
		filtered = strings.TrimSpace(reSpaces.ReplaceAllString(filtered, " "))
	}
	return filtered
}

func sanitizeTitleWithDashes(title string) string {
	// PHP strip_tags (keeps script/style inner text), not wp_strip_all_tags.
	title = strings.ReplaceAll(title, "\x00", "")
	title = reTags.ReplaceAllString(title, "")
	title = strings.TrimSpace(title)
	title = reOctetPreserve.ReplaceAllString(title, "---$1---")
	title = strings.ReplaceAll(title, "%", "")
	title = reOctetRestore.ReplaceAllString(title, "%$1")

	if utf8.ValidString(title) {
		title = strings.ToLower(title)
		title = utf8URIEncode(title, 200)
	}
	title = strings.ToLower(title)

	// save context
	for _, pair := range [][2]string{
		{"%c2%a0", "-"}, {"%e2%80%91", "-"}, {"%e2%80%93", "-"}, {"%e2%80%94", "-"},
		{"&nbsp;", "-"}, {"&#8209;", "-"}, {"&#160;", "-"},
		{"&ndash;", "-"}, {"&#8211;", "-"}, {"&mdash;", "-"}, {"&#8212;", "-"},
	} {
		title = strings.ReplaceAll(title, pair[0], pair[1])
	}
	title = strings.ReplaceAll(title, "/", "-")

	title = reTitleEntity.ReplaceAllString(title, "")
	title = strings.ReplaceAll(title, ".", "-")
	title = reTitleAllowed.ReplaceAllString(title, "")
	title = reTitleSpaces.ReplaceAllString(title, "-")
	title = reTitleDashes.ReplaceAllString(title, "-")
	title = strings.Trim(title, "-")
	return title
}

func removeAccents(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range norm.NFD.String(text) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		switch r {
		case 'Æ':
			b.WriteString("AE")
		case 'æ':
			b.WriteString("ae")
		case 'Ø':
			b.WriteByte('O')
		case 'ø':
			b.WriteByte('o')
		case 'Ð':
			b.WriteByte('D')
		case 'ð':
			b.WriteByte('d')
		case 'Þ':
			b.WriteString("TH")
		case 'þ':
			b.WriteString("th")
		case 'ß':
			b.WriteByte('s')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func utf8URIEncode(s string, length int) string {
	var b strings.Builder
	n := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// encode raw byte
			if length > 0 && n+3 > length {
				break
			}
			b.WriteString("%" + strings.ToLower(hexByte(s[i])))
			n += 3
			i++
			continue
		}
		if r < 128 {
			if length > 0 && n+1 > length {
				break
			}
			b.WriteByte(byte(r))
			n++
		} else {
			enc := s[i : i+size]
			need := 3 * len(enc)
			if length > 0 && n+need > length {
				break
			}
			for j := 0; j < len(enc); j++ {
				b.WriteString("%" + strings.ToLower(hexByte(enc[j])))
			}
			n += need
		}
		i += size
	}
	return b.String()
}

func hexByte(c byte) string {
	const hexdigits = "0123456789abcdef"
	return string([]byte{hexdigits[c>>4], hexdigits[c&0x0f]})
}

func checkUTF8(text string) string {
	if text == "" {
		return ""
	}
	if !utf8.ValidString(text) {
		return ""
	}
	return text
}

// wpSpecialchars mirrors _wp_specialchars with double_encode=false.
// quotes=true → ENT_QUOTES; false → ENT_COMPAT (double quotes only).
func wpSpecialchars(text string, quotes bool) string {
	if text == "" {
		return ""
	}
	if !strings.ContainsAny(text, `&<>"'`) {
		return text
	}
	text = normalizeEntities(text)
	return escapeHTML(text, quotes)
}

func normalizeEntities(content string) string {
	content = strings.ReplaceAll(content, "&", "&amp;")
	// Decimal numeric: &amp;#38; → &#038;
	content = reNormDecEntity.ReplaceAllStringFunc(content, func(m string) string {
		inner := m[len("&amp;#") : len(m)-1]
		inner = strings.TrimLeft(inner, "0")
		if inner == "" {
			return m
		}
		for len(inner) < 3 {
			inner = "0" + inner
		}
		return "&#" + inner + ";"
	})
	// Hex numeric: &amp;#x26; → &#x26;
	content = reNormHexEntity.ReplaceAllStringFunc(content, func(m string) string {
		idx := strings.IndexAny(m, "xX")
		hex := m[idx+1 : len(m)-1]
		hex = strings.TrimLeft(hex, "0")
		if hex == "" {
			return m
		}
		return "&#x" + hex + ";"
	})
	// Named
	content = reNormNamedEnt.ReplaceAllStringFunc(content, func(m string) string {
		name := m[len("&amp;") : len(m)-1]
		if namedEntities[name] {
			return "&" + name + ";"
		}
		return "&amp;" + name + ";"
	})
	return content
}

func escapeHTML(text string, quotes bool) string {
	var b strings.Builder
	b.Grow(len(text) + 16)
	for i := 0; i < len(text); {
		c := text[i]
		switch c {
		case '&':
			// leave existing entities intact (double_encode=false)
			if ent := entityAt(text, i); ent != "" {
				b.WriteString(ent)
				i += len(ent)
				continue
			}
			b.WriteString("&amp;")
			i++
		case '<':
			b.WriteString("&lt;")
			i++
		case '>':
			b.WriteString("&gt;")
			i++
		case '"':
			b.WriteString("&quot;")
			i++
		case '\'':
			if quotes {
				b.WriteString("&#039;")
			} else {
				b.WriteByte('\'')
			}
			i++
		default:
			// copy next rune
			_, size := utf8.DecodeRuneInString(text[i:])
			b.WriteString(text[i : i+size])
			i += size
		}
	}
	return b.String()
}

func entityAt(s string, i int) string {
	if i >= len(s) || s[i] != '&' {
		return ""
	}
	// &#123; or &#xHH; or &name;
	j := i + 1
	if j >= len(s) {
		return ""
	}
	if s[j] == '#' {
		j++
		if j < len(s) && (s[j] == 'x' || s[j] == 'X') {
			j++
			start := j
			for j < len(s) && isHex(s[j]) {
				j++
			}
			if j < len(s) && s[j] == ';' && j > start {
				return s[i : j+1]
			}
			return ""
		}
		start := j
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j < len(s) && s[j] == ';' && j > start {
			return s[i : j+1]
		}
		return ""
	}
	start := j
	for j < len(s) && ((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z') || (s[j] >= '0' && s[j] <= '9')) {
		j++
	}
	if j < len(s) && s[j] == ';' && j > start {
		name := s[start:j]
		if namedEntities[name] || html.UnescapeString("&"+name+";") != "&"+name+";" {
			return s[i : j+1]
		}
		// accept any alphabetic entity-looking sequence for double_encode=false
		if len(name) >= 2 {
			return s[i : j+1]
		}
	}
	return ""
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func filterURLChars(url string) string {
	var b strings.Builder
	b.Grow(len(url))
	for i := 0; i < len(url); {
		c := url[i]
		if c < 0x80 {
			if isURLChar(c) {
				b.WriteByte(c)
			}
			i++
			continue
		}
		// allow UTF-8 high bytes (\\x80-\\xff in WP regex)
		_, size := utf8.DecodeRuneInString(url[i:])
		b.WriteString(url[i : i+size])
		i += size
	}
	return b.String()
}

func isURLChar(c byte) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	switch c {
	case '-', '~', '+', '_', '.', '?', '#', '=', '!', '&', ';', ',', '/', ':', '%', '@', '$', '|', '*', '\'', '(', ')', '[', ']':
		return true
	}
	return false
}

func looksLikePHPFile(url string) bool {
	// /^[a-z0-9-]+?\.php/i
	for i := 0; i < len(url); i++ {
		c := url[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		if c == '.' && strings.HasPrefix(strings.ToLower(url[i:]), ".php") {
			return true
		}
		return false
	}
	return false
}

func ksesBadProtocol(content string) string {
	content = ksesNoNull(content)
	lower := strings.ToLower(content)
	if (strings.HasPrefix(lower, "https://") && allowedProtocols["https"]) ||
		(strings.HasPrefix(lower, "http://") && allowedProtocols["http"]) {
		return content
	}
	for i := 0; i < 6; i++ {
		next := ksesBadProtocolOnce(content)
		if next == content {
			return content
		}
		content = next
	}
	return ""
}

func ksesBadProtocolOnce(content string) string {
	// split on first :
	idx := strings.Index(content, ":")
	if idx < 0 {
		return content
	}
	scheme := content[:idx]
	rest := content[idx+1:]
	// if scheme contains /? skip (WP: preg_match('%/\?%', $content2[0]))
	if strings.Contains(scheme, "/") || strings.Contains(scheme, "?") {
		return content
	}
	scheme = strings.ToLower(ksesNoNull(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, scheme)))
	if allowedProtocols[scheme] {
		return scheme + ":" + rest
	}
	return rest // strip bad protocol
}

func ksesNoNull(content string) string {
	var b strings.Builder
	b.Grow(len(content))
	for i := 0; i < len(content); i++ {
		c := content[i]
		if c < 0x20 && c != '\t' && c != '\n' && c != '\r' {
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func preKsesLessThan(content string) string {
	// %<[^>]*?((?=<)|>|$)% — escape incomplete tags
	var b strings.Builder
	i := 0
	for i < len(content) {
		lt := strings.IndexByte(content[i:], '<')
		if lt < 0 {
			b.WriteString(content[i:])
			break
		}
		lt += i
		b.WriteString(content[i:lt])
		rest := content[lt:]
		gt := strings.IndexByte(rest, '>')
		nextLt := strings.IndexByte(rest[1:], '<')
		if gt < 0 {
			// no close — escape whole remainder match
			b.WriteString(EscHTML(rest))
			break
		}
		if nextLt >= 0 && nextLt+1 < gt {
			// next < before > — incomplete
			frag := rest[:nextLt+1]
			b.WriteString(EscHTML(frag))
			i = lt + nextLt + 1
			continue
		}
		b.WriteString(rest[:gt+1])
		i = lt + gt + 1
	}
	return b.String()
}

func phpAddslashes(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'', '"', '\\':
			b.WriteByte('\\')
			b.WriteByte(s[i])
		case 0:
			b.WriteString(`\0`)
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func phpStripslashes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			b.WriteByte(s[i])
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// SizeFormat converts a byte count to a human-readable size string (size_format, P2.28).
// decimals mirrors the PHP $decimals argument (number_format_i18n subset).
// Returns "" when bytes is negative (PHP returns false).
func SizeFormat(bytes float64, decimals int) string {
	if math.IsNaN(bytes) || math.IsInf(bytes, 0) || bytes < 0 {
		return ""
	}
	if decimals < 0 {
		decimals = 0
	}
	quant := []struct {
		unit string
		mag  float64
	}{
		{"TB", TBInBytes},
		{"GB", GBInBytes},
		{"MB", MBInBytes},
		{"KB", KBInBytes},
		{"B", 1},
	}
	if bytes == 0 {
		return numberFormatI18n(0, decimals) + " B"
	}
	for _, q := range quant {
		if bytes >= q.mag {
			return numberFormatI18n(bytes/q.mag, decimals) + " " + q.unit
		}
	}
	return ""
}

// numberFormatI18n is a minimal number_format_i18n (period decimal, no thousands sep).
func numberFormatI18n(n float64, decimals int) string {
	return fmt.Sprintf("%.*f", decimals, n)
}
