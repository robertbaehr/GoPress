package blocks

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	reCoverTag                 = regexp.MustCompile(`(?i)wp-block-cover\b`)
	reCoverInnerContainerStart = regexp.MustCompile(`(?i)<div\b[^>]*\bwp-block-cover__inner-container\b[^>]*>`)
)

// renderCover renders core/cover (P5.54 / P5.108 / P5.151).
// Prefers saved markup; synthesizes from url/id when empty; when useFeaturedImage
// is set with an image background, injects the featured image before the inner
// container (subset of render_block_core_cover). P5.108: className/anchor/align,
// contentPosition, isDark/is-light, video background synthesize, dimRatio overlay.
// P5.151: coerce align enum; fontSize + sizeSlug + alt polish; sanitize before supports.
func (e *Env) renderCover(block Block) string {
	e = e.ensure()
	attrs := sanitizeCoverBlockAttrs(block.Attrs)
	block.Attrs = attrs
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeCoverBlock(e, block)
	}
	if strings.TrimSpace(out) == "" || !reCoverTag.MatchString(out) {
		return ""
	}
	bgType := strings.TrimSpace(attrString(attrs, "backgroundType"))
	if bgType == "" {
		bgType = "image"
	}
	if bgType == "image" && attrBool(attrs, "useFeaturedImage") {
		out = e.applyCoverFeaturedImage(out, attrs)
	}
	return applyCoverAttrs(out, attrs)
}

// sanitizeCoverBlockAttrs coerces core/cover supports.align to the block.json set (P5.151).
// Allowed: left, center, right, wide, full (align: true). Other align values are dropped.
func sanitizeCoverBlockAttrs(attrs map[string]any) map[string]any {
	if attrs == nil {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	align := strings.TrimSpace(attrString(out, "align"))
	switch align {
	case "", "left", "center", "right", "wide", "full":
		// keep
	default:
		delete(out, "align")
	}
	return out
}

func synthesizeCoverBlock(e *Env, block Block) string {
	bgType := strings.TrimSpace(attrString(block.Attrs, "backgroundType"))
	if bgType == "" {
		bgType = "image"
	}
	url := coverURLFromAttrs(e, block.Attrs)
	useFeatured := attrBool(block.Attrs, "useFeaturedImage")
	if url == "" && !useFeatured && len(block.InnerBlocks) == 0 && bgType != "video" {
		return ""
	}
	dimRatio := 50
	if d, ok := attrInt(block.Attrs, "dimRatio"); ok {
		dimRatio = d
	}
	hasParallax := attrBool(block.Attrs, "hasParallax")
	isRepeated := attrBool(block.Attrs, "isRepeated")
	minHeight := strings.TrimSpace(attrString(block.Attrs, "minHeight"))
	if minHeight == "" {
		if h, ok := attrInt(block.Attrs, "minHeight"); ok && h > 0 {
			unit := strings.TrimSpace(attrString(block.Attrs, "minHeightUnit"))
			if unit == "" {
				unit = "px"
			}
			minHeight = fmt.Sprintf("%d%s", h, unit)
		}
	}

	class := coverClassFromAttrs(block.Attrs, dimRatio, hasParallax, isRepeated)
	style := ""
	if bgType == "image" && url != "" && (hasParallax || isRepeated) {
		style = "background-image:url(" + html.EscapeString(url) + ");"
	}
	if minHeight != "" {
		style += "min-height:" + html.EscapeString(minHeight) + ";"
	}

	tag := coverTagName(block.Attrs)
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(tag)
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(` class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if style != "" {
		b.WriteString(` style="`)
		b.WriteString(style)
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	if bgType == "video" && url != "" {
		b.WriteString(`<video class="wp-block-cover__video-background intrinsic-ignore" autoplay muted loop playsinline src="`)
		b.WriteString(html.EscapeString(url))
		b.WriteString(`"`)
		if poster := strings.TrimSpace(attrString(block.Attrs, "poster")); poster != "" {
			b.WriteString(` poster="`)
			b.WriteString(html.EscapeString(poster))
			b.WriteString(`"`)
		}
		b.WriteString(`></video>`)
	} else if bgType == "image" && url != "" && !hasParallax && !isRepeated {
		alt := strings.TrimSpace(attrString(block.Attrs, "alt"))
		b.WriteString(`<img class="wp-block-cover__image-background`)
		if slug := strings.TrimSpace(attrString(block.Attrs, "sizeSlug")); slug != "" {
			b.WriteString(" size-")
			b.WriteString(sanitizeClassToken(slug))
		}
		b.WriteString(`" alt="`)
		b.WriteString(html.EscapeString(alt))
		b.WriteString(`" src="`)
		b.WriteString(html.EscapeString(url))
		b.WriteString(`" data-object-fit="cover"/>`)
	}
	b.WriteString(`<span aria-hidden="true" class="`)
	b.WriteString(coverBackgroundClass(block.Attrs, dimRatio))
	b.WriteString(`"></span>`)
	b.WriteString(`<div class="wp-block-cover__inner-container">`)
	b.WriteString(e.renderInnerBlocks(block))
	b.WriteString(`</div></`)
	b.WriteString(tag)
	b.WriteString(`>`)
	return b.String()
}

func coverURLFromAttrs(e *Env, attrs map[string]any) string {
	url := strings.TrimSpace(attrString(attrs, "url"))
	if url != "" {
		return url
	}
	id := attrInt64(attrs, "id")
	if id > 0 && e != nil && e.GetAttachmentURL != nil {
		return strings.TrimSpace(e.GetAttachmentURL(id))
	}
	return ""
}

func (e *Env) applyCoverFeaturedImage(content string, attrs map[string]any) string {
	if e == nil || e.currentPost == nil || e.currentPost.FeaturedImageURL == "" {
		return content
	}
	url := e.currentPost.FeaturedImageURL
	alt := e.currentPost.FeaturedImageAlt
	focal := coverObjectPosition(attrs)
	hasParallax := attrBool(attrs, "hasParallax")
	isRepeated := attrBool(attrs, "isRepeated")

	var image string
	if hasParallax || isRepeated {
		class := "wp-block-cover__image-background"
		if hasParallax {
			class += " has-parallax"
		}
		if isRepeated {
			class += " is-repeated"
		}
		pos := focal
		if pos == "" {
			pos = "50% 50%"
		}
		image = fmt.Sprintf(
			`<div class="%s" style="background-position:%s;background-image:url(%s);"></div>`,
			class, html.EscapeString(pos), html.EscapeString(url),
		)
	} else {
		attr := ` class="wp-block-cover__image-background" data-object-fit="cover"`
		if focal != "" {
			attr += ` data-object-position="` + html.EscapeString(focal) + `"`
			attr += ` style="object-position:` + html.EscapeString(focal) + `;"`
		}
		image = `<img` + attr + ` alt="` + html.EscapeString(alt) + `" src="` + html.EscapeString(url) + `"/>`
	}

	loc := reCoverInnerContainerStart.FindStringIndex(content)
	if loc == nil {
		return content
	}
	return content[:loc[0]] + image + content[loc[0]:]
}

func coverObjectPosition(attrs map[string]any) string {
	if attrs == nil {
		return ""
	}
	fp, ok := attrs["focalPoint"].(map[string]any)
	if !ok || fp == nil {
		return ""
	}
	x, xok := focalFloat(fp["x"])
	y, yok := focalFloat(fp["y"])
	if !xok || !yok {
		return ""
	}
	return fmt.Sprintf("%.0f%% %.0f%%", x*100, y*100)
}

func focalFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	default:
		return 0, false
	}
}

func coverTagName(attrs map[string]any) string {
	tag := strings.ToLower(strings.TrimSpace(attrString(attrs, "tagName")))
	switch tag {
	case "div", "section", "header", "footer", "aside", "main":
		return tag
	default:
		return "div"
	}
}

func coverClassFromAttrs(attrs map[string]any, dimRatio int, hasParallax, isRepeated bool) string {
	class := "wp-block-cover"
	if hasParallax {
		class += " has-parallax"
	}
	if isRepeated {
		class += " is-repeated"
	}
	if dimRatio == 0 {
		class += " has-background-dim-0"
	} else {
		class += fmt.Sprintf(" has-background-dim-%d", dimRatio)
	}
	class += " has-background-dim"
	// P5.108: isDark defaults true in block.json; false → is-light.
	if attrs != nil {
		if _, ok := attrs["isDark"]; ok && !attrBool(attrs, "isDark") {
			class += " is-light"
		} else {
			class += " is-dark"
		}
	} else {
		class += " is-dark"
	}
	if pos := coverContentPositionClass(attrs); pos != "" {
		class += " has-custom-content-position " + pos
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	// P5.151: typography.fontSize preset on synthesize.
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		class += " has-" + sanitizeClassToken(fs) + "-font-size"
	}
	return class
}

func coverBackgroundClass(attrs map[string]any, dimRatio int) string {
	class := "wp-block-cover__background has-background-dim"
	if dimRatio == 0 {
		class += " has-background-dim-0"
	} else {
		class += fmt.Sprintf(" has-background-dim-%d", dimRatio)
	}
	if overlay := strings.TrimSpace(attrString(attrs, "overlayColor")); overlay != "" {
		class += " has-" + sanitizeClassToken(overlay) + "-background-color"
	}
	if grad := strings.TrimSpace(attrString(attrs, "gradient")); grad != "" {
		class += " has-" + sanitizeClassToken(grad) + "-gradient-background"
	}
	return class
}

func coverContentPositionClass(attrs map[string]any) string {
	pos := strings.TrimSpace(attrString(attrs, "contentPosition"))
	if pos == "" {
		return ""
	}
	// "top left" → is-position-top-left
	parts := strings.Fields(strings.ToLower(pos))
	if len(parts) == 0 {
		return ""
	}
	token := strings.Join(parts, "-")
	return "is-position-" + sanitizeClassToken(token)
}

// applyCoverAttrs polishes saved + synthesized cover markup (P5.108 / P5.151).
func applyCoverAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		cls := "align" + sanitizeClassToken(align)
		if !strings.Contains(out, cls) {
			out = addRootClass(out, cls)
		}
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	// P5.151: typography.fontSize preset class (also via applySupports).
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		out = addRootClass(out, "has-"+sanitizeClassToken(fs)+"-font-size")
	}
	if pos := coverContentPositionClass(attrs); pos != "" {
		if !strings.Contains(out, "has-custom-content-position") {
			out = addRootClass(out, "has-custom-content-position")
		}
		if !strings.Contains(out, pos) {
			out = addRootClass(out, pos)
		}
	}
	if attrs != nil {
		if _, ok := attrs["isDark"]; ok && !attrBool(attrs, "isDark") {
			out = addRootClass(out, "is-light")
		}
	}
	tag := coverTagName(attrs)
	if tag != "div" {
		out = retagRootElement(out, tag)
	}
	return out
}

func (e *Env) renderInnerBlocks(block Block) string {
	if len(block.InnerContent) > 0 {
		var b strings.Builder
		innerIdx := 0
		for _, chunk := range block.InnerContent {
			if chunk == nil {
				if innerIdx < len(block.InnerBlocks) {
					b.WriteString(e.RenderBlock(block.InnerBlocks[innerIdx]))
					innerIdx++
				}
				continue
			}
			if s, ok := chunk.(string); ok {
				b.WriteString(s)
			}
		}
		return b.String()
	}
	var b strings.Builder
	for _, inner := range block.InnerBlocks {
		b.WriteString(e.RenderBlock(inner))
	}
	return b.String()
}
