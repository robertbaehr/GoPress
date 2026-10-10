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

// renderCover renders core/cover (P5.54).
// Prefers saved markup; synthesizes from url/id when empty; when useFeaturedImage
// is set with an image background, injects the featured image before the inner
// container (subset of render_block_core_cover).
func (e *Env) renderCover(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeCoverBlock(e, block)
	}
	if strings.TrimSpace(out) == "" || !reCoverTag.MatchString(out) {
		return ""
	}
	bgType := strings.TrimSpace(attrString(block.Attrs, "backgroundType"))
	if bgType == "" {
		bgType = "image"
	}
	if bgType == "image" && attrBool(block.Attrs, "useFeaturedImage") {
		out = e.applyCoverFeaturedImage(out, block.Attrs)
	}
	return out
}

func synthesizeCoverBlock(e *Env, block Block) string {
	url := coverURLFromAttrs(e, block.Attrs)
	useFeatured := attrBool(block.Attrs, "useFeaturedImage")
	if url == "" && !useFeatured && len(block.InnerBlocks) == 0 {
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
			minHeight = fmt.Sprintf("%dpx", h)
		}
	}

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

	style := ""
	if url != "" && (hasParallax || isRepeated) {
		style = "background-image:url(" + html.EscapeString(url) + ");"
	}
	if minHeight != "" {
		style += "min-height:" + html.EscapeString(minHeight) + ";"
	}

	var b strings.Builder
	b.WriteString(`<div class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if style != "" {
		b.WriteString(` style="`)
		b.WriteString(style)
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	if url != "" && !hasParallax && !isRepeated {
		b.WriteString(`<img class="wp-block-cover__image-background" alt="" src="`)
		b.WriteString(html.EscapeString(url))
		b.WriteString(`" data-object-fit="cover"/>`)
	}
	b.WriteString(`<span aria-hidden="true" class="wp-block-cover__background has-background-dim"></span>`)
	b.WriteString(`<div class="wp-block-cover__inner-container">`)
	b.WriteString(e.renderInnerBlocks(block))
	b.WriteString(`</div></div>`)
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
