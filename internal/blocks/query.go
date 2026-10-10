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
	Modified         string // ISO / display modified date for core/post-data (P5.39).
	Link             string
	FeaturedImageURL string
	FeaturedImageAlt string
	// Password is post_password; non-empty hides footnotes when PasswordRequired is unset (P5.51).
	Password string
	// ClassList is the get_post_class() list for the post (P5.19).
	ClassList []string
}

// TermBindingContext holds term fields for core/term-data bindings (P5.39).
type TermBindingContext struct {
	ID          int64
	Taxonomy    string
	Name        string
	Slug        string
	Link        string
	Description string
	Parent      int64
	Count       int64
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

// renderQueryTitle renders core/query-title (search/archive/post-type, P3.21 / P5.71).
func (e *Env) renderQueryTitle(block Block) string {
	e = e.ensure()
	titleType, _ := block.Attrs["type"].(string)
	if titleType == "" {
		titleType = "search"
	}
	// level 0 → <p>; 1–6 → <hN> (block.json / render_block_core_query_title, P5.71).
	tag := "h1"
	if block.Attrs != nil {
		if _, ok := block.Attrs["level"]; ok {
			level := attrInt64(block.Attrs, "level")
			switch {
			case level == 0:
				tag = "p"
			case level >= 1 && level <= 6:
				tag = "h" + strconv.FormatInt(level, 10)
			}
		}
	}
	showPrefix := true
	if block.Attrs != nil {
		if _, ok := block.Attrs["showPrefix"]; ok {
			showPrefix = attrBool(block.Attrs, "showPrefix")
		}
	}
	showSearchTerm := true
	if block.Attrs != nil {
		if _, ok := block.Attrs["showSearchTerm"]; ok {
			showSearchTerm = attrBool(block.Attrs, "showSearchTerm")
		}
	}

	classes := "wp-block-query-title"
	if ta, ok := block.Attrs["textAlign"].(string); ok && strings.TrimSpace(ta) != "" {
		classes += " has-text-align-" + strings.TrimSpace(ta)
	}

	text := ""
	switch titleType {
	case "search":
		term := strings.TrimSpace(e.SearchQuery)
		if term == "" || !showSearchTerm {
			text = "Search results"
		} else {
			// Match core/query-title + TT5: curly quotes around the term (P3.36).
			text = "Search results for: \u201C" + term + "\u201D"
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
			if !showPrefix {
				return fmt.Sprintf(`<%s class="%s">%s</%s>`,
					tag, html.EscapeString(classes), html.EscapeString(rest), tag)
			}
			return fmt.Sprintf(`<%s class="%s">%s<span>%s</span></%s>`,
				tag, html.EscapeString(classes), html.EscapeString(label), html.EscapeString(rest), tag)
		}
		text = title
	case "post-type":
		// Subset: use ArchiveTitle when set (singular post type label), else "Post".
		name := strings.TrimSpace(e.ArchiveTitle)
		if name == "" {
			name = "Post"
		}
		if showPrefix {
			text = "Post Type: \u201C" + name + "\u201D"
		} else {
			text = name
		}
	default:
		return ""
	}
	return fmt.Sprintf(`<%s class="%s">%s</%s>`, tag, html.EscapeString(classes), html.EscapeString(text), tag)
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
	name := ""
	if e.SiteName != nil {
		name = strings.TrimSpace(e.SiteName())
	}
	if name == "" {
		return ""
	}
	tag := "h1" // block.json default level=1
	if block.Attrs != nil {
		if _, ok := block.Attrs["level"]; ok {
			level := attrInt64(block.Attrs, "level")
			if level >= 1 && level <= 6 {
				tag = "h" + strconv.FormatInt(level, 10)
			} else if level == 0 {
				// WP site-title level 0 → paragraph.
				tag = "p"
			}
		}
	}
	classes := "wp-block-site-title"
	if align, ok := block.Attrs["textAlign"].(string); ok && strings.TrimSpace(align) != "" {
		classes += " has-text-align-" + strings.TrimSpace(align)
	}
	inner := html.EscapeString(name)
	// isLink defaults to true (block.json).
	if isLinkAttrDefaultTrue(block.Attrs) && e.HomeURL != nil {
		home := strings.TrimRight(e.HomeURL(), "/")
		if home != "" {
			target := "_self"
			if t, ok := block.Attrs["linkTarget"].(string); ok && strings.TrimSpace(t) != "" {
				target = strings.TrimSpace(t)
			}
			aria := ""
			if e.IsFrontPage != nil && e.IsFrontPage() {
				aria = ` aria-current="page"`
			}
			inner = `<a` + aria + ` href="` + html.EscapeString(home) + `" target="` + html.EscapeString(target) + `" rel="home">` + inner + `</a>`
		}
	}
	return `<` + tag + ` class="` + classes + `">` + inner + `</` + tag + `>`
}

func (e *Env) renderSiteTagline(block Block) string {
	e = e.ensure()
	tagline := ""
	if e.SiteTagline != nil {
		tagline = strings.TrimSpace(e.SiteTagline())
	}
	if tagline == "" {
		// Empty tagline is common on fixture sites; omit markup when unset.
		return ""
	}
	level := attrInt64(block.Attrs, "level")
	tag := "p"
	if level >= 1 && level <= 6 {
		tag = "h" + strconv.FormatInt(level, 10)
	}
	classes := "wp-block-site-tagline"
	if align, ok := block.Attrs["textAlign"].(string); ok && strings.TrimSpace(align) != "" {
		classes += " has-text-align-" + strings.TrimSpace(align)
	}
	return `<` + tag + ` class="` + classes + `">` + html.EscapeString(tagline) + `</` + tag + `>`
}

// renderSiteLogo renders core/site-logo from the site_logo / custom logo URL (P5.45).
func (e *Env) renderSiteLogo(block Block) string {
	e = e.ensure()
	if e.SiteLogoURL == nil {
		return ""
	}
	src := strings.TrimSpace(e.SiteLogoURL())
	if src == "" {
		return ""
	}
	width := attrInt64(block.Attrs, "width")
	var imgB strings.Builder
	imgB.WriteString(`<img`)
	if width > 0 {
		imgB.WriteString(` width="`)
		imgB.WriteString(strconv.FormatInt(width, 10))
		imgB.WriteString(`"`)
	}
	imgB.WriteString(` src="`)
	imgB.WriteString(html.EscapeString(src))
	imgB.WriteString(`" class="custom-logo" alt="" decoding="async"`)
	if width > 0 {
		imgB.WriteString(` style="height:auto;max-width:100%"`)
	}
	imgB.WriteString(` />`)
	img := imgB.String()

	inner := img
	if isLinkAttrDefaultTrue(block.Attrs) && e.HomeURL != nil {
		home := strings.TrimRight(e.HomeURL(), "/")
		if home != "" {
			target := "_self"
			if t, ok := block.Attrs["linkTarget"].(string); ok && strings.TrimSpace(t) != "" {
				target = strings.TrimSpace(t)
			}
			aria := ""
			if target == "_blank" {
				aria = ` aria-label="(Home link, opens in a new tab)"`
			}
			inner = `<a href="` + html.EscapeString(home) + `" class="custom-logo-link" rel="home"` + aria + ` target="` + html.EscapeString(target) + `">` + img + `</a>`
		}
	}
	classes := "wp-block-site-logo"
	if width <= 0 {
		classes += " is-default-size"
	}
	return `<div class="` + classes + `">` + inner + `</div>`
}

// renderReadMore renders core/read-more as a permalink link (P5.47).
func (e *Env) renderReadMore(block Block) string {
	e = e.ensure()
	if e.currentPost == nil {
		return ""
	}
	link := strings.TrimSpace(e.currentPost.Link)
	if link == "" {
		return ""
	}
	title := strings.TrimSpace(e.currentPost.Title)
	if title == "" {
		title = fmt.Sprintf("untitled post %d", e.currentPost.ID)
	}
	moreText := "Read more"
	if content, ok := block.Attrs["content"].(string); ok && strings.TrimSpace(content) != "" {
		moreText = strings.TrimSpace(content)
	}
	target := "_self"
	if t, ok := block.Attrs["linkTarget"].(string); ok && strings.TrimSpace(t) != "" {
		target = strings.TrimSpace(t)
	}
	classes := "wp-block-read-more"
	if justify, ok := block.Attrs["justifyContent"].(string); ok && strings.TrimSpace(justify) != "" {
		classes += " is-justified-" + strings.TrimSpace(justify)
	}
	screenReader := ": " + title
	return `<a class="` + html.EscapeString(classes) + `" href="` + html.EscapeString(link) +
		`" target="` + html.EscapeString(target) + `">` + html.EscapeString(moreText) +
		`<span class="screen-reader-text">` + html.EscapeString(screenReader) + `</span></a>`
}

// renderMore emits the classic <!--more--> marker from core/more (P5.47).
func (e *Env) renderMore(block Block) string {
	_ = e
	custom := ""
	if t, ok := block.Attrs["customText"].(string); ok {
		custom = strings.TrimSpace(t)
	}
	out := "<!--more"
	if custom != "" {
		out += " " + custom
	}
	out += "-->"
	noTeaser := false
	if v, ok := block.Attrs["noTeaser"]; ok {
		switch t := v.(type) {
		case bool:
			noTeaser = t
		case float64:
			noTeaser = t != 0
		case string:
			noTeaser = t == "1" || strings.EqualFold(t, "true")
		}
	}
	if noTeaser {
		out += "\n<!--noteaser-->"
	}
	return out
}

// renderNextPage emits the <!--nextpage--> marker from core/nextpage (P5.47).
func (e *Env) renderNextPage(block Block) string {
	_ = e
	_ = block
	return "<!--nextpage-->"
}

// isLinkAttrDefaultTrue mirrors block.json isLink default:true when the attr is omitted.
func isLinkAttrDefaultTrue(attrs map[string]any) bool {
	if attrs == nil {
		return true
	}
	v, ok := attrs["isLink"]
	if !ok {
		return true
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
		// Synthesize previous/numbers/next when inner dynamic blocks are empty (P5.44).
		inner = e.renderQueryPaginationPrevious(block) +
			e.renderQueryPaginationNumbers(block) +
			e.renderQueryPaginationNext(block)
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

func (e *Env) queryPaginationPage() int {
	e = e.ensure()
	if e.Paged > 1 {
		return e.Paged
	}
	return 1
}

func (e *Env) queryPaginationMaxPages() int {
	e = e.ensure()
	if e.MaxPages > 0 {
		return e.MaxPages
	}
	return 1
}

func (e *Env) queryPaginationHome() string {
	e = e.ensure()
	if e.HomeURL == nil {
		return ""
	}
	return strings.TrimRight(e.HomeURL(), "/")
}

// queryPaginationPageURL builds / or /page/N/ under home (P5.44).
func queryPaginationPageURL(home string, page int) string {
	if page <= 1 {
		if home == "" {
			return "/"
		}
		return home + "/"
	}
	if home == "" {
		return "/page/" + strconv.Itoa(page) + "/"
	}
	return home + "/page/" + strconv.Itoa(page) + "/"
}

func (e *Env) renderQueryPaginationPrevious(block Block) string {
	e = e.ensure()
	page := e.queryPaginationPage()
	if page <= 1 {
		return ""
	}
	label := "Previous Page"
	if l, ok := block.Attrs["label"].(string); ok && strings.TrimSpace(l) != "" {
		label = strings.TrimSpace(l)
	}
	href := queryPaginationPageURL(e.queryPaginationHome(), page-1)
	return `<a href="` + html.EscapeString(href) + `" class="wp-block-query-pagination-previous">` +
		`<span class="wp-block-query-pagination-previous-arrow is-arrow-arrow" aria-hidden="true">←</span>` +
		html.EscapeString(label) + `</a>`
}

func (e *Env) renderQueryPaginationNext(block Block) string {
	e = e.ensure()
	page := e.queryPaginationPage()
	max := e.queryPaginationMaxPages()
	if page >= max {
		return ""
	}
	label := "Next Page"
	if l, ok := block.Attrs["label"].(string); ok && strings.TrimSpace(l) != "" {
		label = strings.TrimSpace(l)
	}
	href := queryPaginationPageURL(e.queryPaginationHome(), page+1)
	return `<a href="` + html.EscapeString(href) + `" class="wp-block-query-pagination-next">` +
		html.EscapeString(label) +
		`<span class="wp-block-query-pagination-next-arrow is-arrow-arrow" aria-hidden="true">→</span></a>`
}

func (e *Env) renderQueryPaginationNumbers(block Block) string {
	e = e.ensure()
	_ = block
	page := e.queryPaginationPage()
	max := e.queryPaginationMaxPages()
	if max <= 1 {
		return ""
	}
	home := e.queryPaginationHome()
	midSize := 2
	if n, ok := block.Attrs["midSize"].(float64); ok && n >= 0 {
		midSize = int(n)
	}

	var b strings.Builder
	b.WriteString(`<div class="wp-block-query-pagination-numbers">`)
	writePage := func(n int, current bool) {
		if current {
			b.WriteString(`<span aria-current="page" class="page-numbers current">`)
			b.WriteString(strconv.Itoa(n))
			b.WriteString(`</span>`)
			return
		}
		b.WriteString(`<a class="page-numbers" href="`)
		b.WriteString(html.EscapeString(queryPaginationPageURL(home, n)))
		b.WriteString(`">`)
		b.WriteString(strconv.Itoa(n))
		b.WriteString(`</a>`)
	}

	// paginate_links-shaped: always 1 + mid window around current + last (P5.44).
	shown := map[int]bool{}
	writePage(1, page == 1)
	shown[1] = true

	start := page - midSize
	if start < 2 {
		start = 2
	}
	end := page + midSize
	if end > max-1 {
		end = max - 1
	}
	if start > 2 {
		b.WriteString(`<span class="page-numbers dots">&hellip;</span>`)
	}
	for n := start; n <= end; n++ {
		if shown[n] {
			continue
		}
		writePage(n, n == page)
		shown[n] = true
	}
	if end < max-1 {
		b.WriteString(`<span class="page-numbers dots">&hellip;</span>`)
	}
	if max > 1 && !shown[max] {
		writePage(max, page == max)
	}
	b.WriteString(`</div>`)
	return b.String()
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
	if isLinkAttr(block.Attrs) {
		if link := e.authorArchiveLink(e.currentPost.ID); link != "" {
			target := "_self"
			if t, ok := block.Attrs["linkTarget"].(string); ok && strings.TrimSpace(t) != "" {
				target = strings.TrimSpace(t)
			}
			inner = `<a class="wp-block-post-author-name__link" href="` + html.EscapeString(link) + `" target="` + html.EscapeString(target) + `">` + inner + `</a>`
		}
	}
	classes := "wp-block-post-author-name"
	if align, ok := block.Attrs["textAlign"].(string); ok && strings.TrimSpace(align) != "" {
		classes += " has-text-align-" + strings.TrimSpace(align)
	}
	return `<div class="` + html.EscapeString(classes) + `">` + inner + `</div>`
}

// renderPostAuthorBiography renders core/post-author-biography (P5.46).
func (e *Env) renderPostAuthorBiography(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.AuthorBio == nil {
		return ""
	}
	bio := strings.TrimSpace(e.AuthorBio(e.currentPost.ID))
	if bio == "" {
		return ""
	}
	classes := "wp-block-post-author-biography"
	if align, ok := block.Attrs["textAlign"].(string); ok && strings.TrimSpace(align) != "" {
		classes += " has-text-align-" + strings.TrimSpace(align)
	}
	return `<div class="` + html.EscapeString(classes) + `">` + html.EscapeString(bio) + `</div>`
}

// renderAvatar renders core/avatar for the current post author (P5.46).
func (e *Env) renderAvatar(block Block) string {
	e = e.ensure()
	size := int64(96)
	if s := attrInt64(block.Attrs, "size"); s > 0 {
		size = s
	}
	name := ""
	email := ""
	link := ""
	if e.currentPost != nil {
		if e.AuthorName != nil {
			name = strings.TrimSpace(e.AuthorName(e.currentPost.ID))
		}
		if e.AuthorEmail != nil {
			email = strings.TrimSpace(e.AuthorEmail(e.currentPost.ID))
		}
		link = e.authorArchiveLink(e.currentPost.ID)
	}
	if email == "" {
		return ""
	}
	if name == "" {
		name = "User"
	}
	hash := gravatarHash(email)
	src := fmt.Sprintf("https://secure.gravatar.com/avatar/%s?s=%d&d=mm&r=g", hash, size)
	srcset := fmt.Sprintf("https://secure.gravatar.com/avatar/%s?s=%d&d=mm&r=g 2x", hash, size*2)
	alt := html.EscapeString(name + " Avatar")
	img := fmt.Sprintf(
		`<img alt="%s" class="avatar avatar-%d photo wp-block-avatar__image" decoding="async" height="%d" src="%s" srcset="%s" width="%d"/>`,
		alt, size, size, html.EscapeString(src), html.EscapeString(srcset), size,
	)
	inner := img
	if isLinkAttr(block.Attrs) && link != "" {
		target := "_self"
		if t, ok := block.Attrs["linkTarget"].(string); ok && strings.TrimSpace(t) != "" {
			target = strings.TrimSpace(t)
		}
		aria := ""
		if target == "_blank" {
			aria = ` aria-label="(` + html.EscapeString(name) + ` author archive, opens in a new tab)"`
		}
		inner = `<a href="` + html.EscapeString(link) + `" target="` + html.EscapeString(target) + `"` + aria + ` class="wp-block-avatar__link">` + img + `</a>`
	}
	return `<div class="wp-block-avatar">` + inner + `</div>`
}

func (e *Env) authorArchiveLink(postID int64) string {
	if e.AuthorLink != nil {
		if link := strings.TrimSpace(e.AuthorLink(postID)); link != "" {
			return link
		}
	}
	if e.HomeURL == nil {
		return ""
	}
	home := strings.TrimRight(e.HomeURL(), "/")
	if home == "" {
		return ""
	}
	return home + "/author/admin/"
}

// renderPostAuthor renders core/post-author (P5.43).
func (e *Env) renderPostAuthor(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.AuthorName == nil {
		return ""
	}
	name := strings.TrimSpace(e.AuthorName(e.currentPost.ID))
	if name == "" {
		return ""
	}
	nameHTML := html.EscapeString(name)
	if isLinkAttr(block.Attrs) {
		if link := e.authorArchiveLink(e.currentPost.ID); link != "" {
			target := "_self"
			if t, ok := block.Attrs["linkTarget"].(string); ok && strings.TrimSpace(t) != "" {
				target = strings.TrimSpace(t)
			}
			nameHTML = `<a href="` + html.EscapeString(link) + `" target="` + html.EscapeString(target) + `">` + nameHTML + `</a>`
		}
	}
	var content strings.Builder
	content.WriteString(`<div class="wp-block-post-author__content">`)
	if byline, ok := block.Attrs["byline"].(string); ok && strings.TrimSpace(byline) != "" {
		content.WriteString(`<p class="wp-block-post-author__byline">`)
		content.WriteString(html.EscapeString(byline))
		content.WriteString(`</p>`)
	}
	content.WriteString(`<p class="wp-block-post-author__name">`)
	content.WriteString(nameHTML)
	content.WriteString(`</p></div>`)
	classes := "wp-block-post-author"
	if align, ok := block.Attrs["textAlign"].(string); ok && align != "" {
		classes += " has-text-align-" + align
	}
	return `<div class="` + html.EscapeString(classes) + `">` + content.String() + `</div>`
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

// renderPostNavigationLink renders core/post-navigation-link (P5.3 / P5.72).
// Honors type, label, showTitle, linkLabel, arrow, and textAlign.
func (e *Env) renderPostNavigationLink(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.AdjacentPost == nil {
		return ""
	}
	navType := "next"
	if t, ok := block.Attrs["type"].(string); ok {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "previous" || t == "next" {
			navType = t
		} else {
			return ""
		}
	}
	prev := navType == "previous"
	adj := e.AdjacentPost(e.currentPost.ID, prev)

	classes := "post-navigation-link-" + navType
	if ta := strings.TrimSpace(attrString(block.Attrs, "textAlign")); ta != "" {
		classes += " has-text-align-" + ta
	}
	classes += " wp-block-post-navigation-link"

	if adj == nil {
		return `<div class="` + html.EscapeString(classes) + `"></div>`
	}

	label := strings.TrimSpace(attrString(block.Attrs, "label"))
	showTitle := attrBool(block.Attrs, "showTitle")
	linkLabel := attrBool(block.Attrs, "linkLabel")
	arrowKind := strings.TrimSpace(attrString(block.Attrs, "arrow"))
	if arrowKind == "" {
		arrowKind = "none"
	}

	// Default link text: "Next" / "Previous"; custom label overrides.
	linkText := "Next"
	if prev {
		linkText = "Previous"
	}
	if label != "" {
		linkText = label
	}

	// Build inner <a> content (mirrors render_block_core_post_navigation_link).
	var linkInner string
	var beforeLink string
	if showTitle {
		title := adj.Title
		if !linkLabel {
			if label != "" {
				beforeLink = `<span class="post-navigation-link__label">` + html.EscapeString(label) + `</span> `
			}
			linkInner = html.EscapeString(title)
		} else if label != "" {
			linkInner = `<span class="post-navigation-link__label">` + html.EscapeString(label) +
				`</span> <span class="post-navigation-link__title">` + html.EscapeString(title) + `</span>`
		} else {
			colon := "Next:"
			if prev {
				colon = "Previous:"
			}
			linkInner = `<span class="post-navigation-link__label">` + colon +
				`</span> <span class="post-navigation-link__title">` + html.EscapeString(title) + `</span>`
		}
	} else {
		linkInner = html.EscapeString(linkText)
	}

	rel := "next"
	if prev {
		rel = "prev"
	}
	linkHTML := beforeLink + `<a href="` + html.EscapeString(adj.Link) + `" rel="` + rel + `">` + linkInner + `</a>`

	var arrowHTML string
	if arrowKind != "none" {
		var glyph string
		switch arrowKind {
		case "arrow":
			if prev {
				glyph = "←"
			} else {
				glyph = "→"
			}
		case "chevron":
			if prev {
				glyph = "«"
			} else {
				glyph = "»"
			}
		}
		if glyph != "" {
			side := "next"
			if prev {
				side = "previous"
			}
			arrowHTML = `<span class="wp-block-post-navigation-link__arrow-` + side +
				` is-arrow-` + html.EscapeString(arrowKind) + `" aria-hidden="true">` + glyph + `</span>`
		}
	}

	var b strings.Builder
	b.WriteString(`<div class="` + html.EscapeString(classes) + `">`)
	if prev {
		b.WriteString(arrowHTML)
		b.WriteString(linkHTML)
	} else {
		b.WriteString(linkHTML)
		b.WriteString(arrowHTML)
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
