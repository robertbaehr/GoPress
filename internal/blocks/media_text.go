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
)

// renderMediaText renders core/media-text (P5.55).
// Prefers saved markup; synthesizes from mediaUrl/mediaId when empty; when
// useFeaturedImage is set, injects the featured image into the media figure
// (subset of render_block_core_media_text).
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
	return out
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

	style := fmt.Sprintf("grid-template-columns:%d%% auto", mediaWidth)
	if mediaPos == "right" {
		style = fmt.Sprintf("grid-template-columns:auto %d%%", mediaWidth)
	}

	var media strings.Builder
	media.WriteString(`<figure class="wp-block-media-text__media">`)
	if mediaURL != "" {
		if mediaType == "video" {
			media.WriteString(`<video controls src="`)
			media.WriteString(html.EscapeString(mediaURL))
			media.WriteString(`"></video>`)
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
			media.WriteString(`<img src="`)
			media.WriteString(html.EscapeString(mediaURL))
			media.WriteString(`" alt="`)
			media.WriteString(html.EscapeString(alt))
			media.WriteString(`" class="`)
			media.WriteString(imgClass)
			media.WriteString(`"/>`)
		}
	}
	media.WriteString(`</figure>`)

	content := `<div class="wp-block-media-text__content">` + inner + `</div>`

	var b strings.Builder
	b.WriteString(`<div class="`)
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
