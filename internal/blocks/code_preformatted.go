package blocks

import (
	"html"
	"regexp"
	"strings"
)

var (
	reCodeTag         = regexp.MustCompile(`(?i)wp-block-code\b`)
	rePreformattedTag = regexp.MustCompile(`(?i)wp-block-preformatted\b`)
	reVerseTag        = regexp.MustCompile(`(?i)wp-block-verse\b`)
)

// renderCode renders core/code (P5.62 / P5.102 / P5.143).
// Prefers saved markup; synthesizes <pre><code> from content; honors align/className/anchor.
// P5.143: align enum coerce (wide only), gradient preset + fontSize polish.
// Color/typography/spacing land via applySupports on the caller (sanitized attrs).
func (e *Env) renderCode(block Block) string {
	e = e.ensure()
	attrs := sanitizeCodeBlockAttrs(block.Attrs)
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reCodeTag.MatchString(out) {
		return polishCodeMarkup(out, attrs)
	}
	return synthesizeCodeBlock(attrs)
}

// sanitizeCodeBlockAttrs coerces core/code supports.align to the block.json enum (P5.143).
// Only "wide" is allowed; other align values are dropped before synthesize/polish/supports.
func sanitizeCodeBlockAttrs(attrs map[string]any) map[string]any {
	if attrs == nil {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	align := strings.TrimSpace(attrString(out, "align"))
	if align != "" && align != "wide" {
		delete(out, "align")
	}
	return out
}

// polishCodeMarkup ensures wp-block-code + className/anchor/gradient/fontSize on saved HTML (P5.102 / P5.143).
func polishCodeMarkup(out string, attrs map[string]any) string {
	out = addRootClass(out, "wp-block-code")
	if align := strings.TrimSpace(attrString(attrs, "align")); align == "wide" {
		out = addRootClass(out, "alignwide")
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	// P5.143: typography.fontSize preset class (also via applySupports).
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		out = addRootClass(out, "has-"+sanitizeClassToken(fs)+"-font-size")
	}
	// P5.143: color.gradients preset → has-{slug}-gradient-background.
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		out = addRootClass(out, "has-background")
		out = addRootClass(out, "has-"+sanitizeClassToken(g)+"-gradient-background")
	}
	return out
}

func synthesizeCodeBlock(attrs map[string]any) string {
	content := attrString(attrs, "content")
	class := "wp-block-code"
	if align := strings.TrimSpace(attrString(attrs, "align")); align == "wide" {
		class += " alignwide"
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		class += " has-" + sanitizeClassToken(fs) + "-font-size"
	}
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		class += " has-background has-" + sanitizeClassToken(g) + "-gradient-background"
	}
	var b strings.Builder
	b.WriteString(`<pre class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(`><code>`)
	b.WriteString(content)
	b.WriteString(`</code></pre>`)
	return b.String()
}

// renderPreformatted renders core/preformatted (P5.62 / P5.101 / P5.144).
// Prefers saved markup; synthesizes <pre> from content; honors className/anchor.
// P5.144: drop unsupported align; gradient preset + fontSize polish.
// Color/typography/spacing land via applySupports on the caller (sanitized attrs).
func (e *Env) renderPreformatted(block Block) string {
	e = e.ensure()
	attrs := sanitizePreformattedBlockAttrs(block.Attrs)
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && rePreformattedTag.MatchString(out) {
		return polishPreformattedMarkup(out, attrs)
	}
	return synthesizePreformattedBlock(attrs)
}

// sanitizePreformattedBlockAttrs drops supports.align (not in block.json) before
// synthesize/polish/supports (P5.144).
func sanitizePreformattedBlockAttrs(attrs map[string]any) map[string]any {
	if attrs == nil {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	if strings.TrimSpace(attrString(out, "align")) != "" {
		delete(out, "align")
	}
	return out
}

// polishPreformattedMarkup ensures wp-block-preformatted + className/anchor/gradient/fontSize
// on saved HTML (P5.101 / P5.144).
func polishPreformattedMarkup(out string, attrs map[string]any) string {
	out = addRootClass(out, "wp-block-preformatted")
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	// P5.144: typography.fontSize preset class (also via applySupports).
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		out = addRootClass(out, "has-"+sanitizeClassToken(fs)+"-font-size")
	}
	// P5.144: color.gradients preset → has-{slug}-gradient-background.
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		out = addRootClass(out, "has-background")
		out = addRootClass(out, "has-"+sanitizeClassToken(g)+"-gradient-background")
	}
	return out
}

func synthesizePreformattedBlock(attrs map[string]any) string {
	content := attrString(attrs, "content")
	class := "wp-block-preformatted"
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		class += " has-" + sanitizeClassToken(fs) + "-font-size"
	}
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		class += " has-background has-" + sanitizeClassToken(g) + "-gradient-background"
	}
	var b strings.Builder
	b.WriteString(`<pre class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(content)
	b.WriteString(`</pre>`)
	return b.String()
}

// renderVerse renders core/verse (P5.62 / P5.100).
// Prefers saved markup; synthesizes <pre> from content; honors textAlign/className/anchor.
// Color/typography/spacing land via applySupports on the caller.
func (e *Env) renderVerse(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reVerseTag.MatchString(out) {
		return polishVerseMarkup(out, block.Attrs)
	}
	return synthesizeVerseBlock(block.Attrs)
}

// polishVerseMarkup ensures wp-block-verse + textAlign/className/anchor on saved HTML (P5.100).
func polishVerseMarkup(out string, attrs map[string]any) string {
	out = addRootClass(out, "wp-block-verse")
	if ta := strings.TrimSpace(attrString(attrs, "textAlign")); ta != "" {
		out = addRootClass(out, "has-text-align-"+sanitizeClassToken(ta))
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

func synthesizeVerseBlock(attrs map[string]any) string {
	content := attrString(attrs, "content")
	class := "wp-block-verse"
	if ta := strings.TrimSpace(attrString(attrs, "textAlign")); ta != "" {
		class += " has-text-align-" + sanitizeClassToken(ta)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	var b strings.Builder
	b.WriteString(`<pre class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(content)
	b.WriteString(`</pre>`)
	return b.String()
}
