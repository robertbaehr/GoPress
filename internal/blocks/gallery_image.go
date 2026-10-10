package blocks

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reEmptyFigcaption = regexp.MustCompile(`(?is)<figcaption\b[^>]*>\s*</figcaption>`)
	reImgOpenTag      = regexp.MustCompile(`(?i)<img\b`)
	reFigureOpen      = regexp.MustCompile(`(?is)^(\s*<figure\b)([^>]*)(>)`)
)

// renderImage renders core/image (P5.48 / P5.65).
// Mirrors render_block_core_image subset: require <img>, honor data-id for
// gallery nesting, drop empty figcaptions when caption attr is empty, and
// synthesize markup from attrs when no saved HTML is present.
// P5.65: ensure wp-block-image + align classes; width/height/aspectRatio/scale/
// title/isDecorative/linkClass on synthesize and light saved-markup polish.
func (e *Env) renderImage(block Block) string {
	e = e.ensure()
	block = galleryImageDataIDBackcompat(block, false)

	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeImageFigure(e, block)
	}
	if !reImgOpenTag.MatchString(out) {
		return ""
	}

	if dataID := imageDataIDAttr(block); dataID != "" {
		out = injectDataIDOnIMG(out, dataID)
	}

	caption := strings.TrimSpace(attrString(block.Attrs, "caption"))
	if caption == "" {
		out = reEmptyFigcaption.ReplaceAllString(out, "")
	}

	out = addRootClass(out, "wp-block-image")
	if align := strings.TrimSpace(attrString(block.Attrs, "align")); align != "" {
		out = addRootClass(out, "align"+sanitizeClassToken(align))
	}
	return out
}

// renderGallery renders core/gallery (P5.48).
// Applies data-id backcompat onto nested image blocks, ensures gallery
// wrapper classes (columns / is-cropped), and can build from ids attrs.
func (e *Env) renderGallery(block Block) string {
	e = e.ensure()
	block = applyGalleryDataIDBackcompat(block)

	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeGalleryFromIDs(e, block)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	return ensureGalleryWrapperClasses(out, block.Attrs)
}

func applyGalleryDataIDBackcompat(block Block) Block {
	if len(block.InnerBlocks) == 0 {
		return block
	}
	inners := make([]Block, len(block.InnerBlocks))
	for i, inner := range block.InnerBlocks {
		inners[i] = galleryImageDataIDBackcompat(inner, true)
	}
	block.InnerBlocks = inners
	return block
}

// galleryImageDataIDBackcompat copies attrs.id → attrs["data-id"] for gallery children.
func galleryImageDataIDBackcompat(block Block, force bool) Block {
	if blockName(block) != "core/image" {
		return block
	}
	if block.Attrs == nil {
		block.Attrs = map[string]any{}
	}
	if !force {
		if _, ok := block.Attrs["data-id"]; ok {
			return block
		}
		return block
	}
	if _, ok := block.Attrs["data-id"]; ok {
		return block
	}
	id := attrInt64(block.Attrs, "id")
	if id <= 0 {
		return block
	}
	attrs := copyAttrs(block.Attrs)
	attrs["data-id"] = strconv.FormatInt(id, 10)
	block.Attrs = attrs
	return block
}

func imageDataIDAttr(block Block) string {
	if block.Attrs == nil {
		return ""
	}
	if v, ok := block.Attrs["data-id"]; ok {
		switch t := v.(type) {
		case string:
			return strings.TrimSpace(t)
		case float64:
			if t > 0 {
				return strconv.FormatInt(int64(t), 10)
			}
		case int64:
			if t > 0 {
				return strconv.FormatInt(t, 10)
			}
		case int:
			if t > 0 {
				return strconv.Itoa(t)
			}
		}
	}
	return ""
}

func injectDataIDOnIMG(htmlStr, dataID string) string {
	dataID = strings.TrimSpace(dataID)
	if dataID == "" || !reImgOpenTag.MatchString(htmlStr) {
		return htmlStr
	}
	if strings.Contains(strings.ToLower(htmlStr), `data-id="`) {
		return htmlStr
	}
	return reImgOpenTag.ReplaceAllString(htmlStr, `<img data-id="`+html.EscapeString(dataID)+`"`)
}

func synthesizeImageFigure(e *Env, block Block) string {
	url := strings.TrimSpace(attrString(block.Attrs, "url"))
	id := attrInt64(block.Attrs, "id")
	if url == "" && id > 0 && e.GetAttachmentURL != nil {
		url = strings.TrimSpace(e.GetAttachmentURL(id))
	}
	if url == "" {
		return ""
	}
	decorative := attrBool(block.Attrs, "isDecorative")
	alt := attrString(block.Attrs, "alt")
	if decorative {
		alt = ""
	}
	sizeSlug := strings.TrimSpace(attrString(block.Attrs, "sizeSlug"))
	if sizeSlug == "" {
		sizeSlug = "large"
	}
	class := "wp-block-image size-" + sizeSlug
	if align := strings.TrimSpace(attrString(block.Attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		class += " " + cn
	}

	width := strings.TrimSpace(attrString(block.Attrs, "width"))
	height := strings.TrimSpace(attrString(block.Attrs, "height"))
	aspect := strings.TrimSpace(attrString(block.Attrs, "aspectRatio"))
	scale := strings.TrimSpace(attrString(block.Attrs, "scale"))
	if width != "" || height != "" {
		class += " is-resized"
	}

	var imgClassParts []string
	if id > 0 {
		imgClassParts = append(imgClassParts, fmt.Sprintf("wp-image-%d", id))
	}
	imgAttrs := ` src="` + html.EscapeString(url) + `" alt="` + html.EscapeString(alt) + `"`
	if len(imgClassParts) > 0 {
		imgAttrs += ` class="` + html.EscapeString(strings.Join(imgClassParts, " ")) + `"`
	}
	if title := strings.TrimSpace(attrString(block.Attrs, "title")); title != "" && !decorative {
		imgAttrs += ` title="` + html.EscapeString(title) + `"`
	}
	if decorative {
		imgAttrs += ` role="presentation"`
	}
	var styleParts []string
	if width != "" {
		styleParts = append(styleParts, "width:"+sanitizeCSSLength(width))
	}
	if height != "" {
		styleParts = append(styleParts, "height:"+sanitizeCSSLength(height))
	}
	if aspect != "" {
		styleParts = append(styleParts, "aspect-ratio:"+sanitizeCSSToken(aspect))
	}
	if scale != "" {
		styleParts = append(styleParts, "object-fit:"+sanitizeCSSToken(scale))
	}
	if len(styleParts) > 0 {
		imgAttrs += ` style="` + html.EscapeString(strings.Join(styleParts, ";")) + `"`
	}
	img := `<img` + imgAttrs + `/>`

	href := strings.TrimSpace(attrString(block.Attrs, "href"))
	linkDest := strings.TrimSpace(attrString(block.Attrs, "linkDestination"))
	if href == "" && linkDest == "attachment" && id > 0 && e.GetAttachmentURL != nil {
		href = strings.TrimSpace(e.GetAttachmentURL(id))
	}
	if href == "" && (linkDest == "media" || linkDest == "file") {
		href = url
	}
	inner := img
	if href != "" {
		aAttrs := ` href="` + html.EscapeString(href) + `"`
		if t := strings.TrimSpace(attrString(block.Attrs, "linkTarget")); t != "" {
			aAttrs += ` target="` + html.EscapeString(t) + `"`
		}
		if r := strings.TrimSpace(attrString(block.Attrs, "rel")); r != "" {
			aAttrs += ` rel="` + html.EscapeString(r) + `"`
		}
		if lc := strings.TrimSpace(attrString(block.Attrs, "linkClass")); lc != "" {
			aAttrs += ` class="` + html.EscapeString(lc) + `"`
		}
		inner = `<a` + aAttrs + `>` + img + `</a>`
	}
	caption := strings.TrimSpace(attrString(block.Attrs, "caption"))
	capHTML := ""
	if caption != "" {
		capHTML = `<figcaption class="wp-element-caption">` + html.EscapeString(caption) + `</figcaption>`
	}
	return `<figure class="` + html.EscapeString(class) + `">` + inner + capHTML + `</figure>`
}

// sanitizeCSSLength allows numeric lengths with optional common CSS units.
func sanitizeCSSLength(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	// Bare numbers get px (block editor convention).
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return v + "px"
	}
	return sanitizeCSSToken(v)
}

func sanitizeCSSToken(v string) string {
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == '%' || r == '/' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func synthesizeGalleryFromIDs(e *Env, block Block) string {
	ids := galleryIDs(block.Attrs)
	if len(ids) == 0 || e.GetAttachmentURL == nil {
		return ""
	}
	columns := galleryColumns(block.Attrs)
	cropped := galleryImageCrop(block.Attrs)
	class := "wp-block-gallery has-nested-images"
	if columns > 0 {
		class += fmt.Sprintf(" columns-%d", columns)
	} else {
		class += " columns-default"
	}
	if cropped {
		class += " is-cropped"
	}
	var b strings.Builder
	b.WriteString(`<figure class="` + class + `">`)
	for _, id := range ids {
		url := strings.TrimSpace(e.GetAttachmentURL(id))
		if url == "" {
			continue
		}
		imgBlock := Block{
			BlockName: strPtr("core/image"),
			Attrs: map[string]any{
				"id":      id,
				"url":     url,
				"data-id": strconv.FormatInt(id, 10),
			},
		}
		b.WriteString(e.renderImage(imgBlock))
	}
	if cap := strings.TrimSpace(attrString(block.Attrs, "caption")); cap != "" {
		b.WriteString(`<figcaption class="blocks-gallery-caption wp-element-caption">`)
		b.WriteString(html.EscapeString(cap))
		b.WriteString(`</figcaption>`)
	}
	b.WriteString(`</figure>`)
	return b.String()
}

func ensureGalleryWrapperClasses(htmlStr string, attrs map[string]any) string {
	if !reFigureOpen.MatchString(htmlStr) {
		return htmlStr
	}
	columns := galleryColumns(attrs)
	cropped := galleryImageCrop(attrs)
	return reFigureOpen.ReplaceAllStringFunc(htmlStr, func(m string) string {
		parts := reFigureOpen.FindStringSubmatch(m)
		if len(parts) != 4 {
			return m
		}
		attrsPart := parts[2]
		classVal := extractHTMLAttr(attrsPart, "class")
		classes := strings.Fields(classVal)
		has := map[string]bool{}
		for _, c := range classes {
			has[c] = true
		}
		ensure := []string{"wp-block-gallery", "has-nested-images"}
		if columns > 0 {
			ensure = append(ensure, fmt.Sprintf("columns-%d", columns))
		} else if !hasClassPrefix(classes, "columns-") {
			ensure = append(ensure, "columns-default")
		}
		if cropped {
			ensure = append(ensure, "is-cropped")
		}
		for _, c := range ensure {
			if !has[c] {
				classes = append(classes, c)
				has[c] = true
			}
		}
		newClass := strings.Join(classes, " ")
		if classVal == "" {
			attrsPart = strings.TrimSpace(attrsPart + ` class="` + newClass + `"`)
		} else {
			attrsPart = replaceHTMLAttr(attrsPart, "class", newClass)
		}
		return parts[1] + attrsPart + parts[3]
	})
}

func galleryIDs(attrs map[string]any) []int64 {
	if attrs == nil {
		return nil
	}
	raw, ok := attrs["ids"]
	if !ok {
		return nil
	}
	var out []int64
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			switch t := item.(type) {
			case float64:
				if t > 0 {
					out = append(out, int64(t))
				}
			case int64:
				if t > 0 {
					out = append(out, t)
				}
			case int:
				if t > 0 {
					out = append(out, int64(t))
				}
			case string:
				if n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil && n > 0 {
					out = append(out, n)
				}
			}
		}
	}
	return out
}

func galleryColumns(attrs map[string]any) int {
	n := int(attrInt64(attrs, "columns"))
	if n < 0 {
		return 0
	}
	if n > 8 {
		return 8
	}
	return n
}

func galleryImageCrop(attrs map[string]any) bool {
	if attrs == nil {
		return true
	}
	v, ok := attrs["imageCrop"]
	if !ok {
		return true
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	case float64:
		return t != 0
	default:
		return true
	}
}

func extractHTMLAttr(attrs, name string) string {
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(name) + `\s*=\s*"([^"]*)"`)
	m := re.FindStringSubmatch(attrs)
	if len(m) == 2 {
		return m[1]
	}
	re2 := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(name) + `\s*=\s*'([^']*)'`)
	m = re2.FindStringSubmatch(attrs)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func replaceHTMLAttr(attrs, name, value string) string {
	re := regexp.MustCompile(`(?i)(\b` + regexp.QuoteMeta(name) + `\s*=\s*")([^"]*)(")`)
	if re.MatchString(attrs) {
		return re.ReplaceAllString(attrs, `${1}`+value+`${3}`)
	}
	re2 := regexp.MustCompile(`(?i)(\b` + regexp.QuoteMeta(name) + `\s*=\s*')([^']*)(')`)
	if re2.MatchString(attrs) {
		return re2.ReplaceAllString(attrs, `${1}`+value+`${3}`)
	}
	return strings.TrimSpace(attrs + ` ` + name + `="` + value + `"`)
}

func hasClassPrefix(classes []string, prefix string) bool {
	for _, c := range classes {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func copyAttrs(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}
