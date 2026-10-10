package blocks

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

// QueryPost is a post row used while rendering core/query / post-template (P5.13).
type QueryPost struct {
	ID      int64
	Title   string
	Content string
	Excerpt string
	Date    string
	Link    string
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
	b.WriteString(`<ul class="wp-block-post-template">`)
	for _, post := range posts {
		prev := e.currentPost
		p := post
		e.currentPost = &p
		b.WriteString(`<li class="wp-block-post">`)
		for _, inner := range block.InnerBlocks {
			b.WriteString(e.RenderBlock(inner))
		}
		b.WriteString(`</li>`)
		e.currentPost = prev
	}
	b.WriteString(`</ul>`)
	return b.String()
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
		inner = `<a href="` + html.EscapeString(e.currentPost.Link) + `" rel="bookmark">` + title + `</a>`
	}
	return fmt.Sprintf(`<%s class="wp-block-post-title">%s</%s>`, tag, inner, tag)
}

func (e *Env) renderPostContent(block Block) string {
	e = e.ensure()
	if e.currentPost == nil {
		return ""
	}
	_ = block
	body := e.Render(e.currentPost.Content)
	if strings.TrimSpace(body) == "" {
		return ""
	}
	return `<div class="wp-block-post-content">` + body + `</div>`
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
	date := displayDate(e.currentPost.Date)
	if date == "" {
		return ""
	}
	esc := html.EscapeString(date)
	inner := esc
	if isLinkAttr(block.Attrs) && e.currentPost.Link != "" {
		inner = `<a href="` + html.EscapeString(e.currentPost.Link) + `">` + esc + `</a>`
	}
	return `<div class="wp-block-post-date"><time>` + inner + `</time></div>`
}

func (e *Env) renderPostFeaturedImage(block Block) string {
	_ = e
	_ = block
	// Featured image resolution lands with media meta; stub empty for now.
	return ""
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
