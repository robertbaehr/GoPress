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

// QueryLoopArgs selects posts for a query block (P5.13 / P5.85).
type QueryLoopArgs struct {
	PerPage  int
	PostType string
	Offset   int
	Inherit  bool
	// Order / OrderBy mirror query.order / query.orderBy (P5.85).
	Order   string
	OrderBy string
	// Search is query.search when inherit is false (P5.85).
	Search string
	// Author is a single author ID from query.author (P5.85).
	Author int64
	// Sticky is "", "ignore", "exclude", or "only" (P5.85).
	Sticky string
	// Exclude is query.exclude post IDs (P5.85).
	Exclude []int64
	// Format is query.format post_format slugs, including "standard" (P5.85).
	Format []string
	// Parents is query.parents for hierarchical post types (P5.85).
	Parents []int64
	// CategoryIn / TagIn are legacy categoryIds / tagIds (P5.85).
	CategoryIn []int64
	TagIn      []int64
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
		return polishQueryWrapper(inner, block.Attrs)
	}
	tag := queryTagName(block.Attrs)
	classes := "wp-block-query"
	if align, ok := block.Attrs["align"].(string); ok && strings.TrimSpace(align) != "" {
		classes += " align" + strings.TrimSpace(align)
	}
	return `<` + tag + ` class="` + html.EscapeString(classes) + `">` + inner + `</` + tag + `>`
}

// queryTagName returns the core/query tagName attribute (default div, P5.85).
func queryTagName(attrs map[string]any) string {
	tag := "div"
	if attrs == nil {
		return tag
	}
	if t, ok := attrs["tagName"].(string); ok {
		t = strings.ToLower(strings.TrimSpace(t))
		switch t {
		case "div", "main", "section", "aside":
			return t
		}
	}
	return tag
}

// polishQueryWrapper ensures align/tagName on saved markup that already has wp-block-query (P5.85).
func polishQueryWrapper(inner string, attrs map[string]any) string {
	tag := queryTagName(attrs)
	if tag != "div" {
		// Replace the outermost <div class="…wp-block-query…"> with the requested tag.
		lower := strings.ToLower(inner)
		start := strings.Index(lower, `<div`)
		endOpen := strings.Index(inner[start:], ">")
		if start >= 0 && endOpen > 0 {
			openEnd := start + endOpen
			close := strings.LastIndex(lower, `</div>`)
			if close > openEnd {
				open := inner[start : openEnd+1]
				open = strings.Replace(open, "<div", "<"+tag, 1)
				open = strings.Replace(open, "<Div", "<"+tag, 1)
				inner = inner[:start] + open + inner[openEnd+1:close] + "</" + tag + ">" + inner[close+len("</div>"):]
			}
		}
	}
	if attrs != nil {
		if align, ok := attrs["align"].(string); ok && strings.TrimSpace(align) != "" {
			want := "align" + strings.TrimSpace(align)
			if !strings.Contains(inner, want) {
				inner = strings.Replace(inner, `class="wp-block-query"`, `class="wp-block-query `+want+`"`, 1)
				inner = strings.Replace(inner, `class="wp-block-query `, `class="wp-block-query `+want+` `, 1)
			}
		}
	}
	return inner
}

// renderQueryTitle renders core/query-title (search/archive/post-type, P3.21 / P5.71 / P5.96).
// P5.96: className / has-link-color / textColor / backgroundColor like get_block_wrapper_attributes.
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

	classes := queryTitleWrapperClasses(block.Attrs)

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

// queryTitleWrapperClasses mirrors get_block_wrapper_attributes class list for query-title (P5.96).
func queryTitleWrapperClasses(attrs map[string]any) string {
	parts := []string{"wp-block-query-title"}
	if ta := strings.TrimSpace(attrString(attrs, "textAlign")); ta != "" {
		parts = append(parts, "has-text-align-"+sanitizeClassToken(ta))
	}
	if hasLinkColorAttr(attrs) {
		parts = append(parts, "has-link-color")
	}
	if tc := strings.TrimSpace(attrString(attrs, "textColor")); tc != "" {
		parts = append(parts, "has-text-color", "has-"+sanitizeClassToken(tc)+"-color")
	}
	if bg := strings.TrimSpace(attrString(attrs, "backgroundColor")); bg != "" {
		parts = append(parts, "has-background", "has-"+sanitizeClassToken(bg)+"-background-color")
	}
	if hasCustomTextColor(attrs) {
		parts = append(parts, "has-text-color")
	}
	if hasCustomBackgroundColor(attrs) {
		parts = append(parts, "has-background")
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		parts = append(parts, cn)
	}
	return strings.Join(parts, " ")
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
	b.WriteString(`<ul class="`)
	b.WriteString(html.EscapeString(postTemplateClass(block)))
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

// postTemplateClass builds core/post-template wrapper classes (P5.85 / P5.119).
// Honors align, legacy displayLayout flex columns, layout.grid columnCount,
// has-link-color from style.elements.link.color.text, and className.
func postTemplateClass(block Block) string {
	parts := []string{"wp-block-post-template"}
	if align, ok := block.Attrs["align"].(string); ok && strings.TrimSpace(align) != "" {
		parts = append(parts, "align"+strings.TrimSpace(align))
	}
	// Legacy displayLayout on the parent query is passed via block attrs when nested
	// attrs include displayLayout (editor sometimes copies it onto post-template).
	if dl, ok := block.Attrs["displayLayout"].(map[string]any); ok {
		if typ, _ := dl["type"].(string); typ == "flex" {
			cols := 3
			if c, ok := dl["columns"].(float64); ok && c > 0 {
				cols = int(c)
			}
			parts = append(parts, "is-flex-container", "columns-"+strconv.Itoa(cols))
		}
	}
	if layout, ok := block.Attrs["layout"].(map[string]any); ok {
		if typ, _ := layout["type"].(string); typ == "grid" {
			if c, ok := layout["columnCount"].(float64); ok && c > 0 {
				parts = append(parts, "columns-"+strconv.Itoa(int(c)))
				if _, ok := layout["minimumColumnWidth"]; ok {
					parts = append(parts, "has-native-responsive-grid")
				}
			}
		}
	}
	if style, ok := block.Attrs["style"].(map[string]any); ok {
		if els, ok := style["elements"].(map[string]any); ok {
			if link, ok := els["link"].(map[string]any); ok {
				if color, ok := link["color"].(map[string]any); ok {
					if text, ok := color["text"]; ok && text != nil && fmt.Sprint(text) != "" {
						parts = append(parts, "has-link-color")
					}
				}
			}
		}
	}
	// P5.119: className like get_block_wrapper_attributes.
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		parts = append(parts, cn)
	}
	return strings.Join(parts, " ")
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

// renderQueryNoResults renders core/query-no-results (P5.3 / P5.96).
// P5.96: wrap inner content in a div with get_block_wrapper_attributes classes
// (has-link-color / className / colors) when the parent query has no posts.
func (e *Env) renderQueryNoResults(block Block) string {
	e = e.ensure()
	if e.queryHasPosts {
		return ""
	}
	inner := e.renderSaved(block)
	if strings.TrimSpace(inner) == "" {
		return ""
	}
	classes := queryNoResultsWrapperClasses(block.Attrs)
	return `<div class="` + html.EscapeString(classes) + `">` + inner + `</div>`
}

func queryNoResultsWrapperClasses(attrs map[string]any) string {
	parts := []string{"wp-block-query-no-results"}
	if hasLinkColorAttr(attrs) {
		parts = append(parts, "has-link-color")
	}
	if tc := strings.TrimSpace(attrString(attrs, "textColor")); tc != "" {
		parts = append(parts, "has-text-color", "has-"+sanitizeClassToken(tc)+"-color")
	}
	if bg := strings.TrimSpace(attrString(attrs, "backgroundColor")); bg != "" {
		parts = append(parts, "has-background", "has-"+sanitizeClassToken(bg)+"-background-color")
	}
	if hasCustomTextColor(attrs) {
		parts = append(parts, "has-text-color")
	}
	if hasCustomBackgroundColor(attrs) {
		parts = append(parts, "has-background")
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		parts = append(parts, cn)
	}
	return strings.Join(parts, " ")
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
	// P5.87: empty title → empty markup (render_block_core_post_title).
	rawTitle := strings.TrimSpace(e.currentPost.Title)
	if rawTitle == "" {
		return ""
	}
	title := html.EscapeString(rawTitle)
	tag := "h2"
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
	classes := []string{"wp-block-post-title"}
	if ta := strings.TrimSpace(attrString(block.Attrs, "textAlign")); ta != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(ta))
	}
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	inner := title
	if isLinkAttr(block.Attrs) && e.currentPost.Link != "" {
		aAttrs := ` href="` + html.EscapeString(e.currentPost.Link) + `"`
		target := strings.TrimSpace(attrString(block.Attrs, "linkTarget"))
		if target == "" {
			target = "_self"
		}
		aAttrs += ` target="` + html.EscapeString(target) + `"`
		if rel := strings.TrimSpace(attrString(block.Attrs, "rel")); rel != "" {
			aAttrs += ` rel="` + html.EscapeString(rel) + `"`
		}
		inner = `<a` + aAttrs + `>` + title + `</a>`
	}
	return fmt.Sprintf(`<%s class="%s">%s</%s>`, tag, html.EscapeString(strings.Join(classes, " ")), inner, tag)
}

func (e *Env) renderPostContent(block Block) string {
	e = e.ensure()
	if e.currentPost == nil {
		return ""
	}
	postID := e.currentPost.ID
	// P5.87: reentrancy guard (nested post-content for the same post).
	if e.seenPostContent == nil {
		e.seenPostContent = map[int64]bool{}
	}
	if e.seenPostContent[postID] {
		if e.DebugDisplay {
			return "[block rendering halted]"
		}
		return ""
	}
	e.seenPostContent[postID] = true
	defer delete(e.seenPostContent, postID)

	body := e.Render(e.currentPost.Content)
	if e.FilterPostHTML != nil {
		body = e.FilterPostHTML(body)
	}
	if strings.TrimSpace(body) == "" {
		return ""
	}
	tag := "div"
	if tn := strings.ToLower(strings.TrimSpace(attrString(block.Attrs, "tagName"))); tn != "" {
		switch tn {
		case "div", "main", "section", "article":
			tag = tn
		}
	}
	classes := []string{"entry-content", "wp-block-post-content"}
	if align := strings.TrimSpace(attrString(block.Attrs, "align")); align != "" {
		classes = append(classes, "align"+sanitizeClassToken(align))
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	return `<` + tag + ` class="` + html.EscapeString(strings.Join(classes, " ")) + `">` + body + `</` + tag + `>`
}

func (e *Env) renderPostExcerpt(block Block) string {
	e = e.ensure()
	if e.currentPost == nil {
		return ""
	}
	// P5.89: moreText / excerptLength / showMoreOnNewLine / textAlign / has-link-color.
	ex := strings.TrimSpace(e.currentPost.Excerpt)
	if ex == "" {
		ex = strings.TrimSpace(stripTags(e.currentPost.Content))
	}
	excerptLen := 55
	if v, ok := attrInt(block.Attrs, "excerptLength"); ok {
		excerptLen = v
	}
	if excerptLen > 0 && ex != "" {
		ex = trimExcerptWords(ex, excerptLen)
	}
	moreText := strings.TrimSpace(attrString(block.Attrs, "moreText"))
	moreHTML := ""
	if moreText != "" && e.currentPost.Link != "" {
		moreHTML = `<a class="wp-block-post-excerpt__more-link" href="` +
			html.EscapeString(e.currentPost.Link) + `">` + html.EscapeString(moreText) + `</a>`
	}
	if ex == "" && moreHTML == "" {
		return ""
	}

	classes := []string{}
	if ta := strings.TrimSpace(attrString(block.Attrs, "textAlign")); ta != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(ta))
	}
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	wrapperClass := "wp-block-post-excerpt"
	if len(classes) > 0 {
		wrapperClass += " " + strings.Join(classes, " ")
	}

	showMoreOnNewLine := true
	if block.Attrs != nil {
		if v, ok := block.Attrs["showMoreOnNewLine"]; ok {
			switch t := v.(type) {
			case bool:
				showMoreOnNewLine = t
			case string:
				showMoreOnNewLine = t == "true" || t == "1"
			}
		}
	}

	var inner strings.Builder
	inner.WriteString(`<p class="wp-block-post-excerpt__excerpt">`)
	inner.WriteString(html.EscapeString(ex))
	if moreHTML != "" && showMoreOnNewLine {
		inner.WriteString(`</p><p class="wp-block-post-excerpt__more-text">`)
		inner.WriteString(moreHTML)
		inner.WriteString(`</p>`)
	} else if moreHTML == "" {
		inner.WriteString(`</p>`)
	} else {
		if ex != "" {
			inner.WriteString(" ")
		}
		inner.WriteString(moreHTML)
		inner.WriteString(`</p>`)
	}
	return `<div class="` + html.EscapeString(wrapperClass) + `">` + inner.String() + `</div>`
}

// trimExcerptWords mirrors wp_trim_words for core/post-excerpt (P5.89 / P2.112).
// Uses Unicode ellipsis to match block save/render expectations.
func trimExcerptWords(text string, numWords int) string {
	if numWords < 0 {
		numWords = 0
	}
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\r' || r == '\t'
	})
	if len(words) > numWords {
		return strings.Join(words[:numWords], " ") + "…"
	}
	return strings.Join(words, " ")
}

func (e *Env) renderPostDate(block Block) string {
	e = e.ensure()
	if e.currentPost == nil {
		return ""
	}
	// P5.88: displayType=modified uses post modified time + modifier class.
	raw := strings.TrimSpace(e.currentPost.Date)
	modified := false
	if dt := strings.ToLower(strings.TrimSpace(attrString(block.Attrs, "displayType"))); dt == "modified" {
		raw = strings.TrimSpace(e.currentPost.Modified)
		if raw == "" {
			raw = strings.TrimSpace(e.currentPost.Date)
		}
		modified = true
	}
	if raw == "" {
		return ""
	}
	display := formatPostDateDisplayWithFormat(raw, attrString(block.Attrs, "format"))
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
		// WP render_block_core_post_date: link is href-only unless attrs set target/rel.
		aAttrs := ` href="` + html.EscapeString(e.currentPost.Link) + `"`
		if target := strings.TrimSpace(attrString(block.Attrs, "linkTarget")); target != "" {
			aAttrs += ` target="` + html.EscapeString(target) + `"`
		}
		if rel := strings.TrimSpace(attrString(block.Attrs, "rel")); rel != "" {
			aAttrs += ` rel="` + html.EscapeString(rel) + `"`
		}
		inner = `<a` + aAttrs + `>` + timeHTML + `</a>`
	}
	classes := []string{"wp-block-post-date"}
	if modified {
		classes = append(classes, "wp-block-post-date__modified-date")
	}
	if ta := strings.TrimSpace(attrString(block.Attrs, "textAlign")); ta != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(ta))
	}
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	return `<div class="` + html.EscapeString(strings.Join(classes, " ")) + `">` + inner + `</div>`
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
	// P5.93: textAlign / has-link-color / className like get_block_wrapper_attributes.
	classes := []string{"wp-block-site-title"}
	if align := strings.TrimSpace(attrString(block.Attrs, "textAlign")); align != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(align))
	}
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	inner := html.EscapeString(name)
	// isLink defaults to true (block.json).
	if isLinkAttrDefaultTrue(block.Attrs) && e.HomeURL != nil {
		home := strings.TrimRight(e.HomeURL(), "/")
		if home != "" {
			target := "_self"
			if t := strings.TrimSpace(attrString(block.Attrs, "linkTarget")); t != "" {
				target = t
			}
			aria := ""
			if e.isSiteTitleCurrent() {
				aria = ` aria-current="page"`
			}
			inner = `<a` + aria + ` href="` + html.EscapeString(home) + `" target="` + html.EscapeString(target) + `" rel="home">` + inner + `</a>`
		}
	}
	return `<` + tag + ` class="` + html.EscapeString(strings.Join(classes, " ")) + `">` + inner + `</` + tag + `>`
}

// isSiteTitleCurrent mirrors site-title aria-current: !is_paged && (front || home edge) (P5.93).
func (e *Env) isSiteTitleCurrent() bool {
	if e == nil {
		return false
	}
	if e.IsPaged != nil && e.IsPaged() {
		return false
	}
	return e.isHomeLinkCurrent()
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
	// P5.93: textAlign / className on wrapper.
	classes := []string{"wp-block-site-tagline"}
	if align := strings.TrimSpace(attrString(block.Attrs, "textAlign")); align != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(align))
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	return `<` + tag + ` class="` + html.EscapeString(strings.Join(classes, " ")) + `">` + html.EscapeString(tagline) + `</` + tag + `>`
}

// renderSiteLogo renders core/site-logo from the site_logo / custom logo URL (P5.45 / P5.120).
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
	// P5.120: adjust height from attachment aspect ratio when width is set
	// (mirrors wp_get_attachment_image_src filter in render_block_core_site_logo).
	var height int64
	if width > 0 && e.SiteLogoID != nil && e.GetAttachmentDimensions != nil {
		if id := e.SiteLogoID(); id > 0 {
			if natW, natH := e.GetAttachmentDimensions(id); natW > 0 && natH > 0 {
				height = int64(float64(width) / (float64(natW) / float64(natH)))
				if height < 1 {
					height = 1
				}
			}
		}
	}
	var imgB strings.Builder
	imgB.WriteString(`<img`)
	if width > 0 {
		imgB.WriteString(` width="`)
		imgB.WriteString(strconv.FormatInt(width, 10))
		imgB.WriteString(`"`)
	}
	if height > 0 {
		imgB.WriteString(` height="`)
		imgB.WriteString(strconv.FormatInt(height, 10))
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
			// P5.120: aria-current like get_custom_logo / site-title (!is_paged front/home).
			current := ""
			if e.isSiteTitleCurrent() {
				current = ` aria-current="page"`
			}
			inner = `<a` + current + ` href="` + html.EscapeString(home) + `" class="custom-logo-link" rel="home"` + aria + ` target="` + html.EscapeString(target) + `">` + img + `</a>`
		}
	}
	// P5.120: className / align like get_block_wrapper_attributes.
	classes := []string{"wp-block-site-logo"}
	if width <= 0 {
		classes = append(classes, "is-default-size")
	}
	if align := strings.TrimSpace(attrString(block.Attrs, "align")); align != "" {
		classes = append(classes, "align"+sanitizeClassToken(align))
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	return `<div class="` + html.EscapeString(strings.Join(classes, " ")) + `">` + inner + `</div>`
}

// renderReadMore renders core/read-more as a permalink link (P5.47 / P5.92).
func (e *Env) renderReadMore(block Block) string {
	e = e.ensure()
	if e.currentPost == nil {
		return ""
	}
	link := strings.TrimSpace(e.currentPost.Link)
	if link == "" && e.currentPost.ID > 0 {
		link = fmt.Sprintf("/?p=%d", e.currentPost.ID)
	}
	if link == "" {
		return ""
	}
	title := strings.TrimSpace(e.currentPost.Title)
	if title == "" {
		title = fmt.Sprintf("untitled post %d", e.currentPost.ID)
	}
	moreText := "Read more"
	if content := strings.TrimSpace(attrString(block.Attrs, "content")); content != "" {
		moreText = content
	}
	target := "_self"
	if t := strings.TrimSpace(attrString(block.Attrs, "linkTarget")); t != "" {
		target = t
	}
	classes := []string{"wp-block-read-more"}
	if justify := strings.TrimSpace(attrString(block.Attrs, "justifyContent")); justify != "" {
		classes = append(classes, "is-justified-"+sanitizeClassToken(justify))
	}
	// P5.92: className + has-link-color like get_block_wrapper_attributes.
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	screenReader := ": " + title
	return `<a class="` + html.EscapeString(strings.Join(classes, " ")) + `" href="` + html.EscapeString(link) +
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
	return formatPostDateDisplayWithFormat(mysql, "")
}

// formatPostDateDisplayWithFormat honors block format attr (P5.88), including human-diff.
func formatPostDateDisplayWithFormat(mysql, format string) string {
	mysql = strings.TrimSpace(mysql)
	format = strings.TrimSpace(format)
	if mysql == "" {
		return ""
	}
	t, err := parseMySQLDate(mysql)
	if err != nil {
		if format == "" {
			return displayDate(mysql)
		}
		return mysql
	}
	if format == "human-diff" {
		diff := humanTimeDiffLocal(t, time.Now().UTC())
		if t.After(time.Now().UTC()) {
			return diff + " from now"
		}
		return diff + " ago"
	}
	if format != "" {
		if got := formatPHPDateSubset(t, format); got != "" {
			return got
		}
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

// formatPHPDateSubset maps a small PHP date() subset used by post-date formats.
func formatPHPDateSubset(t time.Time, php string) string {
	var b strings.Builder
	for i := 0; i < len(php); i++ {
		ch := php[i]
		switch ch {
		case '\\':
			if i+1 < len(php) {
				i++
				b.WriteByte(php[i])
			}
		case 'Y':
			b.WriteString(t.Format("2006"))
		case 'y':
			b.WriteString(t.Format("06"))
		case 'm':
			b.WriteString(t.Format("01"))
		case 'n':
			b.WriteString(strconv.Itoa(int(t.Month())))
		case 'd':
			b.WriteString(t.Format("02"))
		case 'j':
			b.WriteString(strconv.Itoa(t.Day()))
		case 'F':
			b.WriteString(t.Format("January"))
		case 'M':
			b.WriteString(t.Format("Jan"))
		case 'H':
			b.WriteString(t.Format("15"))
		case 'i':
			b.WriteString(t.Format("04"))
		case 's':
			b.WriteString(t.Format("05"))
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

func humanTimeDiffLocal(from, to time.Time) string {
	sec := int64(to.Sub(from).Seconds())
	if sec < 0 {
		sec = -sec
	}
	switch {
	case sec < 60:
		if sec <= 1 {
			return "1 second"
		}
		return strconv.FormatInt(sec, 10) + " seconds"
	case sec < 3600:
		m := (sec + 30) / 60
		if m <= 1 {
			return "1 minute"
		}
		return strconv.FormatInt(m, 10) + " minutes"
	case sec < 86400:
		h := (sec + 1800) / 3600
		if h <= 1 {
			return "1 hour"
		}
		return strconv.FormatInt(h, 10) + " hours"
	case sec < 604800:
		d := (sec + 43200) / 86400
		if d <= 1 {
			return "1 day"
		}
		return strconv.FormatInt(d, 10) + " days"
	case sec < 2628000:
		w := (sec + 302400) / 604800
		if w <= 1 {
			return "1 week"
		}
		return strconv.FormatInt(w, 10) + " weeks"
	case sec < 31536000:
		mo := (sec + 1314000) / 2628000
		if mo <= 1 {
			return "1 month"
		}
		return strconv.FormatInt(mo, 10) + " months"
	default:
		y := (sec + 15768000) / 31536000
		if y <= 1 {
			return "1 year"
		}
		return strconv.FormatInt(y, 10) + " years"
	}
}

func showAvatarAttr(attrs map[string]any) bool {
	if attrs == nil {
		return false
	}
	v, ok := attrs["showAvatar"]
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

func showBioAttr(attrs map[string]any) bool {
	if attrs == nil {
		return false
	}
	v, ok := attrs["showBio"]
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
	isLink := isLinkAttr(block.Attrs)
	// P5.86: when linked, prefer the post title as alt (WP post-featured-image).
	if isLink {
		if title := strings.TrimSpace(e.currentPost.Title); title != "" {
			alt = title
		} else if e.currentPost.ID > 0 {
			alt = fmt.Sprintf("Untitled post %d", e.currentPost.ID)
		}
	}

	sizeSlug := strings.TrimSpace(attrString(block.Attrs, "sizeSlug"))
	if sizeSlug == "" {
		sizeSlug = "post-thumbnail"
	}
	imgClass := "attachment-" + sizeSlug + " size-" + sizeSlug

	var styleParts []string
	aspect := strings.TrimSpace(attrString(block.Attrs, "aspectRatio"))
	width := strings.TrimSpace(attrString(block.Attrs, "width"))
	height := strings.TrimSpace(attrString(block.Attrs, "height"))
	scale := strings.TrimSpace(attrString(block.Attrs, "scale"))
	if scale == "" {
		scale = "cover"
	}
	if aspect != "" && aspect != "auto" {
		styleParts = append(styleParts, "aspect-ratio:"+sanitizeCSSToken(aspect))
		styleParts = append(styleParts, "width:100%")
	}
	if height != "" {
		styleParts = append(styleParts, "height:"+sanitizeCSSLength(height))
	} else if width != "" {
		styleParts = append(styleParts, "height:auto")
	}
	if width != "" {
		styleParts = append(styleParts, "width:"+sanitizeCSSLength(width))
	}
	if scale != "" {
		styleParts = append(styleParts, "object-fit:"+sanitizeCSSToken(scale))
	}

	imgAttrs := ` src="` + html.EscapeString(url) + `" alt="` + html.EscapeString(alt) + `" class="` + html.EscapeString(imgClass) + `"`
	if len(styleParts) > 0 {
		imgAttrs += ` style="` + html.EscapeString(strings.Join(styleParts, ";")) + `;"`
	}
	img := `<img` + imgAttrs + `/>`
	overlay := featuredImageOverlayMarkup(block.Attrs)

	inner := img + overlay
	if isLink && e.currentPost.Link != "" {
		aAttrs := ` href="` + html.EscapeString(e.currentPost.Link) + `"`
		target := strings.TrimSpace(attrString(block.Attrs, "linkTarget"))
		if target == "" {
			target = "_self"
		}
		aAttrs += ` target="` + html.EscapeString(target) + `"`
		if rel := strings.TrimSpace(attrString(block.Attrs, "rel")); rel != "" {
			aAttrs += ` rel="` + html.EscapeString(rel) + `"`
		}
		inner = `<a` + aAttrs + `>` + img + overlay + `</a>`
	}

	figClass := "wp-block-post-featured-image"
	if align := strings.TrimSpace(attrString(block.Attrs, "align")); align != "" {
		figClass += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		figClass += " " + cn
	}
	return `<figure class="` + html.EscapeString(figClass) + `">` + inner + `</figure>`
}

// featuredImageOverlayMarkup mirrors get_block_core_post_featured_image_overlay_element_markup (P5.86).
func featuredImageOverlayMarkup(attrs map[string]any) string {
	dim := int(attrInt64(attrs, "dimRatio"))
	if dim <= 0 {
		return ""
	}
	classes := []string{
		"wp-block-post-featured-image__overlay",
		"has-background-dim",
		fmt.Sprintf("has-background-dim-%d", dim),
	}
	var styles []string
	if overlay := strings.TrimSpace(attrString(attrs, "overlayColor")); overlay != "" {
		classes = append(classes, "has-"+sanitizeClassToken(overlay)+"-background-color")
	}
	if grad := strings.TrimSpace(attrString(attrs, "gradient")); grad != "" {
		classes = append(classes, "has-background-gradient", "has-"+sanitizeClassToken(grad)+"-gradient-background")
	}
	if customGrad := strings.TrimSpace(attrString(attrs, "customGradient")); customGrad != "" {
		classes = append(classes, "has-background-gradient")
		styles = append(styles, "background-image:"+customGrad)
	}
	if custom := strings.TrimSpace(attrString(attrs, "customOverlayColor")); custom != "" {
		styles = append(styles, "background-color:"+custom)
	}
	out := `<span class="` + html.EscapeString(strings.Join(classes, " ")) + `"`
	if len(styles) > 0 {
		out += ` style="` + html.EscapeString(strings.Join(styles, ";")) + `;"`
	}
	out += ` aria-hidden="true"></span>`
	return out
}

func queryArgsFromAttrs(attrs map[string]any) QueryLoopArgs {
	args := QueryLoopArgs{PerPage: 10, PostType: "post", Order: "desc", OrderBy: "date"}
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
	// P5.85: order / orderBy / search / author / sticky / exclude / format / parents.
	if o, ok := q["order"].(string); ok && strings.TrimSpace(o) != "" {
		args.Order = strings.ToLower(strings.TrimSpace(o))
	}
	if ob, ok := q["orderBy"].(string); ok && strings.TrimSpace(ob) != "" {
		args.OrderBy = strings.TrimSpace(ob)
	}
	if s, ok := q["search"].(string); ok {
		args.Search = strings.TrimSpace(s)
	}
	if a, ok := q["author"]; ok {
		args.Author = firstAuthorID(a)
	}
	if st, ok := q["sticky"].(string); ok {
		args.Sticky = strings.ToLower(strings.TrimSpace(st))
	}
	args.Exclude = int64SliceFromAny(q["exclude"])
	args.Parents = int64SliceFromAny(q["parents"])
	args.CategoryIn = int64SliceFromAny(q["categoryIds"])
	args.TagIn = int64SliceFromAny(q["tagIds"])
	if formats, ok := q["format"].([]any); ok {
		for _, f := range formats {
			s := strings.TrimSpace(fmt.Sprint(f))
			if s != "" && s != "<nil>" {
				args.Format = append(args.Format, s)
			}
		}
	}
	return args
}

func firstAuthorID(v any) int64 {
	switch t := v.(type) {
	case float64:
		if t > 0 {
			return int64(t)
		}
	case string:
		parts := strings.Split(t, ",")
		if len(parts) > 0 {
			n, _ := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
			if n > 0 {
				return n
			}
		}
	case []any:
		for _, item := range t {
			if n := firstAuthorID(item); n > 0 {
				return n
			}
		}
	}
	return 0
}

func int64SliceFromAny(v any) []int64 {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	out := make([]int64, 0, len(arr))
	for _, item := range arr {
		n := attrInt64(map[string]any{"v": item}, "v")
		if n > 0 {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
	// P5.90: no post context → empty (mirrors render_block_core_comments).
	if e.currentPost == nil {
		return ""
	}
	inner := e.renderSaved(block)
	if strings.Contains(inner, "wp-block-comments") || strings.Contains(inner, "wp-block-post-comments") {
		return inner
	}
	classes := []string{"wp-block-comments"}
	// Legacy core/post-comments / attributes.legacy keep the old class (P5.90).
	legacyName := ""
	if block.BlockName != nil {
		legacyName = *block.BlockName
	}
	if attrBool(block.Attrs, "legacy") || legacyName == "core/post-comments" {
		classes = append(classes, "wp-block-post-comments")
	}
	if ta := strings.TrimSpace(attrString(block.Attrs, "textAlign")); ta != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(ta))
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	return `<div class="` + html.EscapeString(strings.Join(classes, " ")) + `">` + inner + `</div>`
}

func (e *Env) renderQueryPagination(block Block) string {
	e = e.ensure()
	// P5.119: provide paginationArrow / showLabel context to inner blocks.
	prevArrow, prevShow := e.paginationArrow, e.showLabel
	e.paginationArrow = paginationArrowFromAttrs(block.Attrs)
	e.showLabel = showLabelPtrFromAttrs(block.Attrs)
	defer func() {
		e.paginationArrow = prevArrow
		e.showLabel = prevShow
	}()

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
	classes := queryPaginationWrapperClasses(block.Attrs)
	return `<nav aria-label="Pagination" class="` + html.EscapeString(classes) + `">` + inner + `</nav>`
}

// queryPaginationWrapperClasses mirrors get_block_wrapper_attributes for query-pagination (P5.119).
func queryPaginationWrapperClasses(attrs map[string]any) string {
	parts := []string{"wp-block-query-pagination"}
	if align, ok := attrs["align"].(string); ok && strings.TrimSpace(align) != "" {
		parts = append(parts, "align"+strings.TrimSpace(align))
	}
	if hasLinkColorAttr(attrs) {
		parts = append(parts, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		parts = append(parts, cn)
	}
	return strings.Join(parts, " ")
}

// paginationArrowFromAttrs reads paginationArrow (none|arrow|chevron); empty → "arrow" (P5.119).
func paginationArrowFromAttrs(attrs map[string]any) string {
	a := strings.ToLower(strings.TrimSpace(attrString(attrs, "paginationArrow")))
	switch a {
	case "none", "arrow", "chevron":
		return a
	}
	return "arrow" // legacy GoPress default when unset (tests + pre-P5.119 markup)
}

func showLabelPtrFromAttrs(attrs map[string]any) *bool {
	if attrs == nil {
		return nil
	}
	if _, ok := attrs["showLabel"]; !ok {
		return nil
	}
	v := attrBool(attrs, "showLabel")
	return &v
}

func (e *Env) queryPaginationShowLabel() bool {
	e = e.ensure()
	if e.showLabel != nil {
		return *e.showLabel
	}
	return true
}

func (e *Env) queryPaginationArrow() string {
	e = e.ensure()
	if e.paginationArrow != "" {
		return e.paginationArrow
	}
	return "arrow"
}

// queryPaginationArrowHTML mirrors get_query_pagination_arrow (P5.119).
func queryPaginationArrowHTML(arrowKind string, isNext bool) string {
	arrowKind = strings.ToLower(strings.TrimSpace(arrowKind))
	if arrowKind == "" || arrowKind == "none" {
		return ""
	}
	var glyph string
	switch arrowKind {
	case "chevron":
		if isNext {
			glyph = "»"
		} else {
			glyph = "«"
		}
	case "arrow":
		if isNext {
			glyph = "→"
		} else {
			glyph = "←"
		}
	default:
		return ""
	}
	typ := "previous"
	if isNext {
		typ = "next"
	}
	return `<span class="wp-block-query-pagination-` + typ + `-arrow is-arrow-` + html.EscapeString(arrowKind) + `" aria-hidden="true">` + glyph + `</span>`
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
	labelText := "Previous Page"
	if l, ok := block.Attrs["label"].(string); ok && strings.TrimSpace(l) != "" {
		labelText = strings.TrimSpace(l)
	}
	showLabel := e.queryPaginationShowLabel()
	if block.Attrs != nil {
		if _, ok := block.Attrs["showLabel"]; ok {
			showLabel = attrBool(block.Attrs, "showLabel")
		}
	}
	arrowKind := e.queryPaginationArrow()
	// Standalone / synthesize: read arrow from this block's attrs when parent context unset.
	if e.paginationArrow == "" && block.Attrs != nil {
		if _, ok := block.Attrs["paginationArrow"]; ok {
			arrowKind = paginationArrowFromAttrs(block.Attrs)
		}
	}
	label := ""
	if showLabel {
		label = html.EscapeString(labelText)
	}
	arrow := queryPaginationArrowHTML(arrowKind, false)
	if arrow != "" {
		label = arrow + label
	}
	href := queryPaginationPageURL(e.queryPaginationHome(), page-1)
	aria := ""
	if !showLabel {
		aria = ` aria-label="` + html.EscapeString(labelText) + `"`
	}
	return `<a href="` + html.EscapeString(href) + `" class="wp-block-query-pagination-previous"` + aria + `>` +
		label + `</a>`
}

func (e *Env) renderQueryPaginationNext(block Block) string {
	e = e.ensure()
	page := e.queryPaginationPage()
	max := e.queryPaginationMaxPages()
	if page >= max {
		return ""
	}
	labelText := "Next Page"
	if l, ok := block.Attrs["label"].(string); ok && strings.TrimSpace(l) != "" {
		labelText = strings.TrimSpace(l)
	}
	showLabel := e.queryPaginationShowLabel()
	if block.Attrs != nil {
		if _, ok := block.Attrs["showLabel"]; ok {
			showLabel = attrBool(block.Attrs, "showLabel")
		}
	}
	arrowKind := e.queryPaginationArrow()
	if e.paginationArrow == "" && block.Attrs != nil {
		if _, ok := block.Attrs["paginationArrow"]; ok {
			arrowKind = paginationArrowFromAttrs(block.Attrs)
		}
	}
	label := ""
	if showLabel {
		label = html.EscapeString(labelText)
	}
	arrow := queryPaginationArrowHTML(arrowKind, true)
	if arrow != "" {
		label = label + arrow
	}
	href := queryPaginationPageURL(e.queryPaginationHome(), page+1)
	aria := ""
	if !showLabel {
		aria = ` aria-label="` + html.EscapeString(labelText) + `"`
	}
	return `<a href="` + html.EscapeString(href) + `" class="wp-block-query-pagination-next"` + aria + `>` +
		label + `</a>`
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

// renderPostAuthorName renders core/post-author-name (P5.46 / P5.91).
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
	// P5.91: textAlign / has-link-color / className on wrapper.
	classes := []string{"wp-block-post-author-name"}
	if ta := strings.TrimSpace(attrString(block.Attrs, "textAlign")); ta != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(ta))
	}
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	return `<div class="` + html.EscapeString(strings.Join(classes, " ")) + `">` + inner + `</div>`
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

// renderAvatar renders core/avatar for the current post author (P5.46 / P5.91).
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
	// P5.91: textAlign / has-link-color / className on wrapper.
	classes := []string{"wp-block-avatar"}
	if ta := strings.TrimSpace(attrString(block.Attrs, "textAlign")); ta != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(ta))
	}
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	return `<div class="` + html.EscapeString(strings.Join(classes, " ")) + `">` + inner + `</div>`
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

// renderPostAuthor renders core/post-author (P5.43 / P5.88).
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
	var outer strings.Builder
	// P5.88: optional avatar when showAvatar is set.
	if showAvatarAttr(block.Attrs) {
		size := int64(48)
		if s := attrInt64(block.Attrs, "avatarSize"); s > 0 {
			size = s
		}
		email := ""
		if e.AuthorEmail != nil {
			email = strings.TrimSpace(e.AuthorEmail(e.currentPost.ID))
		}
		if email != "" {
			hash := gravatarHash(email)
			src := fmt.Sprintf("https://secure.gravatar.com/avatar/%s?s=%d&d=mm&r=g", hash, size)
			alt := html.EscapeString(name)
			outer.WriteString(`<div class="wp-block-post-author__avatar">`)
			outer.WriteString(fmt.Sprintf(
				`<img alt="%s" src="%s" srcset="%s" class="avatar avatar-%d photo" height="%d" width="%d" decoding="async"/>`,
				alt, html.EscapeString(src),
				html.EscapeString(fmt.Sprintf("https://secure.gravatar.com/avatar/%s?s=%d&d=mm&r=g 2x", hash, size*2)),
				size, size, size,
			))
			outer.WriteString(`</div>`)
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
	content.WriteString(`</p>`)
	if showBioAttr(block.Attrs) && e.AuthorBio != nil {
		if bio := strings.TrimSpace(e.AuthorBio(e.currentPost.ID)); bio != "" {
			content.WriteString(`<p class="wp-block-post-author__bio">`)
			content.WriteString(html.EscapeString(bio))
			content.WriteString(`</p>`)
		}
	}
	content.WriteString(`</div>`)
	outer.WriteString(content.String())
	classes := []string{"wp-block-post-author"}
	if j := strings.TrimSpace(attrString(block.Attrs, "itemsJustification")); j != "" {
		classes = append(classes, "items-justified-"+sanitizeClassToken(j))
	}
	if ta := strings.TrimSpace(attrString(block.Attrs, "textAlign")); ta != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(ta))
	}
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	return `<div class="` + html.EscapeString(strings.Join(classes, " ")) + `">` + outer.String() + `</div>`
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
	classes := []string{"taxonomy-" + tax, "wp-block-post-terms"}
	if ta := strings.TrimSpace(attrString(block.Attrs, "textAlign")); ta != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(ta))
	}
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	var b strings.Builder
	b.WriteString(`<div class="` + html.EscapeString(strings.Join(classes, " ")) + `">`)
	// P5.88: optional prefix / suffix wrappers (render_block_core_post_terms).
	if prefix := attrString(block.Attrs, "prefix"); strings.TrimSpace(prefix) != "" {
		b.WriteString(`<span class="wp-block-post-terms__prefix">`)
		b.WriteString(html.EscapeString(prefix))
		b.WriteString(`</span>`)
	}
	sep := " "
	if s, ok := block.Attrs["separator"].(string); ok {
		sep = s
	}
	for i, t := range terms {
		if i > 0 {
			b.WriteString(`<span class="wp-block-post-terms__separator">`)
			b.WriteString(html.EscapeString(sep))
			b.WriteString(`</span>`)
		}
		b.WriteString(`<a href="` + html.EscapeString(t.URL) + `" rel="tag">`)
		b.WriteString(html.EscapeString(t.Label))
		b.WriteString(`</a>`)
	}
	if suffix := attrString(block.Attrs, "suffix"); strings.TrimSpace(suffix) != "" {
		b.WriteString(`<span class="wp-block-post-terms__suffix">`)
		b.WriteString(html.EscapeString(suffix))
		b.WriteString(`</span>`)
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
	showPostTitle := true
	if _, ok := block.Attrs["showPostTitle"]; ok {
		showPostTitle = attrBool(block.Attrs, "showPostTitle")
	}
	title := e.currentPost.Title
	var text string
	if showPostTitle {
		if n == 1 {
			text = "One response to “" + title + "”"
		} else {
			text = strconv.Itoa(n) + " responses to “" + title + "”"
		}
	} else if n == 1 {
		text = "One response"
	} else {
		text = strconv.Itoa(n) + " responses"
	}
	classes := []string{"wp-block-comments-title"}
	if ta := strings.TrimSpace(attrString(block.Attrs, "textAlign")); ta != "" {
		classes = append(classes, "has-text-align-"+sanitizeClassToken(ta))
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	return `<` + tag + ` class="` + html.EscapeString(strings.Join(classes, " ")) +
		`" id="comments">` + html.EscapeString(text) + `</` + tag + `>`
}

// renderPostNavigationLink renders core/post-navigation-link (P5.3 / P5.72 / P5.97).
// Honors type, label, showTitle, linkLabel, arrow, textAlign, taxonomy,
// and wrapper className / has-link-color / textColor / backgroundColor.
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
	taxonomy := strings.TrimSpace(attrString(block.Attrs, "taxonomy"))
	adj := e.AdjacentPost(e.currentPost.ID, prev, taxonomy)

	classes := postNavigationLinkWrapperClasses(navType, block.Attrs)

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

// postNavigationLinkWrapperClasses mirrors get_block_wrapper_attributes for
// core/post-navigation-link (P5.97): textAlign, has-link-color, colors, className.
func postNavigationLinkWrapperClasses(navType string, attrs map[string]any) string {
	parts := []string{"post-navigation-link-" + navType}
	if ta := strings.TrimSpace(attrString(attrs, "textAlign")); ta != "" {
		parts = append(parts, "has-text-align-"+sanitizeClassToken(ta))
	}
	parts = append(parts, "wp-block-post-navigation-link")
	if hasLinkColorAttr(attrs) {
		parts = append(parts, "has-link-color")
	}
	if tc := strings.TrimSpace(attrString(attrs, "textColor")); tc != "" {
		parts = append(parts, "has-text-color", "has-"+sanitizeClassToken(tc)+"-color")
	}
	if bg := strings.TrimSpace(attrString(attrs, "backgroundColor")); bg != "" {
		parts = append(parts, "has-background", "has-"+sanitizeClassToken(bg)+"-background-color")
	}
	if hasCustomTextColor(attrs) {
		parts = append(parts, "has-text-color")
	}
	if hasCustomBackgroundColor(attrs) {
		parts = append(parts, "has-background")
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		parts = append(parts, cn)
	}
	return strings.Join(parts, " ")
}

// renderCommentsPagination wraps previous/numbers/next (P3.108).
func (e *Env) renderCommentsPagination(block Block) string {
	e = e.ensure()
	if !e.PageComments || e.CommentTotalPages < 2 {
		return ""
	}
	inner := e.renderSaved(block)
	if strings.TrimSpace(inner) == "" {
		inner = e.renderCommentsPaginationPrevious(block) +
			e.renderCommentsPaginationNumbers(block) +
			e.renderCommentsPaginationNext(block)
	}
	if strings.TrimSpace(inner) == "" {
		return ""
	}
	classes := []string{"wp-block-comments-pagination"}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	return `<nav class="` + html.EscapeString(strings.Join(classes, " ")) + `" aria-label="Comments pagination">` + inner + `</nav>`
}

func (e *Env) renderCommentsPaginationPrevious(block Block) string {
	e = e.ensure()
	if !e.PageComments || e.CommentPage <= 1 || e.CommentTotalPages < 2 {
		return ""
	}
	label := strings.TrimSpace(attrString(block.Attrs, "label"))
	if label == "" {
		label = "Older Comments"
	}
	href := e.commentPageHref(e.CommentPage - 1)
	return `<a class="wp-block-comments-pagination-previous" href="` + html.EscapeString(href) + `">` + html.EscapeString(label) + `</a>`
}

func (e *Env) renderCommentsPaginationNext(block Block) string {
	e = e.ensure()
	if !e.PageComments || e.CommentPage >= e.CommentTotalPages || e.CommentTotalPages < 2 {
		return ""
	}
	label := strings.TrimSpace(attrString(block.Attrs, "label"))
	if label == "" {
		label = "Newer Comments"
	}
	href := e.commentPageHref(e.CommentPage + 1)
	return `<a class="wp-block-comments-pagination-next" href="` + html.EscapeString(href) + `">` + html.EscapeString(label) + `</a>`
}

func (e *Env) renderCommentsPaginationNumbers(block Block) string {
	e = e.ensure()
	_ = block
	if !e.PageComments || e.CommentTotalPages < 2 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="wp-block-comments-pagination-numbers">`)
	for n := 1; n <= e.CommentTotalPages; n++ {
		if n == e.CommentPage {
			b.WriteString(`<span aria-current="page" class="page-numbers current">`)
			b.WriteString(strconv.Itoa(n))
			b.WriteString(`</span>`)
			continue
		}
		b.WriteString(`<a class="page-numbers" href="`)
		b.WriteString(html.EscapeString(e.commentPageHref(n)))
		b.WriteString(`">`)
		b.WriteString(strconv.Itoa(n))
		b.WriteString(`</a>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func (e *Env) commentPageHref(page int) string {
	e = e.ensure()
	base := strings.TrimSpace(e.CommentPermalink)
	if base == "" && e.HomeURL != nil {
		base = strings.TrimRight(e.HomeURL(), "/") + "/"
	}
	if base == "" || page <= 0 {
		return "#comments"
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	// Strip existing cpage.
	if i := strings.Index(base, "cpage="); i >= 0 {
		// crude cleanup: rebuild without cpage via query parse is heavier; append wins in browsers poorly.
		// Prefer simple append; callers pass clean permalinks from front.
	}
	_ = sep
	href := base
	if strings.Contains(href, "?") {
		href += "&cpage=" + strconv.Itoa(page)
	} else {
		href += "?cpage=" + strconv.Itoa(page)
	}
	return href + "#comments"
}

func (e *Env) renderCommentTemplate(block Block) string {
	e = e.ensure()
	if e.currentPost == nil || e.ListComments == nil {
		return ""
	}
	if e.PasswordRequired != nil && e.PasswordRequired(e.currentPost.ID) {
		return ""
	}
	comments := e.ListComments(e.currentPost.ID)
	if len(comments) == 0 {
		return ""
	}
	// P5.90: honor comment_order=desc (WP reverses top-level list).
	if strings.EqualFold(strings.TrimSpace(e.CommentOrder), "desc") {
		for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
			comments[i], comments[j] = comments[j], comments[i]
		}
	}
	byParent := map[int64][]CommentItem{}
	var roots []CommentItem
	for _, cm := range comments {
		if cm.Parent > 0 {
			byParent[cm.Parent] = append(byParent[cm.Parent], cm)
		} else {
			roots = append(roots, cm)
		}
	}
	if len(roots) == 0 {
		// Flat fallback when parents are missing from the approved set.
		roots = comments
		byParent = nil
	}
	// P3.108: page_comments slices top-level comments by cpage.
	if e.PageComments && e.CommentsPerPage > 0 && e.CommentPage > 0 {
		start := (e.CommentPage - 1) * e.CommentsPerPage
		if start >= len(roots) {
			roots = nil
		} else {
			end := start + e.CommentsPerPage
			if end > len(roots) {
				end = len(roots)
			}
			roots = roots[start:end]
		}
	}
	classes := []string{"wp-block-comment-template"}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	var b strings.Builder
	b.WriteString(`<ol class="` + html.EscapeString(strings.Join(classes, " ")) + `">`)
	e.writeCommentTemplateItems(&b, roots, byParent, 1, true)
	b.WriteString(`</ol>`)
	return b.String()
}

func (e *Env) writeCommentTemplateItems(b *strings.Builder, items []CommentItem, byParent map[int64][]CommentItem, depth int, threadEven bool) {
	e = e.ensure()
	for i, cm := range items {
		even := i%2 == 0
		parity := "odd"
		if even {
			parity = "even"
		}
		threadParity := "thread-odd"
		if threadEven == even || depth > 1 {
			if even {
				threadParity = "thread-even"
			} else {
				threadParity = "thread-odd"
			}
		} else {
			if threadEven {
				threadParity = "thread-even"
			} else {
				threadParity = "thread-odd"
			}
		}
		if depth == 1 {
			if even {
				threadParity = "thread-even"
			} else {
				threadParity = "thread-odd"
			}
		}
		b.WriteString(`<li id="comment-`)
		b.WriteString(strconv.FormatInt(cm.ID, 10))
		b.WriteString(`" class="comment `)
		b.WriteString(parity)
		b.WriteString(` `)
		b.WriteString(threadParity)
		b.WriteString(` depth-`)
		b.WriteString(strconv.Itoa(depth))
		b.WriteString(`">`)
		b.WriteString(e.renderCommentTemplateItemBody(cm))
		if kids := byParent[cm.ID]; len(kids) > 0 && depth < 10 {
			b.WriteString(`<ol>`)
			e.writeCommentTemplateItems(b, kids, byParent, depth+1, threadEven)
			b.WriteString(`</ol>`)
		}
		b.WriteString(`</li>`)
	}
}

func (e *Env) renderCommentTemplateItemBody(cm CommentItem) string {
	e = e.ensure()
	var b strings.Builder
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
	if cm.Date != "" && e.currentPost != nil {
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
	if e.currentPost != nil {
		href := e.currentPost.Link + "?replytocom=" + strconv.FormatInt(cm.ID, 10) + "#respond"
		b.WriteString(`<a aria-label="Reply to ` + html.EscapeString(cm.Author) + `" class="comment-reply-link"`)
		b.WriteString(` data-belowelement="comment-` + strconv.FormatInt(cm.ID, 10) + `"`)
		b.WriteString(` data-commentid="` + strconv.FormatInt(cm.ID, 10) + `"`)
		b.WriteString(` data-postid="` + strconv.FormatInt(e.currentPost.ID, 10) + `"`)
		b.WriteString(` data-replyto="Reply to ` + html.EscapeString(cm.Author) + `"`)
		b.WriteString(` data-respondelement="respond"`)
		b.WriteString(` href="` + html.EscapeString(href) + `" rel="nofollow">Reply</a>`)
	}
	b.WriteString(`</div></div>`)
	b.WriteString(`</div></div></div>`)
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
