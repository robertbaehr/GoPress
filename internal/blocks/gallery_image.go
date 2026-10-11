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

// renderImage renders core/image (P5.48 / P5.65 / P5.113).
// Mirrors render_block_core_image subset: require <img>, honor data-id for
// gallery nesting, drop empty figcaptions when caption attr is empty, and
// synthesize markup from attrs when no saved HTML is present.
// P5.65: ensure wp-block-image + align classes; width/height/aspectRatio/scale/
// title/isDecorative/linkClass on synthesize and light saved-markup polish.
// P5.113: className/anchor/sizeSlug polish on saved and synthesized markup;
// is-resized when width/height present; rounded border style class.
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

	return applyImageAttrs(out, block.Attrs)
}

// applyImageAttrs polishes className/align/sizeSlug/anchor/is-resized on image roots (P5.113).
func applyImageAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	if strings.TrimSpace(out) == "" {
		return out
	}
	out = addRootClass(out, "wp-block-image")
	sizeSlug := strings.TrimSpace(attrString(attrs, "sizeSlug"))
	if sizeSlug == "" {
		sizeSlug = "large"
	}
	sizeClass := "size-" + sanitizeClassToken(sizeSlug)
	if !hasRootClass(out, sizeClass) && !hasRootClassPrefix(out, "size-") {
		out = addRootClass(out, sizeClass)
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		cls := "align" + sanitizeClassToken(align)
		if !hasRootClass(out, cls) {
			out = addRootClass(out, cls)
		}
	}
	width := strings.TrimSpace(attrString(attrs, "width"))
	height := strings.TrimSpace(attrString(attrs, "height"))
	if (width != "" || height != "") && !hasRootClass(out, "is-resized") {
		out = addRootClass(out, "is-resized")
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	// style.border.radius → is-style-rounded (core/image rounded block style).
	if imageHasRoundedBorder(attrs) {
		out = addRootClass(out, "is-style-rounded")
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	return out
}

func imageHasRoundedBorder(attrs map[string]any) bool {
	if attrs == nil {
		return false
	}
	style, _ := attrs["style"].(map[string]any)
	if style == nil {
		return false
	}
	border, _ := style["border"].(map[string]any)
	if border == nil {
		return false
	}
	switch r := border["radius"].(type) {
	case string:
		return strings.TrimSpace(r) != "" && strings.TrimSpace(r) != "0"
	case float64:
		return r > 0
	case map[string]any:
		return true
	default:
		return false
	}
}

func hasRootClass(htmlFragment, className string) bool {
	className = strings.TrimSpace(className)
	if className == "" {
		return false
	}
	re := regexp.MustCompile(`(?i)\bclass\s*=\s*["']([^"']*)["']`)
	m := re.FindStringSubmatch(htmlFragment)
	if len(m) < 2 {
		return false
	}
	for _, c := range strings.Fields(m[1]) {
		if c == className {
			return true
		}
	}
	return false
}

func hasRootClassPrefix(htmlFragment, prefix string) bool {
	re := regexp.MustCompile(`(?i)\bclass\s*=\s*["']([^"']*)["']`)
	m := re.FindStringSubmatch(htmlFragment)
	if len(m) < 2 {
		return false
	}
	for _, c := range strings.Fields(m[1]) {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

// renderGallery renders core/gallery (P5.48 / P5.110).
// Applies data-id backcompat onto nested image blocks, ensures gallery
// wrapper classes (columns / is-cropped), and can build from ids attrs.
// P5.110: className/align/anchor polish; linkTo/sizeSlug/linkTarget on
// synthesized ids galleries; fixedHeight class when enabled.
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
	out = ensureGalleryWrapperClasses(out, block.Attrs)
	return applyGalleryAttrs(out, block.Attrs)
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
	class := "wp-block-image size-" + sanitizeClassToken(sizeSlug)
	if align := strings.TrimSpace(attrString(block.Attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		class += " " + cn
	}
	if imageHasRoundedBorder(block.Attrs) {
		class += " is-style-rounded"
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
	idAttr := ""
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		idAttr = ` id="` + html.EscapeString(anchor) + `"`
	}
	return `<figure` + idAttr + ` class="` + html.EscapeString(class) + `">` + inner + capHTML + `</figure>`
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
	class := galleryClassFromAttrs(block.Attrs, columns, cropped)
	var b strings.Builder
	b.WriteString(`<figure`)
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(` class="` + class + `">`)
	sizeSlug := strings.TrimSpace(attrString(block.Attrs, "sizeSlug"))
	if sizeSlug == "" {
		sizeSlug = "large"
	}
	linkTo := strings.ToLower(strings.TrimSpace(attrString(block.Attrs, "linkTo")))
	linkTarget := strings.TrimSpace(attrString(block.Attrs, "linkTarget"))
	for _, id := range ids {
		url := strings.TrimSpace(e.GetAttachmentURL(id))
		if url == "" {
			continue
		}
		imgAttrs := map[string]any{
			"id":       id,
			"url":      url,
			"data-id":  strconv.FormatInt(id, 10),
			"sizeSlug": sizeSlug,
		}
		switch linkTo {
		case "media", "file":
			imgAttrs["href"] = url
			imgAttrs["linkDestination"] = "media"
		case "attachment", "post":
			imgAttrs["href"] = url
			imgAttrs["linkDestination"] = "attachment"
		}
		if linkTarget != "" && imgAttrs["href"] != nil {
			imgAttrs["linkTarget"] = linkTarget
			if linkTarget == "_blank" {
				imgAttrs["rel"] = "noopener"
			}
		}
		imgBlock := Block{
			BlockName: strPtr("core/image"),
			Attrs:     imgAttrs,
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

func galleryClassFromAttrs(attrs map[string]any, columns int, cropped bool) string {
	class := "wp-block-gallery has-nested-images"
	if columns > 0 {
		class += fmt.Sprintf(" columns-%d", columns)
	} else {
		class += " columns-default"
	}
	if cropped {
		class += " is-cropped"
	}
	// Surface align + className on the wrapper (P5.110).
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + cn
	}
	return class
}

// applyGalleryAttrs polishes className/align/anchor on gallery wrappers (P5.110).
func applyGalleryAttrs(htmlStr string, attrs map[string]any) string {
	out := htmlStr
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		out = addRootClass(out, "align"+sanitizeClassToken(align))
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	return out
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
