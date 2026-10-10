package blocks

import (
	"fmt"
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
	block = e.applyBindings(block)
	name := blockName(block)
	var out string
	switch name {
	case "":
		// Freeform / classic HTML.
		return block.InnerHTML
	case "core/paragraph":
		out = e.renderParagraph(block)
	case "core/quote":
		out = e.renderQuote(block)
	case "core/terms-query":
		// WP layout support defaults to flow for terms-query (P10.8 / feed parity).
		html := e.renderSaved(block)
		if strings.TrimSpace(html) == "" {
			html = `<div class="wp-block-terms-query"></div>`
		}
		html = addRootClass(html, "is-layout-flow")
		out = addRootClass(html, "wp-block-terms-query-is-layout-flow")
	case "core/image":
		out = e.renderImage(block)
	case "core/gallery":
		out = e.renderGallery(block)
	case "core/cover":
		out = e.renderCover(block)
	case "core/media-text":
		out = e.renderMediaText(block)
	case "core/columns":
		out = e.renderColumns(block)
	case "core/column":
		out = e.renderColumn(block)
	case "core/group":
		out = e.renderGroup(block)
	case "core/list":
		out = e.renderList(block)
	case "core/list-item":
		out = e.renderListItem(block)
	case "core/heading":
		out = e.renderHeading(block)
	case "core/table":
		out = e.renderTable(block)
	case "core/html":
		out = e.renderHTML(block)
	case "core/details":
		out = e.renderDetails(block)
	case "core/code":
		out = e.renderCode(block)
	case "core/preformatted":
		out = e.renderPreformatted(block)
	case "core/verse":
		out = e.renderVerse(block)
	case "core/pullquote":
		out = e.renderPullquote(block)
	case "core/buttons":
		out = e.renderButtons(block)
	case "core/button":
		out = e.renderButton(block)
	case "core/audio":
		out = e.renderAudio(block)
	case "core/video":
		out = e.renderVideo(block)
	case "core/file":
		out = e.renderFile(block)
	case "core/freeform":
		// Classic editor block (P5.68).
		return e.renderFreeform(block)
	case "core/missing":
		return e.renderMissing(block)
	case "core/legacy-widget":
		// P5.77: widgets gone — empty/placeholder (no WP_Widget_Factory).
		return e.renderLegacyWidget(block)
	case "core/widget-group":
		// P5.78: widgets gone — empty/placeholder (no sidebars).
		return e.renderWidgetGroup(block)
	case "core/footnotes":
		// Return via out so applySupports can attach color/typography/etc. (P5.51).
		out = e.renderFootnotes(block)
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
	case "core/home-link":
		return e.renderHomeLink(block)
	case "core/loginout":
		return e.renderLoginout(block)
	case "core/page-list":
		// Standalone page-list (outside navigation): wrap as a simple list.
		items := e.renderPageList(block)
		if items == "" {
			return ""
		}
		out = `<ul class="wp-block-page-list">` + items + `</ul>`
	case "core/query":
		out = e.renderQuery(block)
	case "core/query-title":
		out = e.renderQueryTitle(block)
	case "core/term-description":
		out = e.renderTermDescription(block)
	case "core/post-template":
		out = e.renderPostTemplate(block)
	case "core/query-no-results":
		return e.renderQueryNoResults(block)
	case "core/pattern":
		return e.renderPattern(block)
	case "core/block":
		return e.renderSyncedPattern(block)
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
	case "core/site-logo":
		return e.renderSiteLogo(block)
	case "core/query-pagination":
		out = e.renderQueryPagination(block)
	case "core/query-pagination-next":
		return e.renderQueryPaginationNext(block)
	case "core/query-pagination-previous":
		return e.renderQueryPaginationPrevious(block)
	case "core/query-pagination-numbers":
		return e.renderQueryPaginationNumbers(block)
	case "core/post-author":
		out = e.renderPostAuthor(block)
	case "core/post-author-name":
		out = e.renderPostAuthorName(block)
	case "core/post-author-biography":
		out = e.renderPostAuthorBiography(block)
	case "core/avatar":
		out = e.renderAvatar(block)
	case "core/post-terms":
		out = e.renderPostTerms(block)
	case "core/post-navigation-link":
		return e.renderPostNavigationLink(block)
	case "core/comments-title":
		out = e.renderCommentsTitle(block)
	case "core/search":
		out = e.renderSearch(block)
	case "core/calendar":
		out = e.renderCalendar(block)
	case "core/archives":
		out = e.renderArchives(block)
	case "core/latest-posts":
		out = e.renderLatestPosts(block)
	case "core/latest-comments":
		out = e.renderLatestComments(block)
	case "core/tag-cloud":
		out = e.renderTagCloud(block)
	case "core/read-more":
		out = e.renderReadMore(block)
	case "core/more":
		return e.renderMore(block)
	case "core/nextpage":
		return e.renderNextPage(block)
	case "core/social-links":
		out = e.renderSocialLinks(block)
	case "core/social-link":
		return e.renderSocialLink(block)
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

// attrLength coerces string or numeric block attrs to a CSS length (P4.50).
func attrLength(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%dpx", int64(t))
		}
		return fmt.Sprintf("%gpx", t)
	case int:
		return fmt.Sprintf("%dpx", t)
	case int64:
		return fmt.Sprintf("%dpx", t)
	default:
		return ""
	}
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
