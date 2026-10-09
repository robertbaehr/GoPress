package wp

import (
	"regexp"
	"strings"
)

// Subset of safecss_filter_attr — enough for golden vectors (color; reject expression()).
var safeStyleCSS = map[string]bool{
	"background": true, "background-color": true, "background-image": true,
	"background-position": true, "background-repeat": true, "background-size": true,
	"border": true, "border-radius": true, "border-width": true, "border-color": true,
	"border-style": true, "border-right": true, "border-bottom": true, "border-left": true,
	"border-top": true, "color": true, "display": true, "float": true, "font": true,
	"font-family": true, "font-size": true, "font-style": true, "font-weight": true,
	"height": true, "width": true, "margin": true, "margin-top": true, "margin-right": true,
	"margin-bottom": true, "margin-left": true, "padding": true, "padding-top": true,
	"padding-right": true, "padding-bottom": true, "padding-left": true,
	"text-align": true, "text-decoration": true, "vertical-align": true,
	"line-height": true, "list-style": true, "list-style-type": true,
	"opacity": true, "overflow": true, "position": true, "top": true, "right": true,
	"bottom": true, "left": true, "z-index": true, "clear": true, "cursor": true,
	"direction": true, "visibility": true, "white-space": true, "min-height": true,
	"max-height": true, "min-width": true, "max-width": true,
}

var (
	reCSSCustomVar = regexp.MustCompile(`^--[a-zA-Z0-9-_]+$`)
	reCSSURL       = regexp.MustCompile(`url\([^)]+\)`)
	reCSSGradient  = regexp.MustCompile(`(?:repeating-)?(?:linear|radial|conic)-gradient\((?:[^()]|\([^()]*\))*\)`)
	reCSSFn        = regexp.MustCompile(`(?i)\b(?:var|calc|min|max|minmax|clamp|repeat|matrix|matrix3d|perspective|rotate|rotate3d|rotateX|rotateY|rotateZ|scale|scale3d|scaleX|scaleY|scaleZ|skew|skewX|skewY|translate|translate3d|translateX|translateY|translateZ|circle|ellipse|inset|path|polygon|rect|shape|xywh)\((?:[^()]|\([^()]*\))*\)`)
)

// safecssFilterAttr mirrors safecss_filter_attr for the properties we allowlist.
func safecssFilterAttr(css string) string {
	css = ksesNoNull(css)
	css = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, css)

	parts := strings.Split(strings.TrimSpace(css), ";")
	var kept []string
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		test := item
		found := false
		if !strings.Contains(item, ":") {
			found = true
		} else {
			colon := strings.IndexByte(item, ':')
			sel := strings.TrimSpace(item[:colon])
			if safeStyleCSS[sel] || reCSSCustomVar.MatchString(sel) {
				found = true
			}
		}
		if !found {
			continue
		}
		// Strip url()/gradient()/allowed fns from the test string, then reject unsafe chars.
		test = reCSSURL.ReplaceAllString(test, "")
		test = reCSSGradient.ReplaceAllString(test, "")
		test = reCSSFn.ReplaceAllString(test, "")
		// WP: disallow \ ( & } = or comments after removing allowed functions.
		if strings.ContainsAny(test, "\\(&=}") || strings.Contains(test, "/*") {
			continue
		}
		kept = append(kept, item)
	}
	return strings.Join(kept, ";")
}
