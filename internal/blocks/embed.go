package blocks

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

var reEmbedFigureOpen = regexp.MustCompile(`(?is)^(\s*<figure\b)([^>]*)(>)`)

// renderEmbed renders core/embed (P5.49).
// Honors providerNameSlug / type / align attrs (WP save.mjs classnames), keeps
// saved InnerHTML when present, and synthesizes a figure fallback from url.
func (e *Env) renderEmbed(block Block) string {
	e = e.ensure()
	urlStr := strings.TrimSpace(attrString(block.Attrs, "url"))
	provider := strings.TrimSpace(attrString(block.Attrs, "providerNameSlug"))
	typ := strings.TrimSpace(attrString(block.Attrs, "type"))
	if provider == "" && urlStr != "" {
		provider = inferEmbedProvider(urlStr)
	}
	if typ == "" && provider != "" {
		typ = "rich"
		if provider == "youtube" || provider == "vimeo" || provider == "dailymotion" {
			typ = "video"
		}
	}

	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeEmbedFigure(urlStr, attrString(block.Attrs, "caption"), typ, provider, block.Attrs)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	return ensureEmbedFigureClasses(out, typ, provider, block.Attrs)
}

func synthesizeEmbedFigure(rawURL, caption, typ, provider string, attrs map[string]any) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	esc := html.EscapeString(rawURL)
	var b strings.Builder
	b.WriteString(`<figure class="`)
	b.WriteString(html.EscapeString(embedFigureClassList(typ, provider, attrs)))
	b.WriteString(`"><div class="wp-block-embed__wrapper"><a href="`)
	b.WriteString(esc)
	b.WriteString(`">`)
	b.WriteString(esc)
	b.WriteString(`</a></div>`)
	caption = strings.TrimSpace(caption)
	if caption != "" {
		b.WriteString(`<figcaption class="wp-element-caption">`)
		b.WriteString(html.EscapeString(caption))
		b.WriteString(`</figcaption>`)
	}
	b.WriteString(`</figure>`)
	return b.String()
}

func ensureEmbedFigureClasses(htmlFragment, typ, provider string, attrs map[string]any) string {
	want := embedFigureClassList(typ, provider, attrs)
	if want == "" || want == "wp-block-embed" {
		return htmlFragment
	}
	m := reEmbedFigureOpen.FindStringSubmatchIndex(htmlFragment)
	if m == nil {
		return htmlFragment
	}
	open := htmlFragment[m[2]:m[3]]
	attrPart := htmlFragment[m[4]:m[5]]
	close := htmlFragment[m[6]:m[7]]
	rest := htmlFragment[m[1]:]

	classVal, hasClass, other := splitClassAttr(attrPart)
	merged := mergeClassList(classVal, want)
	var b strings.Builder
	b.WriteString(open)
	if hasClass || merged != "" {
		b.WriteString(` class="`)
		b.WriteString(html.EscapeString(merged))
		b.WriteString(`"`)
	}
	b.WriteString(other)
	b.WriteString(close)
	b.WriteString(rest)
	return b.String()
}

func embedFigureClassList(typ, provider string, attrs map[string]any) string {
	parts := []string{"wp-block-embed"}
	if typ != "" {
		parts = append(parts, "is-type-"+sanitizeClassToken(typ))
	}
	if provider != "" {
		p := sanitizeClassToken(provider)
		parts = append(parts, "is-provider-"+p, "wp-block-embed-"+p)
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		parts = append(parts, "align"+sanitizeClassToken(align))
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		parts = append(parts, strings.Fields(cn)...)
	}
	if responsiveAttrTrue(attrs) {
		parts = append(parts, "wp-embed-responsive")
	}
	return strings.Join(uniqueClassParts(parts), " ")
}

func responsiveAttrTrue(attrs map[string]any) bool {
	if attrs == nil {
		return false
	}
	v, ok := attrs["responsive"]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}

func inferEmbedProvider(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	switch {
	case strings.Contains(host, "youtube.com"), host == "youtu.be":
		return "youtube"
	case strings.Contains(host, "vimeo.com"):
		return "vimeo"
	case strings.Contains(host, "twitter.com"), host == "x.com":
		return "twitter"
	case strings.Contains(host, "instagram.com"):
		return "instagram"
	case strings.Contains(host, "spotify.com"):
		return "spotify"
	case strings.Contains(host, "soundcloud.com"):
		return "soundcloud"
	case strings.Contains(host, "flickr.com"):
		return "flickr"
	case strings.Contains(host, "tiktok.com"):
		return "tiktok"
	case strings.Contains(host, "dailymotion.com"):
		return "dailymotion"
	default:
		return ""
	}
}

func sanitizeClassToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' || r == '.' {
			b.WriteByte('-')
		}
	}
	return b.String()
}

func splitClassAttr(attrPart string) (classVal string, hasClass bool, other string) {
	re := regexp.MustCompile(`(?i)\sclass\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)
	loc := re.FindStringSubmatchIndex(attrPart)
	if loc == nil {
		return "", false, attrPart
	}
	full := attrPart[loc[0]:loc[1]]
	sub := re.FindStringSubmatch(attrPart)
	if len(sub) >= 5 {
		switch {
		case sub[2] != "":
			classVal = sub[2]
		case sub[3] != "":
			classVal = sub[3]
		default:
			classVal = sub[4]
		}
	}
	other = strings.TrimSpace(attrPart[:loc[0]] + attrPart[loc[1]:])
	if other != "" {
		other = " " + other
	}
	_ = full
	return classVal, true, other
}

func mergeClassList(existing, want string) string {
	parts := append(strings.Fields(existing), strings.Fields(want)...)
	return strings.Join(uniqueClassParts(parts), " ")
}

func uniqueClassParts(parts []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
