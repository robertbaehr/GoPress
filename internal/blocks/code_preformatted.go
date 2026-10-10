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

// renderCode renders core/code (P5.62).
// Prefers saved markup; synthesizes <pre><code> from content; honors align/className.
func (e *Env) renderCode(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reCodeTag.MatchString(out) {
		return out
	}
	return synthesizeCodeBlock(block.Attrs)
}

func synthesizeCodeBlock(attrs map[string]any) string {
	content := attrString(attrs, "content")
	class := "wp-block-code"
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	return `<pre class="` + class + `"><code>` + content + `</code></pre>`
}

// renderPreformatted renders core/preformatted (P5.62).
// Prefers saved markup; synthesizes <pre> from content; honors className.
func (e *Env) renderPreformatted(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && rePreformattedTag.MatchString(out) {
		return out
	}
	return synthesizePreformattedBlock(block.Attrs)
}

func synthesizePreformattedBlock(attrs map[string]any) string {
	content := attrString(attrs, "content")
	class := "wp-block-preformatted"
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	return `<pre class="` + class + `">` + content + `</pre>`
}

// renderVerse renders core/verse (P5.62).
// Prefers saved markup; synthesizes <pre> from content; honors textAlign/className.
func (e *Env) renderVerse(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reVerseTag.MatchString(out) {
		return out
	}
	return synthesizeVerseBlock(block.Attrs)
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
	return `<pre class="` + class + `">` + content + `</pre>`
}
