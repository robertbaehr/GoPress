package blocks

import (
	"html"
	"net/mail"
	"net/url"
	"strings"
)

// socialLinksCtx is parent context provided by core/social-links to children (P5.50).
type socialLinksCtx struct {
	openInNewTab             bool
	showLabels               bool
	iconColorValue           string
	iconBackgroundColorValue string
	hasIconColor             bool
	hasIconBackgroundColor   bool
}

// renderSocialLinks renders core/social-links (P5.50).
// Prefer saved InnerContent wrapper; synthesize a <ul> when missing.
func (e *Env) renderSocialLinks(block Block) string {
	e = e.ensure()
	prev := e.socialCtx
	e.socialCtx = socialLinksCtxFromAttrs(block.Attrs)
	defer func() { e.socialCtx = prev }()

	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" && len(block.InnerBlocks) == 0 {
		return ""
	}
	// Saved markup usually includes the <ul class="wp-block-social-links…"> wrapper.
	// When only children rendered (no wrapper HTML), synthesize the list.
	if !strings.Contains(out, "wp-block-social-links") {
		var b strings.Builder
		b.WriteString(`<ul class="`)
		b.WriteString(html.EscapeString(socialLinksClassList(block.Attrs)))
		b.WriteString(`">`)
		if strings.TrimSpace(out) != "" {
			b.WriteString(out)
		} else {
			for _, inner := range block.InnerBlocks {
				b.WriteString(e.RenderBlock(inner))
			}
		}
		b.WriteString(`</ul>`)
		out = b.String()
	}
	return out
}

// renderSocialLink renders core/social-link (P5.50).
// Self-closing blocks have no saved HTML; synthesize the <li><a>…</a></li> markup.
func (e *Env) renderSocialLink(block Block) string {
	e = e.ensure()
	urlStr := strings.TrimSpace(attrString(block.Attrs, "url"))
	if urlStr == "" {
		return ""
	}
	urlStr = normalizeSocialURL(urlStr)
	service := strings.TrimSpace(attrString(block.Attrs, "service"))
	if service == "" {
		service = "share"
	}
	serviceKey := strings.ToLower(service)
	label := strings.TrimSpace(attrString(block.Attrs, "label"))
	if label == "" {
		label = socialLinkServiceName(serviceKey)
	}
	rel := strings.TrimSpace(attrString(block.Attrs, "rel"))
	ctx := e.socialCtx
	openNew := ctx.openInNewTab || attrBool(block.Attrs, "openInNewTab")
	showLabels := ctx.showLabels || attrBool(block.Attrs, "showLabels")

	classParts := []string{"wp-social-link", "wp-social-link-" + sanitizeClassToken(serviceKey)}
	if ctx.hasIconColor {
		classParts = append(classParts, "has-text-color")
	}
	if ctx.hasIconBackgroundColor {
		classParts = append(classParts, "has-background")
	}

	var styleParts []string
	if ctx.iconColorValue != "" {
		styleParts = append(styleParts, "color:"+ctx.iconColorValue)
	}
	if ctx.iconBackgroundColorValue != "" {
		styleParts = append(styleParts, "background-color:"+ctx.iconBackgroundColorValue)
	}

	var b strings.Builder
	b.WriteString(`<li class="`)
	b.WriteString(html.EscapeString(strings.Join(uniqueClassParts(classParts), " ")))
	b.WriteString(`"`)
	if len(styleParts) > 0 {
		b.WriteString(` style="`)
		b.WriteString(html.EscapeString(strings.Join(styleParts, ";")))
		b.WriteString(`"`)
	}
	b.WriteString(`><a href="`)
	b.WriteString(html.EscapeString(urlStr))
	b.WriteString(`" class="wp-block-social-link-anchor"`)
	if openNew {
		b.WriteString(` target="_blank"`)
		rel = strings.TrimSpace(rel + " noopener nofollow")
	}
	if rel != "" {
		b.WriteString(` rel="`)
		b.WriteString(html.EscapeString(rel))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(socialLinkIconSVG(serviceKey))
	labelClass := "wp-block-social-link-label"
	if !showLabels {
		labelClass += " screen-reader-text"
	}
	b.WriteString(`<span class="`)
	b.WriteString(labelClass)
	b.WriteString(`">`)
	b.WriteString(html.EscapeString(label))
	b.WriteString(`</span></a></li>`)
	return b.String()
}

func socialLinksCtxFromAttrs(attrs map[string]any) socialLinksCtx {
	ctx := socialLinksCtx{
		openInNewTab:             attrBool(attrs, "openInNewTab"),
		showLabels:               attrBool(attrs, "showLabels"),
		iconColorValue:           strings.TrimSpace(attrString(attrs, "iconColorValue")),
		iconBackgroundColorValue: strings.TrimSpace(attrString(attrs, "iconBackgroundColorValue")),
	}
	if ctx.iconColorValue != "" || strings.TrimSpace(attrString(attrs, "iconColor")) != "" {
		ctx.hasIconColor = true
	}
	if ctx.iconBackgroundColorValue != "" || strings.TrimSpace(attrString(attrs, "iconBackgroundColor")) != "" {
		ctx.hasIconBackgroundColor = true
	}
	return ctx
}

func socialLinksClassList(attrs map[string]any) string {
	parts := []string{"wp-block-social-links"}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		parts = append(parts, "align"+sanitizeClassToken(align))
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		parts = append(parts, strings.Fields(cn)...)
	}
	if strings.TrimSpace(attrString(attrs, "iconColorValue")) != "" || strings.TrimSpace(attrString(attrs, "iconColor")) != "" {
		parts = append(parts, "has-icon-color")
	}
	if strings.TrimSpace(attrString(attrs, "iconBackgroundColorValue")) != "" || strings.TrimSpace(attrString(attrs, "iconBackgroundColor")) != "" {
		parts = append(parts, "has-icon-background-color")
	}
	if size := strings.TrimSpace(attrString(attrs, "size")); size != "" {
		parts = append(parts, "has-"+sanitizeClassToken(size)+"-icon-size")
	}
	return strings.Join(uniqueClassParts(parts), " ")
}

func normalizeSocialURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "mailto:") || strings.HasPrefix(raw, "tel:") {
		return raw
	}
	if _, err := mail.ParseAddress(raw); err == nil && !strings.Contains(raw, "://") {
		return "mailto:" + raw
	}
	if u, err := url.Parse(raw); err == nil && u.Scheme != "" {
		return raw
	}
	return "https://" + raw
}

func socialLinkServiceName(service string) string {
	names := map[string]string{
		"facebook":  "Facebook",
		"twitter":   "Twitter",
		"x":         "X",
		"instagram": "Instagram",
		"wordpress": "WordPress",
		"youtube":   "YouTube",
		"linkedin":  "LinkedIn",
		"github":    "GitHub",
		"mail":      "Mail",
		"email":     "Mail",
		"share":     "Share Link",
		"bluesky":   "Bluesky",
		"whatsapp":  "WhatsApp",
		"mastodon":  "Mastodon",
	}
	if n, ok := names[service]; ok {
		return n
	}
	if service == "" {
		return "Share Link"
	}
	// Title-case unknown service slug.
	return strings.ToUpper(service[:1]) + service[1:]
}

func socialLinkIconSVG(service string) string {
	// Compact brand paths (WP social-link subset); unknown services use share.
	icons := map[string]string{
		"facebook":  `<path d="M12 2C6.5 2 2 6.5 2 12c0 5 3.7 9.1 8.4 9.9v-7H7.9V12h2.5V9.8c0-2.5 1.5-3.9 3.8-3.9 1.1 0 2.2.2 2.2.2v2.5h-1.3c-1.2 0-1.6.8-1.6 1.6V12h2.8l-.4 2.9h-2.3v7C18.3 21.1 22 17 22 12c0-5.5-4.5-10-10-10z"></path>`,
		"twitter":   `<path d="M22.23 5.924c-.77.343-1.596.574-2.463.678a4.29 4.29 0 0 0 1.882-2.37 8.577 8.577 0 0 1-2.722 1.04 4.28 4.28 0 0 0-7.293 3.902 12.148 12.148 0 0 1-8.82-4.474 4.268 4.268 0 0 0 1.324 5.71 4.26 4.26 0 0 1-1.94-.536v.054a4.28 4.28 0 0 0 3.432 4.197 4.3 4.3 0 0 1-1.934.074 4.285 4.285 0 0 0 4 2.973 8.582 8.582 0 0 1-5.314 1.832c-.345 0-.685-.02-1.02-.06a12.11 12.11 0 0 0 6.556 1.92c7.868 0 12.174-6.52 12.174-12.174 0-.185-.004-.37-.013-.553a8.695 8.695 0 0 0 2.135-2.215z"></path>`,
		"x":         `<path d="M13.982 10.622 20.54 3h-1.554l-5.693 6.618L8.745 3H3.5l6.876 10.007L3.5 21h1.554l6.012-6.989L15.868 21H21.11l-7.128-10.378Zm-2.128 2.473-.697-.997L5.61 4.17h2.387l4.47 6.39.696.996 5.868 8.387h-2.386l-4.791-6.848Z"></path>`,
		"instagram": `<path d="M12 2c2.717 0 3.056.01 4.122.06 1.065.05 1.79.217 2.428.465.66.254 1.216.598 1.772 1.153a4.908 4.908 0 0 1 1.153 1.772c.247.637.415 1.363.465 2.428.047 1.066.06 1.405.06 4.122 0 2.717-.01 3.056-.06 4.122-.05 1.065-.218 1.79-.465 2.428a4.883 4.883 0 0 1-1.153 1.772 4.915 4.915 0 0 1-1.772 1.153c-.637.247-1.363.415-2.428.465-1.066.047-1.405.06-4.122.06-2.717 0-3.056-.01-4.122-.06-1.065-.05-1.79-.218-2.428-.465a4.89 4.89 0 0 1-1.772-1.153 4.904 4.904 0 0 1-1.153-1.772c-.248-.637-.415-1.363-.465-2.428C2.013 15.056 2 14.717 2 12c0-2.717.01-3.056.06-4.122.05-1.066.217-1.79.465-2.428a4.88 4.88 0 0 1 1.153-1.772A4.897 4.897 0 0 1 5.45 2.525c.638-.248 1.362-.415 2.428-.465C8.944 2.013 9.283 2 12 2zm0 5a5 5 0 1 0 0 10 5 5 0 0 0 0-10zm6.5-.25a1.25 1.25 0 1 0-2.5 0 1.25 1.25 0 0 0 2.5 0zM12 9a3 3 0 1 1 0 6 3 3 0 0 1 0-6z"></path>`,
		"wordpress": `<path d="M12.158 12.786l-2.698 7.84c.806.236 1.657.365 2.54.365 1.047 0 2.05-.18 2.986-.51-.024-.037-.046-.078-.065-.123l-2.763-7.572zM3.01 12c0 3.486 1.995 6.5 4.888 7.975l-4.132-11.33A8.935 8.935 0 0 0 3.01 12zm15.934-.5c0-1.086-.39-1.836-.724-2.422-.445-.724-.86-1.333-.86-2.056 0-.805.613-1.555 1.475-1.555.04 0 .076.004.114.006A8.958 8.958 0 0 0 12 3.01c-2.723 0-5.15 1.213-6.81 3.125h.302c.89 0 2.266-.108 2.266-.108.46-.026.513.648.053.702 0 0-.462.054-1.002.08l3.188 9.488 1.916-5.744-1.365-3.744c-.46-.027-.894-.08-.894-.08-.46-.027-.406-.728.053-.702 0 0 1.403.108 2.24.108.89 0 2.266-.108 2.266-.108.46-.026.513.648.053.702 0 0-.463.054-.1.002.08l3.166 9.425.875-2.922c.378-1.216.666-2.088.666-2.843zm1.502-4.97c.04.286.062.594.062.926 0 .912-.17 1.938-.68 3.222l-2.73 7.892A8.986 8.986 0 0 0 21 12a8.95 8.95 0 0 0-.554-5.47z"></path>`,
		"youtube":   `<path d="M21.8 8s-.195-1.377-.795-1.984c-.76-.797-1.613-.8-2.004-.847-2.799-.203-6.996-.203-6.996-.203h-.01s-4.197 0-6.996.202c-.39.046-1.242.05-2.003.846C2.395 6.623 2.2 8 2.2 8S2 9.62 2 11.24v1.517c0 1.618.2 3.237.2 3.237s.195 1.378.795 1.985c.76.797 1.76.77 2.205.855 1.6.153 6.8.2 6.8.2s4.203-.006 7.001-.209c.391-.047 1.244-.05 2.005-.847.6-.606.795-1.985.795-1.985s.2-1.618.2-3.237V11.24C22 9.62 21.8 8 21.8 8zM9.935 14.595V9.21l5.403 2.702-5.403 2.683z"></path>`,
		"linkedin":  `<path d="M19.7 3H4.3A1.3 1.3 0 0 0 3 4.3v15.4A1.3 1.3 0 0 0 4.3 21h15.4a1.3 1.3 0 0 0 1.3-1.3V4.3A1.3 1.3 0 0 0 19.7 3zM8.34 18.34H5.67v-8.59h2.67v8.59zM7 8.63a1.55 1.55 0 1 1 0-3.1 1.55 1.55 0 0 1 0 3.1zm11.34 9.71h-2.67v-4.17c0-1-.02-2.28-1.39-2.28-1.39 0-1.6 1.08-1.6 2.2v4.25H10V9.75h2.56v1.17h.04c.36-.68 1.23-1.39 2.53-1.39 2.7 0 3.2 1.78 3.2 4.09v4.72z"></path>`,
		"github":    `<path fill-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 12 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0 0 22 12.017C22 6.484 17.522 2 12 2z"></path>`,
		"mail":      `<path d="M19 5H5c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm.5 12.5h-15v-8.93l7.06 4.5a.85.85 0 0 0 .88 0l7.06-4.5v8.93zm-7.06-6.74L4.5 6.5h15l-7.06 4.26z"></path>`,
		"whatsapp":  `<path d="M12.04 2c-5.46 0-9.91 4.45-9.91 9.91 0 1.75.46 3.45 1.32 4.95L2.05 22l5.25-1.38c1.45.79 3.08 1.21 4.74 1.21 5.46 0 9.91-4.45 9.91-9.91 0-2.65-1.03-5.14-2.9-7.01A9.815 9.815 0 0 0 12.04 2zm.01 1.67c2.2 0 4.26.86 5.82 2.42a8.225 8.225 0 0 1 2.41 5.83c0 4.54-3.7 8.23-8.24 8.23-1.48 0-2.93-.39-4.19-1.15l-.3-.18-3.12.82.83-3.04-.2-.32a8.2 8.2 0 0 1-1.26-4.36c0-4.54 3.7-8.24 8.25-8.24z"></path>`,
	}
	path, ok := icons[service]
	if !ok {
		if service == "email" {
			path = icons["mail"]
		} else {
			// Generic share icon.
			path = `<path d="M18 16.08c-.76 0-1.44.3-1.96.77L8.91 12.7c.05-.23.09-.46.09-.7s-.04-.47-.09-.7l7.05-4.11c.54.5 1.25.81 2.04.81 1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3c0 .24.04.47.09.7L8.04 9.81C7.5 9.31 6.79 9 6 9c-1.66 0-3 1.34-3 3s1.34 3 3 3c.79 0 1.5-.31 2.04-.81l7.12 4.16c-.05.21-.08.43-.08.65 0 1.61 1.31 2.92 2.92 2.92s2.92-1.31 2.92-2.92-1.31-2.92-2.92-2.92z"></path>`
		}
	}
	return `<svg width="24" height="24" viewBox="0 0 24 24" version="1.1" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">` + path + `</svg>`
}

func attrBool(attrs map[string]any, key string) bool {
	if attrs == nil {
		return false
	}
	v, ok := attrs[key]
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
