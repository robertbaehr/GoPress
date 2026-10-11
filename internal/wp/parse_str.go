package wp

import (
	"net/url"
	"strings"
)

// ParseStr parses a query string into a nested map (wp_parse_str / PHP parse_str, P2.35 / P2.105).
//
// Supports PHP-style nested keys: a[b]=1&a[c]=2 → {"a":{"b":"1","c":"2"}}.
// Repeated scalar keys become []any. Leading "?" is stripped.
// P2.105: NUL bytes stripped; HTML `&amp;` accepted as separators; PHP key
// normalization converts '.' and spaces in bare key segments to '_'.
func ParseStr(input string) map[string]any {
	return parseStrRaw(input)
}

// ParseStrFiltered runs ParseStr then the wp_parse_str filter when hooks exist.
func (c *Context) ParseStrFiltered(input string) map[string]any {
	result := parseStrRaw(input)
	if c == nil || c.Hooks == nil {
		return result
	}
	if v := c.Hooks.ApplyFilters("wp_parse_str", result); v != nil {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return result
}

func parseStrRaw(input string) map[string]any {
	input = strings.TrimSpace(input)
	if input == "" {
		return map[string]any{}
	}
	// P2.105: PHP parse_str ignores embedded NUL bytes.
	input = strings.ReplaceAll(input, "\x00", "")
	input = strings.TrimPrefix(input, "?")
	// Accept HTML-entity ampersands commonly passed from markup.
	input = strings.ReplaceAll(input, "&amp;", "&")
	input = strings.ReplaceAll(input, "&Amp;", "&")
	vals, err := url.ParseQuery(input)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	for key, vs := range vals {
		key = normalizeParseStrKey(key)
		for _, v := range vs {
			assignParseStr(out, key, v)
		}
	}
	return out
}

// normalizeParseStrKey mirrors PHP parse_str key munging (P2.105):
// spaces and dots in each segment become underscores. Bracket structure
// is preserved (a[b.c] → a[b_c]).
func normalizeParseStrKey(key string) string {
	if key == "" {
		return key
	}
	if !strings.ContainsAny(key, ". ") {
		return key
	}
	if !strings.Contains(key, "[") {
		return strings.ReplaceAll(strings.ReplaceAll(key, ".", "_"), " ", "_")
	}
	base, rest, ok := strings.Cut(key, "[")
	if !ok {
		return strings.ReplaceAll(strings.ReplaceAll(key, ".", "_"), " ", "_")
	}
	base = strings.ReplaceAll(strings.ReplaceAll(base, ".", "_"), " ", "_")
	var b strings.Builder
	b.WriteString(base)
	b.WriteByte('[')
	for i := 0; i < len(rest); {
		if rest[i] == ']' {
			b.WriteByte(']')
			i++
			continue
		}
		if rest[i] == '[' {
			b.WriteByte('[')
			i++
			continue
		}
		// Segment until next ']' or '['.
		j := i
		for j < len(rest) && rest[j] != ']' && rest[j] != '[' {
			j++
		}
		seg := rest[i:j]
		seg = strings.ReplaceAll(strings.ReplaceAll(seg, ".", "_"), " ", "_")
		b.WriteString(seg)
		i = j
	}
	return b.String()
}

// assignParseStr writes value into dest under a PHP-style key path.
func assignParseStr(dest map[string]any, key, value string) {
	if key == "" {
		return
	}
	// Flat key without brackets.
	if !strings.Contains(key, "[") {
		setParseStrScalar(dest, key, value)
		return
	}
	base, rest, ok := strings.Cut(key, "[")
	if !ok || base == "" {
		setParseStrScalar(dest, key, value)
		return
	}
	parts := splitParseStrBrackets(rest)
	cur := dest
	// Ensure base is a map.
	node, ok := cur[base]
	if !ok {
		m := map[string]any{}
		cur[base] = m
		cur = m
	} else if m, ok := node.(map[string]any); ok {
		cur = m
	} else {
		// Scalar collision: replace with map.
		m := map[string]any{}
		cur[base] = m
		cur = m
	}
	for i, part := range parts {
		last := i == len(parts)-1
		if part == "" {
			// a[]=x → append to slice under current... PHP uses numeric append.
			// Represent empty-bracket append as a slice under a synthetic path:
			// for a[]=1&a[]=2 we store under "" key as growing slice is awkward;
			// use consecutive int keys when parent is map and key is "".
			if last {
				appendParseStrEmpty(cur, value)
				return
			}
			// Nested empty mid-path is rare; treat as "".
			part = ""
		}
		if last {
			setParseStrScalar(cur, part, value)
			return
		}
		next, ok := cur[part]
		if !ok {
			m := map[string]any{}
			cur[part] = m
			cur = m
			continue
		}
		if m, ok := next.(map[string]any); ok {
			cur = m
			continue
		}
		m := map[string]any{}
		cur[part] = m
		cur = m
	}
}

func splitParseStrBrackets(rest string) []string {
	// rest is everything after the first '[', e.g. "b][c]=..." already cut at '=' by ParseQuery.
	// Keys look like "b][c]" or "b]" or "]".
	var parts []string
	s := rest
	for s != "" {
		end := strings.IndexByte(s, ']')
		if end < 0 {
			parts = append(parts, s)
			break
		}
		parts = append(parts, s[:end])
		s = s[end+1:]
		if strings.HasPrefix(s, "[") {
			s = s[1:]
			continue
		}
		if s != "" {
			// Trailing junk — append as final segment.
			parts = append(parts, s)
			break
		}
	}
	return parts
}

func setParseStrScalar(dest map[string]any, key, value string) {
	if existing, ok := dest[key]; ok {
		switch t := existing.(type) {
		case []any:
			dest[key] = append(t, value)
		default:
			dest[key] = []any{t, value}
		}
		return
	}
	dest[key] = value
}

func appendParseStrEmpty(dest map[string]any, value string) {
	// Prefer numeric keys 0,1,2… like PHP when using a[].
	i := 0
	for {
		k := itoaParseStr(i)
		if _, ok := dest[k]; !ok {
			dest[k] = value
			return
		}
		i++
		if i > 10000 {
			return
		}
	}
}

func itoaParseStr(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
