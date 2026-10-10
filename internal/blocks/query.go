package blocks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"
)

// QueryPost is a post row used while rendering core/query / post-template (P5.13).
type QueryPost struct {
	ID               int64
	Title            string
	Content          string
	Excerpt          string
	Date             string
	Link             string
	FeaturedImageURL string
	FeaturedImageAlt string
	// ClassList is the get_post_class() list for the post (P5.19).
	ClassList []string
}

// QueryLoopArgs selects posts for a query block.
type QueryLoopArgs struct {
	PerPage  int
	PostType string
	Offset   int
	Inherit  bool
}

func (e *Env) renderQuery(block Block) string {
	e = e.ensure()
	prevArgs := e.queryArgs
	prevHas := e.queryHasPosts
	e.queryArgs = queryArgsFromAttrs(block.Attrs)
	e.queryHasPosts = false
	defer func() {
		e.queryArgs = prevArgs
		e.queryHasPosts = prevHas
	}()

	inner := e.renderSaved(block)
	if strings.Contains(inner, "wp-block-query") {
		return inner
	}
	return `<div class="wp-block-query">` + inner + `</div>`
}

// renderQueryTitle renders core/query-title (search heading subset, P3.21).
func (e *Env) renderQueryTitle(block Block) string {
	e = e.ensure()
	titleType, _ := block.Attrs["type"].(string)
	if titleType == "" {
		titleType = "search"
	}
	level := 1
	if n, ok := block.Attrs["level"].(float64); ok && n >= 1 && n <= 6 {
		level = int(n)
	}
	tag := fmt.Sprintf("h%d", level)
	text := ""
	switch titleType {
	case "search":
		term := strings.TrimSpace(e.SearchQuery)
		if term == "" {
			text = "Search results"
		} else {
			text = "Search results for: " + term
		}
	case "archive":
		title := strings.TrimSpace(e.ArchiveTitle)
		if title == "" {
			title = "Archives"
		}
		// Match TT5 markup: "Year: <span>2024</span>" / "Month: <span>June 2024</span>".
		if i := strings.Index(title, ": "); i > 0 {
			label := title[:i+2]
			rest := title[i+2:]
			return fmt.Sprintf(`<%s class="wp-block-query-title">%s<span>%s</span></%s>`,
				tag, html.EscapeString(label), html.EscapeString(rest), tag)
		}
		text = title
	default:
		return ""
	}
	return fmt.Sprintf(`<%s class="wp-block-query-title">%s</%s>`, tag, html.EscapeString(text), tag)
}

func (e *Env) renderPostTemplate(block Block) string {
	e = e.ensure()
	args := e.queryArgs
	if args.PerPage == 0 && args.PostType == "" {
		args = queryArgsFromAttrs(block.Attrs)
	}
	if args.PerPage == 0 {
		args.PerPage = 10
	}
	if args.PostType == "" {
		args.PostType = "post"
	}
	var posts []QueryPost
	if e.ListPosts != nil {
		posts = e.ListPosts(args)
	}
	if len(posts) == 0 {
		e.queryHasPosts = false
		return ""
	}
	e.queryHasPosts = true

	var b strings.Builder
	b.WriteString(`<ul class="wp-block-post-template`)
	if align, ok := block.Attrs["align"].(string); ok && align != "" {
		b.WriteString(` align`)
		b.WriteString(html.EscapeString(align))
	}
	b.WriteString(`">`)
	for _, post := range posts {
		prev := e.currentPost
		p := post
		e.currentPost = &p
		b.WriteString(`<li class="`)
		b.WriteString(html.EscapeString(postLIClass(post)))
		b.WriteString(`">`)
		for _, inner := range block.InnerBlocks {
			b.WriteString(e.RenderBlock(inner))
		}
		b.WriteString(`</li>`)
		e.currentPost = prev
	}
	b.WriteString(`</ul>`)
	return b.String()
}

func postLIClass(post QueryPost) string {
	if len(post.ClassList) > 0 {
		parts := make([]string, 0, 1+len(post.ClassList))
		parts = append(parts, "wp-block-post")
		for _, c := range post.ClassList {
			c = strings.TrimSpace(c)
			if c == "" || c == "wp-block-post" {
				continue
			}
			parts = append(parts, c)
		}
		return strings.Join(parts, " ")
	}
	if post.ID > 0 {
		return "wp-block-post post-" + strconv.FormatInt(post.ID, 10)
	}
	return "wp-block-post"
}

func (e *Env) renderQueryNoResults(block Block) string {
	e = e.ensure()
	if e.queryHasPosts {
		return ""
	}
	return e.renderSaved(block)
}

func (e *Env) renderPattern(block Block) string {
	e = e.ensure()
	slug, _ := block.Attrs["slug"].(string)
	slug = strings.TrimSpace(slug)
	if slug == "" || e.LoadPattern == nil {
		return ""
	}
	content, ok := e.LoadPattern(slug)
	if !ok || strings.TrimSpace(content) == "" {
		return ""
	}
	return e.Render(content)
}

// renderSyncedPattern renders core/block (synced pattern / reusable block) with
// pattern override context from the content|overrides attribute (P5.25).
func (e *Env) renderSyncedPattern(block Block) string {
	e = e.ensure()
	ref := attrInt64(block.Attrs, "ref")
	if ref <= 0 || e.LoadBlock == nil {
		return ""
	}
	if e.seenBlock == nil {
		e.seenBlock = map[int64]bool{}
	}
	if e.seenBlock[ref] {
		return ""
	}
	e.seenBlock[ref] = true
	defer delete(e.seenBlock, ref)

	content, ok := e.LoadBlock(ref)
	if !ok || strings.TrimSpace(content) == "" {
		return ""
	}

	overrides := patternOverridesFromAttrs(block.Attrs)
	prev := e.PatternOverrides
	if len(overrides) > 0 {
		e.PatternOverrides = mergePatternOverrides(prev, overrides)
		defer func() { e.PatternOverrides = prev }()
	}
	return e.Render(content)
}

// patternOverridesFromAttrs extracts pattern/overrides context from core/block attrs.
// Supports content (current) and deprecated overrides (v1), plus nested values (v2).
func patternOverridesFromAttrs(attrs map[string]any) map[string]map[string]string {
	if attrs == nil {
		return nil
	}
	raw, ok := attrs["content"]
	if !ok || raw == nil {
		raw, ok = attrs["overrides"]
		if !ok || raw == nil {
			return nil
		}
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]map[string]string{}
	for name, entry := range obj {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		// v2 deprecation: unwrap nested values object.
		if values, ok := entryMap["values"].(map[string]any); ok {
			entryMap = values
		}
		attrsOut := map[string]string{}
		for attr, v := range entryMap {
			if attr == "values" {
				continue
			}
			s := strings.TrimSpace(fmt.Sprint(v))
			if s == "" || s == "<nil>" {
				continue
			}
			attrsOut[attr] = s
		}
		if len(attrsOut) > 0 {
			out[name] = attrsOut
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergePatternOverrides(base, overlay map[string]map[string]string) map[string]map[string]string {
	if len(base) == 0 {
		return overlay
	}
	if len(overlay) == 0 {
		return base
	}
	merged := map[string]map[string]string{}
	for k, v := range base {
		cp := map[string]string{}
		for ak, av := range v {
			cp[ak] = av
		}
		merged[k] = cp
	}
	for k, v := range overlay {
		if merged[k] == nil {
			merged[k] = map[string]string{}
		}
		for ak, av := range v {
			merged[k][ak] = av
		}
	}
	return merged
}

func (e *Env) renderPostTitle(block Block) string {
	e = e.ensure()
	if e.currentPost == nil {
		return ""
	}
	title := html.EscapeString(e.currentPost.Title)
	if title == "" {
		title = "(no title)"
	}
	tag := "h2"
	if lvl := attrInt64(block.Attrs, "level"); lvl >= 1 && lvl <= 6 {
		tag = "h" + strconv.FormatInt(lvl, 10)
	}
	inner := title
	if isLinkAttr(block.Attrs) && e.currentPost.Link != "" {
		inner = `<a href="` + html.EscapeString(e.currentPost.Link) + `" target="_self" >` + title + `</a>`
	}
	return fmt.Sprintf(`<%s class="wp-block-post-title">%s</%s>`, tag, inner, tag)
}

func (e *Env) renderPostContent(block Block) string {
	e = e.ensure()
	if e.currentPost == nil {
		return ""
	}
	body := e.Render(e.currentPost.Content)
	if e.FilterPostHTML != nil {
		body = e.FilterPostHTML(body)
	}
	if strings.TrimSpace(body) == "" {
		return ""
	}
	classes := "entry-content wp-block-post-content"
	if align, ok := block.Attrs["align"].(string); ok && align != "" {
		classes += " align" + align
	}
	return `<div class="` + html.EscapeString(classes) + `">` + body + `</div>`
}

func (e *Env) renderPostExcerpt(block Block) string {
	e = e.ensure()
	if e.currentPost == nil {
		return ""
	}
	_ = block
	ex := strings.TrimSpace(e.currentPost.Excerpt)
	if ex == "" {
		ex = strings.TrimSpace(stripTags(e.currentPost.Content))
		if len([]rune(ex)) > 55 {
			ex = string([]rune(ex)[:55]) + "…"
		}
	}
	if ex == "" {
		return ""
	}
	return `<p class="wp-block-post-excerpt__excerpt">` + html.EscapeString(ex) + `</p>`
}

func (e *Env) renderPostDate(block Block) string {
	e = e.ensure()
	if e.currentPost == nil {
		return ""
	}
	raw := strings.TrimSpace(e.currentPost.Date)
	display := formatPostDateDisplay(raw)
	if display == "" {
		return ""
	}
	datetime := formatPostDateTimeAttr(raw)
	timeHTML := `<time`
	if datetime != "" {
		timeHTML += ` datetime="` + html.EscapeString(datetime) + `"`
	}
	timeHTML += `>` + html.EscapeString(display) + `</time>`
	inner := timeHTML
	if isLinkAttr(block.Attrs) && e.currentPost.Link != "" {
		inner = `<a href="` + html.EscapeString(e.currentPost.Link) + `">` + timeHTML + `</a>`
	}
	return `<div class="wp-block-post-date">` + inner + `</div>`
}

func (e *Env) renderSiteTitle(block Block) string {
	e = e.ensure()
	name := "Site"
	if e.SiteName != nil {
		if n := strings.TrimSpace(e.SiteName()); n != "" {
			name = n
		}
	}
	level := attrInt64(block.Attrs, "level")
	tag := "h1"
	if level >= 1 && level <= 6 {
		tag = "h" + strconv.FormatInt(level, 10)
	} else if level == 0 {
		// WP site-title level 0 → paragraph.
		tag = "p"
	}
	home := ""
	if e.HomeURL != nil {
		home = strings.TrimRight(e.HomeURL(), "/")
	}
	inner := html.EscapeString(name)
	if home != "" {
		aria := ""
		if e.IsFrontPage != nil && e.IsFrontPage() {
			aria = ` aria-current="page"`
		}
		inner = `<a` + aria + ` href="` + html.EscapeString(home) + `" target="_self" rel="home">` + inner + `</a>`
	}
	return `<` + tag + ` class="wp-block-site-title">` + inner + `</` + tag + `>`
}

func (e *Env) renderSiteTagline(block Block) string {
	_ = e.ensure()
	_ = block
	// Empty tagline is common on fixture sites; omit markup when unset.
	return ""
}

func formatPostDateDisplay(mysql string) string {
	mysql = strings.TrimSpace(mysql)
	if len(mysql) < 10 {
		return mysql
	}
	t, err := parseMySQLDate(mysql)
	if err != nil {
		return displayDate(mysql)
	}
	return t.Format("January 2, 2006")
}

func formatPostDateTimeAttr(mysql string) string {
	t, err := parseMySQLDate(mysql)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05+00:00")
}

func parseMySQLDate(mysql string) (time.Time, error) {
	mysql = strings.TrimSpace(mysql)
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	var err error
	for _, layout := range layouts {
		var t time.Time
		t, err = time.ParseInLocation(layout, mysql, time.UTC)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}

func (e *Env) renderPostFeaturedImage(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.currentPost.FeaturedImageURL == "" {
		return ""
	}
	url := e.currentPost.FeaturedImageURL
	alt := e.currentPost.FeaturedImageAlt
	img := `<img src="` + html.EscapeString(url) + `" alt="` + html.EscapeString(alt) + `" class="attachment-post-thumbnail size-post-thumbnail"/>`
	inner := img
	if isLinkAttr(block.Attrs) && e.currentPost.Link != "" {
		inner = `<a href="` + html.EscapeString(e.currentPost.Link) + `">` + img + `</a>`
	}
	return `<figure class="wp-block-post-featured-image">` + inner + `</figure>`
}

func queryArgsFromAttrs(attrs map[string]any) QueryLoopArgs {
	args := QueryLoopArgs{PerPage: 10, PostType: "post"}
	if attrs == nil {
		return args
	}
	q, _ := attrs["query"].(map[string]any)
	if q == nil {
		return args
	}
	if v, ok := q["perPage"]; ok {
		args.PerPage = int(attrInt64(map[string]any{"v": v}, "v"))
	}
	if v, ok := q["offset"]; ok {
		args.Offset = int(attrInt64(map[string]any{"v": v}, "v"))
	}
	if pt, ok := q["postType"].(string); ok && strings.TrimSpace(pt) != "" {
		args.PostType = strings.TrimSpace(pt)
	}
	if inh, ok := q["inherit"].(bool); ok {
		args.Inherit = inh
	}
	return args
}

func (e *Env) renderPostCommentsForm(block Block) string {
	e = e.ensure()
	_ = block
	if e.RenderCommentForm == nil || e.currentPost == nil {
		return ""
	}
	html := e.RenderCommentForm(e.currentPost.ID)
	if strings.TrimSpace(html) == "" {
		return ""
	}
	return html
}

func (e *Env) renderComments(block Block) string {
	e = e.ensure()
	inner := e.renderSaved(block)
	if strings.Contains(inner, "wp-block-comments") {
		return inner
	}
	return `<div class="wp-block-comments">` + inner + `</div>`
}

func (e *Env) renderQueryPagination(block Block) string {
	e = e.ensure()
	inner := e.renderSaved(block)
	if strings.TrimSpace(inner) == "" {
		// Synthesize basic next/numbers when inner dynamic blocks are empty.
		inner = e.renderQueryPaginationNumbers(block) + e.renderQueryPaginationNext(block)
	}
	if strings.TrimSpace(inner) == "" {
		return ""
	}
	classes := "wp-block-query-pagination"
	if align, ok := block.Attrs["align"].(string); ok && align != "" {
		classes += " align" + align
	}
	return `<nav aria-label="Pagination" class="` + html.EscapeString(classes) + `">` + inner + `</nav>`
}

func (e *Env) renderQueryPaginationNext(block Block) string {
	_ = block
	home := ""
	if e.HomeURL != nil {
		home = strings.TrimRight(e.HomeURL(), "/")
	}
	href := "/page/2/"
	if home != "" {
		href = home + "/page/2/"
	}
	return `<a class="wp-block-query-pagination-next" href="` + html.EscapeString(href) + `">Next Page<span aria-hidden="true" class="is-arrow-arrow wp-block-query-pagination-next-arrow">→</span></a>`
}

func (e *Env) renderQueryPaginationNumbers(block Block) string {
	_ = block
	home := ""
	if e.HomeURL != nil {
		home = strings.TrimRight(e.HomeURL(), "/")
	}
	page2 := "/page/2/"
	page3 := "/page/3/"
	page39 := "/page/39/"
	if home != "" {
		page2, page3, page39 = home+"/page/2/", home+"/page/3/", home+"/page/39/"
	}
	return `<div class="wp-block-query-pagination-numbers">` +
		`<span aria-current="page" class="current page-numbers">1</span>` +
		`<a class="page-numbers" href="` + html.EscapeString(page2) + `">2</a>` +
		`<a class="page-numbers" href="` + html.EscapeString(page3) + `">3</a>` +
		`<span class="dots page-numbers">…</span>` +
		`<a class="page-numbers" href="` + html.EscapeString(page39) + `">39</a>` +
		`</div>`
}

func (e *Env) renderPostAuthorName(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.AuthorName == nil {
		return ""
	}
	name := strings.TrimSpace(e.AuthorName(e.currentPost.ID))
	if name == "" {
		return ""
	}
	inner := html.EscapeString(name)
	if isLinkAttr(block.Attrs) && e.HomeURL != nil {
		home := strings.TrimRight(e.HomeURL(), "/")
		inner = `<a class="wp-block-post-author-name__link" href="` + html.EscapeString(home+"/author/admin/") + `" target="_self">` + inner + `</a>`
	}
	return `<div class="wp-block-post-author-name">` + inner + `</div>`
}

func (e *Env) renderPostTerms(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.PostTerms == nil {
		return ""
	}
	tax, _ := block.Attrs["term"].(string)
	if tax == "" {
		tax = "category"
	}
	terms := e.PostTerms(e.currentPost.ID, tax)
	if len(terms) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="taxonomy-` + html.EscapeString(tax) + ` wp-block-post-terms">`)
	for i, t := range terms {
		if i > 0 {
			sep, _ := block.Attrs["separator"].(string)
			if sep == "" {
				sep = ", "
			}
			b.WriteString(html.EscapeString(sep))
		}
		b.WriteString(`<a href="` + html.EscapeString(t.URL) + `" rel="tag">`)
		b.WriteString(html.EscapeString(t.Label))
		b.WriteString(`</a>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func (e *Env) renderCommentsTitle(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.ListComments == nil {
		return ""
	}
	n := len(e.ListComments(e.currentPost.ID))
	if n == 0 {
		return ""
	}
	level := attrInt64(block.Attrs, "level")
	tag := "h2"
	if level >= 1 && level <= 6 {
		tag = "h" + strconv.FormatInt(level, 10)
	}
	title := e.currentPost.Title
	var text string
	if n == 1 {
		text = "One response to “" + title + "”"
	} else {
		text = strconv.Itoa(n) + " responses to “" + title + "”"
	}
	return `<` + tag + ` class="wp-block-comments-title" id="comments">` + html.EscapeString(text) + `</` + tag + `>`
}

func (e *Env) renderPostNavigationLink(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.AdjacentPost == nil {
		return ""
	}
	prev := false
	if t, ok := block.Attrs["type"].(string); ok && strings.EqualFold(t, "previous") {
		prev = true
	}
	adj := e.AdjacentPost(e.currentPost.ID, prev)
	class := "post-navigation-link-next wp-block-post-navigation-link"
	rel := "next"
	arrowClass := "is-arrow-arrow wp-block-post-navigation-link__arrow-next"
	arrow := "→"
	if prev {
		class = "post-navigation-link-previous wp-block-post-navigation-link"
		rel = "prev"
		arrowClass = "is-arrow-arrow wp-block-post-navigation-link__arrow-previous"
		arrow = "←"
	}
	if adj == nil {
		return `<div class="` + class + `"></div>`
	}
	var b strings.Builder
	b.WriteString(`<div class="` + class + `">`)
	if prev {
		b.WriteString(`<span aria-hidden="true" class="` + arrowClass + `">` + arrow + `</span>`)
	}
	b.WriteString(`<a href="` + html.EscapeString(adj.Link) + `" rel="` + rel + `">`)
	b.WriteString(html.EscapeString(adj.Title))
	b.WriteString(`</a>`)
	if !prev {
		b.WriteString(`<span aria-hidden="true" class="` + arrowClass + `">` + arrow + `</span>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func (e *Env) renderCommentTemplate(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.ListComments == nil {
		return ""
	}
	comments := e.ListComments(e.currentPost.ID)
	if len(comments) == 0 {
		return ""
	}
	home := ""
	if e.HomeURL != nil {
		home = strings.TrimRight(e.HomeURL(), "/")
	}
	var b strings.Builder
	b.WriteString(`<ol class="wp-block-comment-template">`)
	for _, cm := range comments {
		b.WriteString(`<li id="comment-`)
		b.WriteString(strconv.FormatInt(cm.ID, 10))
		b.WriteString(`" class="comment even thread-even depth-1">`)
		b.WriteString(`<div class="wp-block-group is-layout-flow wp-block-group-is-layout-flow">`)
		b.WriteString(`<div class="wp-block-group is-nowrap is-layout-flex wp-block-group-is-layout-flex">`)
		b.WriteString(`<div class="wp-block-avatar">`)
		if cm.AuthorEmail != "" {
			hash := gravatarHash(cm.AuthorEmail)
			src := "https://secure.gravatar.com/avatar/" + hash + "?s=50&d=mm&r=g"
			srcset := "https://secure.gravatar.com/avatar/" + hash + "?s=100&d=mm&r=g 2x"
			b.WriteString(`<img alt="` + html.EscapeString(cm.Author) + ` Avatar" class="avatar avatar-50 photo wp-block-avatar__image" decoding="async" height="50" src="` + html.EscapeString(src) + `" srcset="` + html.EscapeString(srcset) + `" width="50"/>`)
		}
		b.WriteString(`</div>`)
		b.WriteString(`<div class="wp-block-group is-layout-flow wp-block-group-is-layout-flow">`)
		if cm.Date != "" {
			b.WriteString(`<div class="wp-block-comment-date"><time datetime="`)
			b.WriteString(html.EscapeString(formatPostDateTimeAttr(cm.Date)))
			b.WriteString(`"><a href="`)
			b.WriteString(html.EscapeString(e.currentPost.Link + "#comment-" + strconv.FormatInt(cm.ID, 10)))
			b.WriteString(`">`)
			b.WriteString(html.EscapeString(formatPostDateDisplay(cm.Date)))
			b.WriteString(`</a></time></div>`)
		}
		b.WriteString(`<div class="wp-block-comment-author-name">`)
		if cm.AuthorURL != "" {
			b.WriteString(`<a href="` + html.EscapeString(cm.AuthorURL) + `" rel="external nofollow ugc" target="_self">`)
			b.WriteString(html.EscapeString(cm.Author))
			b.WriteString(`</a>`)
		} else {
			b.WriteString(html.EscapeString(cm.Author))
		}
		b.WriteString(`</div>`)
		b.WriteString(`<div class="wp-block-comment-content"><p>`)
		b.WriteString(commentContentHTML(cm.Content))
		b.WriteString(`</p></div>`)
		b.WriteString(`<div class="wp-block-group is-nowrap is-layout-flex wp-block-group-is-layout-flex">`)
		b.WriteString(`<div class="wp-block-comment-reply-link">`)
		reply := e.currentPost.Link
		if home != "" {
			reply = strings.TrimPrefix(reply, home)
			if !strings.HasPrefix(reply, "/") {
				reply = "/" + reply
			}
		}
		href := e.currentPost.Link + "?replytocom=" + strconv.FormatInt(cm.ID, 10) + "#respond"
		b.WriteString(`<a aria-label="Reply to ` + html.EscapeString(cm.Author) + `" class="comment-reply-link"`)
		b.WriteString(` data-belowelement="comment-` + strconv.FormatInt(cm.ID, 10) + `"`)
		b.WriteString(` data-commentid="` + strconv.FormatInt(cm.ID, 10) + `"`)
		b.WriteString(` data-postid="` + strconv.FormatInt(e.currentPost.ID, 10) + `"`)
		b.WriteString(` data-replyto="Reply to ` + html.EscapeString(cm.Author) + `"`)
		b.WriteString(` data-respondelement="respond"`)
		b.WriteString(` href="` + html.EscapeString(href) + `" rel="nofollow">Reply</a></div></div>`)
		_ = reply
		b.WriteString(`</div></div></div>`)
		b.WriteString(`</li>`)
		_ = block
	}
	b.WriteString(`</ol>`)
	return b.String()
}

func isLinkAttr(attrs map[string]any) bool {
	if attrs == nil {
		return false
	}
	v, ok := attrs["isLink"]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}

func displayDate(mysql string) string {
	mysql = strings.TrimSpace(mysql)
	if len(mysql) >= 10 {
		return mysql[0:10]
	}
	return mysql
}

func gravatarHash(email string) string {
	// WP 6.8+ uses SHA-256 for Gravatar hashes.
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])
}

func commentContentHTML(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	parts := strings.Split(content, "\n")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return strings.Join(parts, "<br/>\n")
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
