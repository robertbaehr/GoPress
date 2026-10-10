package wp

import (
	"regexp"
	"sync"
)

// DefaultSpacesRegexp is the default wp_spaces_regexp() pattern (P2.48).
// Matches CR/LF/TAB/space, UTF-8 non-breaking space (U+00A0 / PHP \xC2\xA0),
// or the &nbsp; entity. Go RE2 uses \x{00A0} for the NBSP code point.
const DefaultSpacesRegexp = `[\r\n\t ]|\x{00A0}|&nbsp;`

var (
	spacesRegexpMu    sync.Mutex
	spacesRegexpCache string
	spacesRegexpRE    *regexp.Regexp
)

// SpacesRegexp returns the regexp pattern for common whitespace (wp_spaces_regexp).
// When c is non-nil, the "wp_spaces_regexp" filter may replace the default.
func SpacesRegexp(c *Context) string {
	pattern := DefaultSpacesRegexp
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_spaces_regexp", pattern); v != nil {
			if s, ok := v.(string); ok && s != "" {
				pattern = s
			}
		}
	}
	return pattern
}

// SpacesRegexpCompiled returns a compiled form of SpacesRegexp(c).
// Results for the default (and last filtered) pattern are cached.
func SpacesRegexpCompiled(c *Context) *regexp.Regexp {
	pattern := SpacesRegexp(c)
	spacesRegexpMu.Lock()
	defer spacesRegexpMu.Unlock()
	if spacesRegexpRE != nil && spacesRegexpCache == pattern {
		return spacesRegexpRE
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		re = regexp.MustCompile(DefaultSpacesRegexp)
		pattern = DefaultSpacesRegexp
	}
	spacesRegexpCache = pattern
	spacesRegexpRE = re
	return re
}

// MatchSpaces reports whether s begins with a wp_spaces_regexp match.
func MatchSpaces(c *Context, s string) bool {
	if s == "" {
		return false
	}
	loc := SpacesRegexpCompiled(c).FindStringIndex(s)
	return loc != nil && loc[0] == 0
}
