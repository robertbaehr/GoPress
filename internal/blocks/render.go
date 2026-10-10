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
		"core/image", "core/quote", "core/code", "core/html":
		return renderSaved(block)
	case "core/separator":
		return renderSeparator(block)
	default:
		// Unknown blocks: emit saved innerHTML as-is.
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
