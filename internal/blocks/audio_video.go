package blocks

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	reAudioTag  = regexp.MustCompile(`(?i)<audio\b`)
	reVideoTag  = regexp.MustCompile(`(?i)<video\b`)
	reVideoOpen = regexp.MustCompile(`(?is)(<video\b)([^>]*)(>)`)
)

// renderAudio renders core/audio (P5.53).
// Prefers saved markup; synthesizes a figure/audio player from src/id attrs.
func (e *Env) renderAudio(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeAudioBlock(e, block.Attrs)
	}
	if strings.TrimSpace(out) == "" || !reAudioTag.MatchString(out) {
		return ""
	}
	return out
}

// renderVideo renders core/video (P5.53).
// Prefers saved markup; synthesizes from src/id; applies width/height + aspect-ratio
// from block attrs when present (subset of render_block_core_video).
func (e *Env) renderVideo(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeVideoBlock(e, block.Attrs)
	}
	if strings.TrimSpace(out) == "" || !reVideoTag.MatchString(out) {
		return ""
	}
	return applyVideoDimensions(out, block.Attrs)
}

func synthesizeAudioBlock(e *Env, attrs map[string]any) string {
	src := mediaSrcFromAttrs(e, attrs)
	if src == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<figure class="wp-block-audio"><audio`)
	writeBoolAttr(&b, "autoplay", attrBool(attrs, "autoplay"))
	writeBoolAttr(&b, "loop", attrBool(attrs, "loop"))
	if preload := strings.TrimSpace(attrString(attrs, "preload")); preload != "" {
		b.WriteString(` preload="`)
		b.WriteString(html.EscapeString(preload))
		b.WriteString(`"`)
	}
	b.WriteString(` controls src="`)
	b.WriteString(html.EscapeString(src))
	b.WriteString(`"></audio>`)
	if caption := strings.TrimSpace(attrString(attrs, "caption")); caption != "" {
		b.WriteString(`<figcaption class="wp-element-caption">`)
		b.WriteString(caption)
		b.WriteString(`</figcaption>`)
	}
	b.WriteString(`</figure>`)
	return b.String()
}

func synthesizeVideoBlock(e *Env, attrs map[string]any) string {
	src := mediaSrcFromAttrs(e, attrs)
	if src == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<figure class="wp-block-video"><video`)
	writeBoolAttr(&b, "autoplay", attrBool(attrs, "autoplay"))
	writeBoolAttr(&b, "loop", attrBool(attrs, "loop"))
	writeBoolAttr(&b, "muted", attrBool(attrs, "muted"))
	writeBoolAttr(&b, "playsinline", attrBool(attrs, "playsInline"))
	controls := true
	if attrs != nil {
		if _, ok := attrs["controls"]; ok {
			controls = attrBool(attrs, "controls")
		}
	}
	writeBoolAttr(&b, "controls", controls)
	preload := strings.TrimSpace(attrString(attrs, "preload"))
	if preload == "" {
		preload = "metadata"
	}
	b.WriteString(` preload="`)
	b.WriteString(html.EscapeString(preload))
	b.WriteString(`"`)
	if poster := strings.TrimSpace(attrString(attrs, "poster")); poster != "" {
		b.WriteString(` poster="`)
		b.WriteString(html.EscapeString(poster))
		b.WriteString(`"`)
	}
	if w, ok := attrInt(attrs, "width"); ok && w > 0 {
		b.WriteString(fmt.Sprintf(` width="%d"`, w))
	}
	if h, ok := attrInt(attrs, "height"); ok && h > 0 {
		b.WriteString(fmt.Sprintf(` height="%d"`, h))
	}
	b.WriteString(` src="`)
	b.WriteString(html.EscapeString(src))
	b.WriteString(`"></video>`)
	if caption := strings.TrimSpace(attrString(attrs, "caption")); caption != "" {
		b.WriteString(`<figcaption class="wp-element-caption">`)
		b.WriteString(caption)
		b.WriteString(`</figcaption>`)
	}
	b.WriteString(`</figure>`)
	return b.String()
}

func mediaSrcFromAttrs(e *Env, attrs map[string]any) string {
	src := strings.TrimSpace(attrString(attrs, "src"))
	if src != "" {
		return src
	}
	id := attrInt64(attrs, "id")
	if id > 0 && e != nil && e.GetAttachmentURL != nil {
		return strings.TrimSpace(e.GetAttachmentURL(id))
	}
	return ""
}

func writeBoolAttr(b *strings.Builder, name string, on bool) {
	if on {
		b.WriteString(" ")
		b.WriteString(name)
	}
}

// applyVideoDimensions injects width/height and aspect-ratio style onto the first <video>.
func applyVideoDimensions(content string, attrs map[string]any) string {
	w, wok := attrInt(attrs, "width")
	h, hok := attrInt(attrs, "height")
	if !wok || !hok || w <= 0 || h <= 0 {
		return content
	}
	m := reVideoOpen.FindStringSubmatchIndex(content)
	if m == nil {
		return content
	}
	before := content[:m[0]]
	open := content[m[2]:m[3]]
	attrPart := content[m[4]:m[5]]
	close := content[m[6]:m[7]]
	rest := content[m[1]:]

	attrPart = upsertHTMLAttr(attrPart, "width", fmt.Sprintf("%d", w))
	attrPart = upsertHTMLAttr(attrPart, "height", fmt.Sprintf("%d", h))
	aspect := fmt.Sprintf("aspect-ratio: %d / %d;", w, h)
	style := htmlAttrValue(attrPart, "style")
	if !strings.Contains(style, "aspect-ratio") {
		attrPart = upsertHTMLAttr(attrPart, "style", aspect+style)
	}

	var b strings.Builder
	b.WriteString(before)
	b.WriteString(open)
	b.WriteString(attrPart)
	b.WriteString(close)
	b.WriteString(rest)
	return b.String()
}

var reHTMLAttr = regexp.MustCompile(`(?i)\s([a-z_:][\w:.-]*)\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)

func htmlAttrValue(attrPart, key string) string {
	for _, m := range reHTMLAttr.FindAllStringSubmatch(attrPart, -1) {
		if strings.EqualFold(m[1], key) {
			if m[3] != "" {
				return m[3]
			}
			if m[4] != "" {
				return m[4]
			}
			return m[5]
		}
	}
	return ""
}
