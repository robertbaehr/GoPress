package blocks

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	reMediaTextTag         = regexp.MustCompile(`(?i)wp-block-media-text\b`)
	reMediaTextFigureOpen  = regexp.MustCompile(`(?is)<figure\b[^>]*\bwp-block-media-text__media\b[^>]*>`)
	reMediaTextIsImageFill = regexp.MustCompile(`(?i)\bis-image-fill\b`)
	reMediaTextFigureInner = regexp.MustCompile(`(?is)(<figure\b[^>]*\bwp-block-media-text__media\b[^>]*>)(.*?)(</figure>)`)
)

// renderMediaText renders core/media-text (P5.55 / P5.109).
// Prefers saved markup; synthesizes from mediaUrl/mediaId when empty; when
// useFeaturedImage is set, injects the featured image into the media figure
// (subset of render_block_core_media_text). P5.109: className/anchor/align polish
// and href/linkTarget/rel wrapping around the media figure contents.
func (e *Env) renderMediaText(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" || !reMediaTextTag.MatchString(out) {
		out = synthesizeMediaTextBlock(e, block)
	}
	if strings.TrimSpace(out) == "" || !reMediaTextTag.MatchString(out) {
		return ""
	}
	if attrBool(block.Attrs, "useFeaturedImage") {
		out = e.applyMediaTextFeaturedImage(out, block.Attrs)
	}
	return applyMediaTextAttrs(out, block.Attrs)
}

func synthesizeMediaTextBlock(e *Env, block Block) string {
	mediaURL := mediaTextURLFromAttrs(e, block.Attrs)
	useFeatured := attrBool(block.Attrs, "useFeaturedImage")
	inner := e.renderInnerBlocks(block)
	if mediaURL == "" && !useFeatured && strings.TrimSpace(inner) == "" {
		return ""
	}

	mediaPos := strings.ToLower(strings.TrimSpace(attrString(block.Attrs, "mediaPosition")))
	if mediaPos == "" {
		mediaPos = "left"
	}
	mediaType := strings.ToLower(strings.TrimSpace(attrString(block.Attrs, "mediaType")))
	if mediaType == "" {
		mediaType = "image"
	}
	mediaWidth := 50
	if w, ok := attrInt(block.Attrs, "mediaWidth"); ok && w > 0 {
		mediaWidth = w
	}
	stacked := true
	if block.Attrs != nil {
		if _, ok := block.Attrs["isStackedOnMobile"]; ok {
			stacked = attrBool(block.Attrs, "isStackedOnMobile")
		}
	}
	imageFill := attrBool(block.Attrs, "imageFill")
	vAlign := strings.TrimSpace(attrString(block.Attrs, "verticalAlignment"))

	class := mediaTextClassFromAttrs(block.Attrs, mediaPos, stacked, imageFill, vAlign)

	style := fmt.Sprintf("grid-template-columns:%d%% auto", mediaWidth)
	if mediaPos == "right" {
		style = fmt.Sprintf("grid-template-columns:auto %d%%", mediaWidth)
	}

	var media strings.Builder
	media.WriteString(`<figure class="wp-block-media-text__media">`)
	mediaInner := ""
	if mediaURL != "" {
		if mediaType == "video" {
			mediaInner = `<video controls src="` + html.EscapeString(mediaURL) + `"></video>`
		} else {
			alt := strings.TrimSpace(attrString(block.Attrs, "mediaAlt"))
			imgClass := "wp-image"
			if id := attrInt64(block.Attrs, "mediaId"); id > 0 {
				imgClass = fmt.Sprintf("wp-image-%d", id)
			}
			slug := strings.TrimSpace(attrString(block.Attrs, "mediaSizeSlug"))
			if slug == "" {
				slug = "full"
			}
			imgClass += " size-" + html.EscapeString(slug)
			mediaInner = `<img src="` + html.EscapeString(mediaURL) + `" alt="` + html.EscapeString(alt) + `" class="` + imgClass + `"/>`
		}
	}
	mediaInner = wrapMediaTextHref(mediaInner, block.Attrs)
	media.WriteString(mediaInner)
	media.WriteString(`</figure>`)

	content := `<div class="wp-block-media-text__content">` + inner + `</div>`

	var b strings.Builder
	b.WriteString(`<div`)
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(` class="`)
	b.WriteString(class)
	b.WriteString(`" style="`)
	b.WriteString(style)
	b.WriteString(`">`)
	if mediaPos == "right" {
		b.WriteString(content)
		b.WriteString(media.String())
	} else {
		b.WriteString(media.String())
		b.WriteString(content)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func mediaTextClassFromAttrs(attrs map[string]any, mediaPos string, stacked, imageFill bool, vAlign string) string {
	class := "wp-block-media-text"
	if mediaPos == "right" {
		class += " has-media-on-the-right"
	}
	if stacked {
		class += " is-stacked-on-mobile"
	}
	if imageFill {
		class += " is-image-fill-element"
	}
	if vAlign != "" {
		class += " is-vertically-aligned-" + html.EscapeString(vAlign)
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" && align != "none" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	return class
}

func mediaTextURLFromAttrs(e *Env, attrs map[string]any) string {
	url := strings.TrimSpace(attrString(attrs, "mediaUrl"))
	if url != "" {
		return url
	}
	id := attrInt64(attrs, "mediaId")
	if id > 0 && e != nil && e.GetAttachmentURL != nil {
		return strings.TrimSpace(e.GetAttachmentURL(id))
	}
	return ""
}

func (e *Env) applyMediaTextFeaturedImage(content string, attrs map[string]any) string {
	if e == nil || e.currentPost == nil || e.currentPost.FeaturedImageURL == "" {
		return content
	}
	url := e.currentPost.FeaturedImageURL
	alt := e.currentPost.FeaturedImageAlt
	imageFill := attrBool(attrs, "imageFill")
	focal := coverObjectPosition(attrs)
	if focal == "" {
		focal = "50% 50%"
	}
	slug := strings.TrimSpace(attrString(attrs, "mediaSizeSlug"))
	if slug == "" {
		slug = "full"
	}
	imgClass := "wp-block-media-text__featured_image size-" + html.EscapeString(slug)

	img := `<img class="` + imgClass + `" src="` + html.EscapeString(url) + `" alt="` + html.EscapeString(alt) + `"`
	if imageFill {
		img += ` style="object-position:` + html.EscapeString(focal) + `;"`
	}
	img += `/>`

	if imageFill {
		content = reMediaTextIsImageFill.ReplaceAllStringFunc(content, func(m string) string {
			if strings.Contains(strings.ToLower(m), "is-image-fill-element") {
				return m
			}
			return "is-image-fill-element"
		})
		// Prefer WP 6.6+ class name when the deprecated is-image-fill is present.
		content = strings.ReplaceAll(content, " is-image-fill ", " is-image-fill-element ")
		content = strings.ReplaceAll(content, ` is-image-fill"`, ` is-image-fill-element"`)
	}

	loc := reMediaTextFigureOpen.FindStringIndex(content)
	if loc == nil {
		return content
	}
	// Insert featured image immediately after the opening figure tag.
	insertAt := loc[1]
	return content[:insertAt] + img + content[insertAt:]
}

// wrapMediaTextHref wraps media markup in an anchor when href is set (P5.109).
func wrapMediaTextHref(inner string, attrs map[string]any) string {
	href := strings.TrimSpace(attrString(attrs, "href"))
	if href == "" {
		href = strings.TrimSpace(attrString(attrs, "mediaLink"))
	}
	if href == "" || strings.TrimSpace(inner) == "" {
		return inner
	}
	// Avoid double-wrapping when saved markup already has an anchor.
	if strings.Contains(strings.ToLower(inner), "<a ") {
		return inner
	}
	var b strings.Builder
	b.WriteString(`<a href="`)
	b.WriteString(html.EscapeString(href))
	b.WriteString(`"`)
	if target := strings.TrimSpace(attrString(attrs, "linkTarget")); target != "" {
		b.WriteString(` target="`)
		b.WriteString(html.EscapeString(target))
		b.WriteString(`"`)
	}
	if rel := strings.TrimSpace(attrString(attrs, "rel")); rel != "" {
		b.WriteString(` rel="`)
		b.WriteString(html.EscapeString(rel))
		b.WriteString(`"`)
	}
	if linkClass := strings.TrimSpace(attrString(attrs, "linkClass")); linkClass != "" {
		b.WriteString(` class="`)
		b.WriteString(html.EscapeString(linkClass))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(inner)
	b.WriteString(`</a>`)
	return b.String()
}

// applyMediaTextAttrs polishes saved + synthesized media-text markup (P5.109).
func applyMediaTextAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" && align != "none" {
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
	// When synthesizing already wrapped href; for saved markup with empty figure
	// media and href attrs, wrap figure contents if not already linked.
	href := strings.TrimSpace(attrString(attrs, "href"))
	if href == "" {
		href = strings.TrimSpace(attrString(attrs, "mediaLink"))
	}
	if href != "" && !strings.Contains(strings.ToLower(out), `href="`+strings.ToLower(href)) {
		out = reMediaTextFigureInner.ReplaceAllStringFunc(out, func(m string) string {
			parts := reMediaTextFigureInner.FindStringSubmatch(m)
			if len(parts) != 4 {
				return m
			}
			inner := parts[2]
			if strings.Contains(strings.ToLower(inner), "<a ") {
				return m
			}
			return parts[1] + wrapMediaTextHref(inner, attrs) + parts[3]
		})
	}
	return out
}
