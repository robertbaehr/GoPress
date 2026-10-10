package blocks

import (
	"fmt"
	"html"
	"strings"
)

// Render parses serialized block content and returns HTML (do_blocks).
func Render(content string) string {
	if content == "" {
		return ""
	}
	blocks := Parse(content)
	var b strings.Builder
	for _, block := range blocks {
		b.WriteString(RenderBlock(block))
	}
	return b.String()
}

// RenderBlock renders a single parsed block to HTML.
func RenderBlock(block Block) string {
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
		return renderSaved(block)
	case "core/separator":
		return renderSeparator(block)
	case "core/spacer":
		return renderSpacer(block)
	case "core/embed":
		return renderEmbed(block)
	default:
		// Unknown blocks: prefer saved markup with nested reconstruction.
		if len(block.InnerBlocks) > 0 || len(block.InnerContent) > 0 {
			return renderSaved(block)
		}
		return block.InnerHTML
	}
}

func renderSeparator(block Block) string {
	if block.InnerHTML != "" {
		return renderSaved(block)
	}
	class := "wp-block-separator"
	if cn, ok := block.Attrs["className"].(string); ok && cn != "" {
		class += " " + html.EscapeString(cn)
	}
	return fmt.Sprintf(`<hr class="%s"/>`, class)
}

func renderSpacer(block Block) string {
	if block.InnerHTML != "" {
		return renderSaved(block)
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

func renderEmbed(block Block) string {
	if block.InnerHTML != "" {
		return renderSaved(block)
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
func renderSaved(block Block) string {
	if len(block.InnerBlocks) == 0 {
		return block.InnerHTML
	}
	if len(block.InnerContent) == 0 {
		var b strings.Builder
		b.WriteString(block.InnerHTML)
		for _, inner := range block.InnerBlocks {
			b.WriteString(RenderBlock(inner))
		}
		return b.String()
	}
	var b strings.Builder
	innerIdx := 0
	for _, part := range block.InnerContent {
		if part == nil {
			if innerIdx < len(block.InnerBlocks) {
				b.WriteString(RenderBlock(block.InnerBlocks[innerIdx]))
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
