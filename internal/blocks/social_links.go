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
	// P5.83: customIconColor / customIconBackgroundColor fall back when *Value unset.
	iconColor := strings.TrimSpace(attrString(attrs, "iconColorValue"))
	if iconColor == "" {
		iconColor = strings.TrimSpace(attrString(attrs, "customIconColor"))
	}
	iconBg := strings.TrimSpace(attrString(attrs, "iconBackgroundColorValue"))
	if iconBg == "" {
		iconBg = strings.TrimSpace(attrString(attrs, "customIconBackgroundColor"))
	}
	ctx := socialLinksCtx{
		openInNewTab:             attrBool(attrs, "openInNewTab"),
		showLabels:               attrBool(attrs, "showLabels"),
		iconColorValue:           iconColor,
		iconBackgroundColorValue: iconBg,
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
	if strings.TrimSpace(attrString(attrs, "iconColorValue")) != "" ||
		strings.TrimSpace(attrString(attrs, "customIconColor")) != "" ||
		strings.TrimSpace(attrString(attrs, "iconColor")) != "" {
		parts = append(parts, "has-icon-color")
	}
	if strings.TrimSpace(attrString(attrs, "iconBackgroundColorValue")) != "" ||
		strings.TrimSpace(attrString(attrs, "customIconBackgroundColor")) != "" ||
		strings.TrimSpace(attrString(attrs, "iconBackgroundColor")) != "" {
		parts = append(parts, "has-icon-background-color")
	}
	if size := strings.TrimSpace(attrString(attrs, "size")); size != "" {
		parts = append(parts, "has-"+sanitizeClassToken(size)+"-icon-size")
	}
	// P5.83: visible-label + default flex layout classes (block supports layout).
	if attrBool(attrs, "showLabels") {
		parts = append(parts, "has-visible-labels")
	}
	parts = append(parts, "is-layout-flex", "wp-block-social-links-is-layout-flex")
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
		"share":     "Share Icon", // P5.133: WP block_core_social_link_services share name
		"bluesky":   "Bluesky",
		"whatsapp":  "WhatsApp",
		"mastodon":  "Mastodon",
		"telegram":  "Telegram",
		"threads":   "Threads",
		"tiktok":    "TikTok",
		"discord":   "Discord",
		"pinterest": "Pinterest",
		"reddit":    "Reddit",
	}
	if n, ok := names[service]; ok {
		return n
	}
	// P5.133: deepen catalog with remaining WP social-link services.
	if extra, ok := socialLinkServicesP5133[service]; ok && extra.Name != "" {
		return extra.Name
	}
	if service == "" {
		return "Share Icon"
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
		"bluesky":   `<path d="M6.3,4.2c2.3,1.7,4.8,5.3,5.7,7.2.9-1.9,3.4-5.4,5.7-7.2,1.7-1.3,4.3-2.2,4.3.9s-.4,5.2-.6,5.9c-.7,2.6-3.3,3.2-5.6,2.8,4,.7,5.1,3,2.9,5.3-5,5.2-6.7-2.8-6.7-2.8,0,0-1.7,8-6.7,2.8-2.2-2.3-1.2-4.6,2.9-5.3-2.3.4-4.9-.3-5.6-2.8-.2-.7-.6-5.3-.6-5.9,0-3.1,2.7-2.1,4.3-.9h0Z"></path>`,
		"mastodon":  `<path d="M23.193 7.879c0-5.206-3.411-6.732-3.411-6.732C18.062.357 15.108.025 12.041 0h-.076c-3.068.025-6.02.357-7.74 1.147 0 0-3.411 1.526-3.411 6.732 0 1.192-.023 2.618.015 4.129.124 5.092.934 10.109 5.641 11.355 2.17.574 4.034.695 5.535.612 2.722-.15 4.25-.972 4.25-.972l-.09-1.975s-1.945.613-4.129.539c-2.165-.074-4.449-.233-4.799-2.891a5.499 5.499 0 0 1-.048-.745s2.125.52 4.817.643c1.646.075 3.19-.097 4.758-.283 3.007-.359 5.625-2.212 5.954-3.905.517-2.665.475-6.507.475-6.507zm-4.024 6.709h-2.497V8.469c0-1.29-.543-1.944-1.628-1.944-1.2 0-1.802.776-1.802 2.312v3.349h-2.483v-3.35c0-1.536-.602-2.312-1.802-2.312-1.085 0-1.628.655-1.628 1.944v6.119H4.832V8.284c0-1.289.328-2.313.987-3.07.68-.758 1.569-1.146 2.674-1.146 1.278 0 2.246.491 2.886 1.474L12 6.585l.622-1.043c.64-.983 1.608-1.474 2.886-1.474 1.104 0 1.994.388 2.674 1.146.658.757.986 1.781.986 3.07v6.304z"></path>`,
		"telegram":  `<path d="M21.5 4.5 2.8 11.7c-.9.3-.9 1.6.1 1.9l4.4 1.4 1.7 5.3c.3.9 1.5 1.1 2.1.4l2.5-2.7 4.6 3.4c.8.6 1.9.1 2.1-.9L22.9 5.9c.2-1.1-.8-2-1.4-1.4z"></path>`,
		"threads":   `<path d="M16.3 11.3c-.1 0-.2-.1-.2-.1-.1-2.6-1.5-4-3.9-4-1.4 0-2.6.6-3.3 1.7l1.3.9c.5-.8 1.4-1 2-1 .8 0 1.4.2 1.7.7.3.3.5.8.5 1.3-.7-.1-1.4-.2-2.2-.1-2.2.1-3.7 1.4-3.6 3.2 0 .9.5 1.7 1.3 2.2.7.4 1.5.6 2.4.6 1.2-.1 2.1-.5 2.7-1.3.5-.6.8-1.4.9-2.4.6.3 1 .8 1.2 1.3.4.9.4 2.4-.8 3.6-1.1 1.1-2.3 1.5-4.3 1.5-2.1 0-3.8-.7-4.8-2S5.7 14.3 5.7 12c0-2.3.5-4.1 1.5-5.4 1.1-1.3 2.7-2 4.8-2 2.2 0 3.8.7 4.9 2 .5.7.9 1.5 1.2 2.5l1.5-.4c-.3-1.2-.8-2.2-1.5-3.1-1.3-1.7-3.3-2.6-6-2.6-2.6 0-4.7.9-6 2.6C4.9 7.2 4.3 9.3 4.3 12s.6 4.8 1.9 6.4c1.4 1.7 3.4 2.6 6 2.6 2.3 0 4-.6 5.3-2 1.8-1.8 1.7-4 1.1-5.4-.4-.9-1.2-1.7-2.3-2.3zm-4 3.8c-1 .1-2-.4-2-1.3 0-.7.5-1.5 2.1-1.6h.5c.6 0 1.1.1 1.6.2-.2 2.3-1.3 2.7-2.2 2.7z"></path>`,
		"tiktok":    `<path d="M16.7.03c1.75-.03 3.48-.01 5.21-.03.1 2.04.84 4.12 2.33 5.56 1.49 1.48 3.6 2.16 5.65 2.39v5.37c-1.92-.06-3.86-.46-5.6-1.29-.76-.34-1.47-.79-2.16-1.24 0 3.9.02 7.79-.03 11.67-.1 1.86-.72 3.72-1.8 5.25-1.74 2.56-4.77 4.22-7.88 4.28-1.91.11-3.81-.41-5.44-1.37C4.42 28.6 2.53 25.7 2.25 22.58c-.03-.67-.04-1.33-.02-1.98.24-2.54 1.5-4.96 3.44-6.62 2.21-1.92 5.3-2.84 8.2-2.3.03 1.98-.05 3.95-.05 5.92-1.32-.43-2.87-.31-4.03.5-.84.55-1.49 1.39-1.82 2.33-.28.68-.2 1.43-.18 2.15.32 2.19 2.42 4.03 4.67 3.83 1.49-.02 2.92-.88 3.69-2.15.25-.44.53-.9.55-1.42.13-2.39.08-4.76.1-7.15.01-5.38-.02-10.74.03-16.09z" transform="scale(0.75)"></path>`,
		"discord":   `<path d="M20.317 4.369A19.88 19.88 0 0 0 15.894 3a14.145 14.145 0 0 0-.719 1.518 19.205 19.205 0 0 0-5.351 0A14.183 14.183 0 0 0 9.104 3 19.896 19.896 0 0 0 4.682 4.369a18.921 18.921 0 0 0-3.012 12.52 19.929 19.929 0 0 0 6.081 3.097c.487-.65.922-1.339 1.3-2.061a12.445 12.445 0 0 1-1.958-.896c.165-.12.326-.246.483-.374a12.445 12.445 0 0 0 8.946 0c.157.128.318.253.483.374-.627.371-1.281.683-1.958.896.379.722.813 1.41 1.3 2.061a19.94 19.94 0 0 0 6.081-3.097 18.921 18.921 0 0 0-3.012-12.52ZM8.12 15.233c-1.202 0-2.184-1.09-2.184-2.431 0-1.34.97-2.431 2.184-2.431 1.213 0 2.202 1.09 2.184 2.431 0 1.341-.97 2.431-2.184 2.431Zm7.757 0c-1.202 0-2.184-1.09-2.184-2.431 0-1.34.97-2.431 2.184-2.431 1.213 0 2.202 1.09 2.184 2.431 0 1.341-.97 2.431-2.184 2.431Z"></path>`,
		"pinterest": `<path d="M12.289,2C6.617,2,3.606,5.648,3.606,9.622c0,1.846,1.025,4.146,2.666,4.878c0.25,0.111,0.381,0.063,0.439-0.169 c0.044-0.175,0.267-1.029,0.365-1.428c0.032-0.128,0.017-0.237-0.091-0.362C6.445,11.911,6.01,10.75,6.01,9.668 c0-2.777,2.194-5.464,5.933-5.464c3.23,0,5.49,2.108,5.49,5.122c0,3.407-1.794,5.768-4.13,5.768c-1.291,0-2.257-1.021-1.948-2.277 c0.372-1.495,1.089-3.112,1.089-4.191c0-0.967-0.542-1.775-1.663-1.775c-1.319,0-2.379,1.309-2.379,3.059 c0,1.115,0.394,1.869,0.394,1.869s-1.302,5.279-1.54,6.261c-0.405,1.666,0.053,4.368,0.094,4.604 c0.021,0.126,0.167,0.169,0.25,0.063c0.129-0.165,1.699-2.419,2.142-4.051c0.158-0.59,0.817-2.995,0.817-2.995 c0.43,0.784,1.681,1.446,3.013,1.446c3.963,0,6.822-3.494,6.822-7.833C20.394,5.112,16.849,2,12.289,2"></path>`,
		"reddit":    `<path d="M5.27 9.221A2.775 2.775 0 0 0 2.498 11.993a2.785 2.785 0 0 0 1.6 2.511 5.337 5.337 0 0 0 2.374 4.11 9.386 9.386 0 0 0 5.539 1.7 9.386 9.386 0 0 0 5.541-1.7 5.331 5.331 0 0 0 2.372-4.114 2.787 2.787 0 0 0 1.583-2.5 2.775 2.775 0 0 0-2.772-2.772 2.742 2.742 0 0 0-1.688.574 9.482 9.482 0 0 0-4.637-1.348v-.008a2.349 2.349 0 0 1 2.011-2.316 1.97 1.97 0 0 0 1.926 1.521 1.98 1.98 0 0 0 1.978-1.978 1.98 1.98 0 0 0-1.978-1.978 1.985 1.985 0 0 0-1.938 1.578 3.183 3.183 0 0 0-2.849 3.172v.011a9.463 9.463 0 0 0-4.59 1.35 2.741 2.741 0 0 0-1.688-.574Zm6.736 9.1a3.162 3.162 0 0 1-2.921-1.944.215.215 0 0 1 .014-.2.219.219 0 0 1 .168-.106 27.327 27.327 0 0 1 2.74-.133 27.357 27.357 0 0 1 2.74.133.219.219 0 0 1 .168.106.215.215 0 0 1 .014.2 3.158 3.158 0 0 1-2.921 1.944Z"></path>`,
	}
	path, ok := icons[service]
	if !ok {
		if service == "email" {
			path = icons["mail"]
		} else if extra, ok := socialLinkServicesP5133[service]; ok && extra.Path != "" {
			// P5.133: brand paths from block_core_social_link_services.
			path = extra.Path
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
