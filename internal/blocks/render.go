package blocks

import (
	"encoding/json"
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
	var out string
	switch name {
	case "":
		// Freeform / classic HTML.
		return block.InnerHTML
	case "core/paragraph":
		out = addRootClass(e.renderSaved(block), "wp-block-paragraph")
	case "core/quote":
		html := e.renderSaved(block)
		html = addRootClass(html, "is-layout-flow")
		out = addRootClass(html, "wp-block-quote-is-layout-flow")
	case "core/terms-query":
		// WP layout support defaults to flow for terms-query (P10.8 / feed parity).
		html := e.renderSaved(block)
		if strings.TrimSpace(html) == "" {
			html = `<div class="wp-block-terms-query"></div>`
		}
		html = addRootClass(html, "is-layout-flow")
		out = addRootClass(html, "wp-block-terms-query-is-layout-flow")
	case "core/heading", "core/list", "core/list-item",
		"core/image", "core/code", "core/html",
		"core/group", "core/columns", "core/column", "core/buttons", "core/button",
		"core/pullquote", "core/preformatted", "core/verse", "core/table",
		"core/file", "core/audio", "core/video", "core/cover", "core/media-text",
		"core/details":
		out = e.renderSaved(block)
	case "core/freeform":
		// Classic editor block: prefer attribute content, else inner HTML.
		if content, ok := block.Attrs["content"].(string); ok && content != "" {
			return content
		}
		return block.InnerHTML
	case "core/missing":
		return e.renderMissing(block)
	case "core/footnotes":
		return e.renderFootnotes(block)
	case "core/separator":
		out = e.renderSeparator(block)
	case "core/spacer":
		out = e.renderSpacer(block)
	case "core/embed":
		out = e.renderEmbed(block)
	case "core/template-part":
		return e.renderTemplatePart(block)
	case "core/navigation":
		out = e.renderNavigation(block)
	case "core/navigation-link":
		return e.renderNavigationLink(block)
	case "core/page-list":
		// Standalone page-list (outside navigation): wrap as a simple list.
		items := e.renderPageList(block)
		if items == "" {
			return ""
		}
		out = `<ul class="wp-block-page-list">` + items + `</ul>`
	case "core/query":
		out = e.renderQuery(block)
	case "core/post-template":
		out = e.renderPostTemplate(block)
	case "core/query-no-results":
		return e.renderQueryNoResults(block)
	case "core/pattern":
		return e.renderPattern(block)
	case "core/post-title":
		out = e.renderPostTitle(block)
	case "core/post-content":
		out = e.renderPostContent(block)
	case "core/post-excerpt":
		out = e.renderPostExcerpt(block)
	case "core/post-date":
		out = e.renderPostDate(block)
	case "core/post-featured-image":
		return e.renderPostFeaturedImage(block)
	case "core/post-comments-form":
		return e.renderPostCommentsForm(block)
	case "core/comments":
		out = e.renderComments(block)
	case "core/comment-template":
		return e.renderCommentTemplate(block)
	case "core/site-title":
		out = e.renderSiteTitle(block)
	case "core/site-tagline":
		out = e.renderSiteTagline(block)
	case "core/query-pagination":
		out = e.renderQueryPagination(block)
	case "core/query-pagination-next":
		return e.renderQueryPaginationNext(block)
	case "core/query-pagination-previous":
		return ""
	case "core/query-pagination-numbers":
		return e.renderQueryPaginationNumbers(block)
	case "core/post-author-name":
		out = e.renderPostAuthorName(block)
	case "core/post-terms":
		out = e.renderPostTerms(block)
	default:
		// Unknown blocks: prefer saved markup with nested reconstruction.
		if len(block.InnerBlocks) > 0 || len(block.InnerContent) > 0 {
			out = e.renderSaved(block)
		} else {
			out = block.InnerHTML
		}
	}
	return applySupports(out, block.Attrs, name)
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
	// WP defaults the wrapping tag from the part slug when tagName/area unset (P5.16).
	if slug, ok := block.Attrs["slug"].(string); ok {
		switch strings.ToLower(strings.TrimSpace(slug)) {
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

// renderMissing outputs the originalContent attribute (unsupported block placeholder).
func (e *Env) renderMissing(block Block) string {
	if oc, ok := block.Attrs["originalContent"].(string); ok && oc != "" {
		return oc
	}
	if block.InnerHTML != "" {
		return block.InnerHTML
	}
	return e.renderSaved(block)
}

// renderFootnotes renders core/footnotes from post meta JSON (core/footnote format data).
func (e *Env) renderFootnotes(block Block) string {
	e = e.ensure()
	_ = block
	if e.currentPost == nil || e.LoadFootnotes == nil {
		return ""
	}
	raw := strings.TrimSpace(e.LoadFootnotes(e.currentPost.ID))
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
	var b strings.Builder
	b.WriteString(`<ol class="wp-block-footnotes">`)
	for i, note := range notes {
		id := html.EscapeString(note.ID)
		aria := html.EscapeString(fmt.Sprintf("Jump to footnote reference %d", i+1))
		b.WriteString(`<li id="`)
		b.WriteString(id)
		b.WriteString(`">`)
		b.WriteString(note.Content)
		b.WriteString(` <a href="#`)
		b.WriteString(id)
		b.WriteString(`-link" aria-label="`)
		b.WriteString(aria)
		b.WriteString(`">↩︎</a></li>`)
	}
	b.WriteString(`</ol>`)
	return b.String()
}

// addRootClass injects className onto the first HTML opening tag if absent.
func addRootClass(htmlFragment, className string) string {
	if htmlFragment == "" || className == "" {
		return htmlFragment
	}
	i := strings.Index(htmlFragment, "<")
	if i < 0 {
		return htmlFragment
	}
	rest := htmlFragment[i+1:]
	if rest == "" || rest[0] == '/' || rest[0] == '!' {
		return htmlFragment
	}
	// Find end of opening tag.
	end := strings.Index(rest, ">")
	if end < 0 {
		return htmlFragment
	}
	open := rest[:end]
	selfClose := strings.HasSuffix(open, "/")
	if selfClose {
		open = strings.TrimSpace(strings.TrimSuffix(open, "/"))
	}
	lower := strings.ToLower(open)
	if strings.Contains(lower, `class="`) || strings.Contains(lower, `class='`) {
		// Merge into existing class attribute.
		for _, quote := range []string{`"`, `'`} {
			needle := "class=" + quote
			idx := strings.Index(lower, needle)
			if idx < 0 {
				continue
			}
			start := idx + len(needle)
			closeIdx := strings.Index(open[start:], quote)
			if closeIdx < 0 {
				break
			}
			existing := open[start : start+closeIdx]
			if strings.Contains(" "+existing+" ", " "+className+" ") {
				return htmlFragment
			}
			merged := className
			if existing != "" {
				merged = existing + " " + className
			}
			newOpen := open[:start] + merged + open[start+closeIdx:]
			if selfClose {
				newOpen += " /"
			}
			return htmlFragment[:i+1] + newOpen + htmlFragment[i+1+end:]
		}
		return htmlFragment
	}
	newOpen := open + ` class="` + className + `"`
	if selfClose {
		newOpen += " /"
	}
	return htmlFragment[:i+1] + newOpen + htmlFragment[i+1+end:]
}
