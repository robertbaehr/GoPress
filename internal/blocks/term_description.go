package blocks

import (
	"html"
	"strings"
)

// renderTermDescription renders core/term-description (P5.76).
// Uses TermContext when set (term archives / bindings); otherwise empty.
func (e *Env) renderTermDescription(block Block) string {
	e = e.ensure()
	desc := ""
	if e.TermContext != nil {
		desc = strings.TrimSpace(e.TermContext.Description)
	}
	if desc == "" {
		return ""
	}

	classes := []string{"wp-block-term-description"}
	if ta, ok := block.Attrs["textAlign"].(string); ok && strings.TrimSpace(ta) != "" {
		classes = append(classes, "has-text-align-"+strings.TrimSpace(ta))
	}
	if align, ok := block.Attrs["align"].(string); ok && strings.TrimSpace(align) != "" {
		classes = append(classes, "align"+strings.TrimSpace(align))
	}
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}

	return `<div class="` + html.EscapeString(strings.Join(classes, " ")) + `">` + desc + `</div>`
}

func hasLinkColorAttr(attrs map[string]any) bool {
	if attrs == nil {
		return false
	}
	style, _ := attrs["style"].(map[string]any)
	if style == nil {
		return false
	}
	elements, _ := style["elements"].(map[string]any)
	if elements == nil {
		return false
	}
	link, _ := elements["link"].(map[string]any)
	if link == nil {
		return false
	}
	color, _ := link["color"].(map[string]any)
	if color == nil {
		return false
	}
	_, ok := color["text"]
	return ok
}
