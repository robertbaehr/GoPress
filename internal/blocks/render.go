package blocks

import (
	"fmt"
	"html"
	"strings"
)

// Render parses serialized block content and returns HTML (do_blocks).
func Render(content string) string {
	return RenderWith(nil, content)
}

// RenderWith is like Render but resolves dynamic blocks via env.
func RenderWith(env *Env, content string) string {
	return env.ensure().Render(content)
}

// Render parses serialized block content using this env.
func (e *Env) Render(content string) string {
	e = e.ensure()
	if content == "" {
		return ""
	}
	parsed := Parse(content)
	var b strings.Builder
	for _, block := range parsed {
		b.WriteString(e.RenderBlock(block))
	}
	return b.String()
}

// RenderBlock renders a single parsed block to HTML (no dynamic env).
func RenderBlock(block Block) string {
	return (&Env{}).RenderBlock(block)
}

// RenderBlock renders a single parsed block to HTML.
func (e *Env) RenderBlock(block Block) string {
	e = e.ensure()
	name := blockName(block)
	switch name {
	case "":
		// Freeform / classic HTML.
		return block.InnerHTML
	case "core/paragraph", "core/heading", "core/list", "core/list-item",
		"core/image", "core/quote", "core/code", "core/html",
		"core/group", "core/columns", "core/column", "core/buttons", "core/button",
		"core/pullquote", "core/preformatted", "core/verse", "core/table",
		"core/file", "core/audio", "core/video", "core/cover", "core/media-text":
		return e.renderSaved(block)
	case "core/separator":
		return e.renderSeparator(block)
	case "core/spacer":
		return e.renderSpacer(block)
	case "core/embed":
		return e.renderEmbed(block)
	case "core/template-part":
		return e.renderTemplatePart(block)
	default:
		// Unknown blocks: prefer saved markup with nested reconstruction.
		if len(block.InnerBlocks) > 0 || len(block.InnerContent) > 0 {
			return e.renderSaved(block)
		}
		return block.InnerHTML
	}
}

func (e *Env) renderTemplatePart(block Block) string {
	slug, _ := block.Attrs["slug"].(string)
	slug = strings.TrimSpace(slug)
	if slug == "" || e.LoadTemplatePart == nil {
		return ""
	}
	if e.seenPart == nil {
		e.seenPart = map[string]bool{}
	}
	if e.seenPart[slug] {
		return ""
	}
	e.seenPart[slug] = true
	defer delete(e.seenPart, slug)

	content, ok := e.LoadTemplatePart(slug)
	if !ok || strings.TrimSpace(content) == "" {
		return ""
	}
	inner := e.Render(content)
	tag := templatePartTag(block)
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(tag)
	b.WriteString(` class="wp-block-template-part">`)
	b.WriteString(inner)
	b.WriteString("</")
	b.WriteString(tag)
	b.WriteString(">")
	return b.String()
}

func templatePartTag(block Block) string {
	if tn, ok := block.Attrs["tagName"].(string); ok {
		tn = strings.TrimSpace(strings.ToLower(tn))
		if isSafeHTMLTag(tn) {
			return tn
		}
	}
	if area, ok := block.Attrs["area"].(string); ok {
		switch strings.ToLower(strings.TrimSpace(area)) {
		case "header":
			return "header"
		case "footer":
			return "footer"
		}
	}
	return "div"
}

func isSafeHTMLTag(tag string) bool {
	switch tag {
	case "div", "header", "footer", "section", "aside", "main", "nav":
		return true
	default:
		return false
	}
}

func (e *Env) renderSeparator(block Block) string {
	if block.InnerHTML != "" {
		return e.renderSaved(block)
	}
	class := "wp-block-separator"
	if cn, ok := block.Attrs["className"].(string); ok && cn != "" {
		class += " " + html.EscapeString(cn)
	}
	return fmt.Sprintf(`<hr class="%s"/>`, class)
}

func (e *Env) renderSpacer(block Block) string {
	if block.InnerHTML != "" {
		return e.renderSaved(block)
	}
	height := "100px"
	if h, ok := block.Attrs["height"].(string); ok && h != "" {
		height = h
	}
	style := fmt.Sprintf("height:%s", html.EscapeString(height))
	if w, ok := block.Attrs["width"].(string); ok && w != "" {
		style += ";width:" + html.EscapeString(w)
	}
	return fmt.Sprintf(`<div style="%s" aria-hidden="true" class="wp-block-spacer"></div>`, style)
}

func (e *Env) renderEmbed(block Block) string {
	if block.InnerHTML != "" {
		return e.renderSaved(block)
	}
	url, _ := block.Attrs["url"].(string)
	if url == "" {
		return ""
	}
	esc := html.EscapeString(url)
	caption, _ := block.Attrs["caption"].(string)
	var b strings.Builder
	b.WriteString(`<figure class="wp-block-embed"><div class="wp-block-embed__wrapper"><a href="`)
	b.WriteString(esc)
	b.WriteString(`">`)
	b.WriteString(esc)
	b.WriteString(`</a></div>`)
	if caption != "" {
		b.WriteString(`<figcaption class="wp-element-caption">`)
		b.WriteString(html.EscapeString(caption))
		b.WriteString(`</figcaption>`)
	}
	b.WriteString(`</figure>`)
	return b.String()
}

// renderSaved reconstructs block HTML from innerContent + nested blocks.
// Leaf blocks simply return InnerHTML.
func (e *Env) renderSaved(block Block) string {
	if len(block.InnerBlocks) == 0 {
		return block.InnerHTML
	}
	if len(block.InnerContent) == 0 {
		var b strings.Builder
		b.WriteString(block.InnerHTML)
		for _, inner := range block.InnerBlocks {
			b.WriteString(e.RenderBlock(inner))
		}
		return b.String()
	}
	var b strings.Builder
	innerIdx := 0
	for _, part := range block.InnerContent {
		if part == nil {
			if innerIdx < len(block.InnerBlocks) {
				b.WriteString(e.RenderBlock(block.InnerBlocks[innerIdx]))
				innerIdx++
			}
			continue
		}
		switch v := part.(type) {
		case string:
			b.WriteString(v)
		default:
			b.WriteString(fmt.Sprint(v))
		}
	}
	return b.String()
}

func blockName(b Block) string {
	if b.BlockName == nil {
		return ""
	}
	return *b.BlockName
}
