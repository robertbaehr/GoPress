// wp_basename (P2.50 / P2.113) — subset of wp-includes/formatting.php.
package wp

import (
	"net/url"
	"path"
	"strings"
)

// Basename returns the trailing name component of path, locale-independent
// (wp_basename). Encoded slashes/backslashes are treated as separators; an
// optional suffix is stripped when present and shorter than the base
// (PHP 8+ basename semantics — a suffix equal to the whole base is kept).
func Basename(p, suffix string) string {
	// urlencode → replace %2F/%5C with / → basename → urldecode
	enc := url.QueryEscape(p)
	enc = strings.ReplaceAll(enc, "%2F", "/")
	enc = strings.ReplaceAll(enc, "%2f", "/")
	enc = strings.ReplaceAll(enc, "%5C", "/")
	enc = strings.ReplaceAll(enc, "%5c", "/")
	base := path.Base(enc)
	if base == "." || base == "/" {
		// path.Base("") == "."; PHP basename("") == "".
		// path.Base("//") == "/"; PHP basename("//") == "".
		if strings.Trim(enc, "/") == "" {
			base = ""
		}
	}
	// Trailing slash: PHP basename("foo/") == "foo"; path.Base("foo/") == "foo".
	// P2.113: PHP 8+ keeps the name when suffix equals the entire base.
	if suffix != "" && len(base) > len(suffix) && strings.HasSuffix(base, suffix) {
		base = strings.TrimSuffix(base, suffix)
	}
	dec, err := url.QueryUnescape(base)
	if err != nil {
		return base
	}
	return dec
}
