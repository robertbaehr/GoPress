package blocks

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// renderFootnotes renders core/footnotes from post meta JSON (P5.15 / P5.51 / P5.98).
// Mirrors render_block_core_footnotes: empty when no post, password-protected,
// or missing meta; wrapper attributes come from applySupports on the caller.
func (e *Env) renderFootnotes(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.LoadFootnotes == nil {
		return ""
	}
	postID := e.currentPost.ID
	if e.PasswordRequired != nil && e.PasswordRequired(postID) {
		return ""
	}
	if e.PasswordRequired == nil && strings.TrimSpace(e.currentPost.Password) != "" {
		return ""
	}
	raw := strings.TrimSpace(e.LoadFootnotes(postID))
	if raw == "" {
		return ""
	}
	var notes []struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(raw), &notes); err != nil || len(notes) == 0 {
		return ""
	}

	// Base class only; className / color / typography land via applySupports (P5.51).
	var b strings.Builder
	b.WriteString(`<ol class="wp-block-footnotes"`)
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	written := 0
	for _, note := range notes {
		id := strings.TrimSpace(note.ID)
		if id == "" {
			// P5.98: skip entries without an id (empty id breaks return links).
			continue
		}
		written++
		escID := html.EscapeString(id)
		aria := html.EscapeString(fmt.Sprintf("Jump to footnote reference %d", written))
		content := note.Content
		// P5.98: sanitize like wp_kses_post when Env provides SanitizeHTML.
		if e.SanitizeHTML != nil {
			content = e.SanitizeHTML(content)
		}
		b.WriteString(`<li id="`)
		b.WriteString(escID)
		b.WriteString(`">`)
		b.WriteString(content)
		b.WriteString(` <a href="#`)
		b.WriteString(escID)
		b.WriteString(`-link" aria-label="`)
		b.WriteString(aria)
		b.WriteString(`">↩︎</a></li>`)
	}
	if written == 0 {
		return ""
	}
	b.WriteString(`</ol>`)
	return b.String()
}
