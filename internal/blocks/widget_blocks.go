package blocks

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// ArchiveItem is a monthly (or yearly) archive link for core/archives (P5.42).
type ArchiveItem struct {
	Label string
	URL   string
	Count int
}

// TagCloudItem is a tag for core/tag-cloud (P5.42).
type TagCloudItem struct {
	Name  string
	URL   string
	Count int
}

// LatestCommentItem is a site-wide comment for core/latest-comments (P5.42 / P5.123).
type LatestCommentItem struct {
	ID          int64
	Author      string
	AuthorURL   string
	AuthorEmail string // for displayAvatar gravatar (P5.123)
	Content     string
	Date        string
	PostID      int64
	PostTitle   string
	PostURL     string
}

var searchInputSeq atomic.Uint64

func nextSearchInputID() string {
	n := searchInputSeq.Add(1)
	return fmt.Sprintf("wp-block-search__input-%d", n)
}

// searchIconSVG is the core/search buttonUseIcon glyph (P5.124).
const searchIconSVG = `<svg class="search-icon" viewBox="0 0 24 24" width="24" height="24">
					<path d="M13 5c-3.3 0-6 2.7-6 6 0 1.4.5 2.7 1.3 3.7l-3.8 3.8 1.1 1.1 3.8-3.8c1 .8 2.3 1.3 3.7 1.3 3.3 0 6-2.7 6-6S16.3 5 13 5zm0 10.5c-2.5 0-4.5-2-4.5-4.5s2-4.5 4.5-4.5 4.5 2 4.5 4.5-2 4.5-4.5 4.5z"></path>
				</svg>`

func (e *Env) renderSearch(block Block) string {
	e = e.ensure()
	label := attrString(block.Attrs, "label")
	if label == "" {
		label = "Search"
	}
	buttonText := attrString(block.Attrs, "buttonText")
	if buttonText == "" {
		buttonText = "Search"
	}
	placeholder := attrString(block.Attrs, "placeholder")
	showLabel := true
	if v, ok := block.Attrs["showLabel"].(bool); ok {
		showLabel = v
	}
	buttonPos := attrString(block.Attrs, "buttonPosition")
	if buttonPos == "" {
		buttonPos = "button-outside"
	}
	showButton := buttonPos != "no-button"
	useIcon := false
	if v, ok := block.Attrs["buttonUseIcon"].(bool); ok {
		useIcon = v
	}
	home := "/"
	if e.HomeURL != nil {
		if h := strings.TrimSpace(e.HomeURL()); h != "" {
			home = h
			if !strings.HasSuffix(home, "/") {
				home += "/"
			}
		}
	}
	value := strings.TrimSpace(e.SearchQuery)
	inputID := nextSearchInputID()

	// P5.124: classnames_for_block_core_search (+ wp-block-search root).
	var classParts []string
	switch buttonPos {
	case "button-inside":
		classParts = append(classParts, "wp-block-search__button-inside")
	case "button-only":
		classParts = append(classParts, "wp-block-search__button-only", "wp-block-search__searchfield-hidden")
	case "no-button":
		classParts = append(classParts, "wp-block-search__no-button")
	default:
		classParts = append(classParts, "wp-block-search__button-outside")
	}
	if showButton {
		if useIcon {
			classParts = append(classParts, "wp-block-search__icon-button")
		} else {
			classParts = append(classParts, "wp-block-search__text-button")
		}
	}
	classParts = append(classParts, "wp-block-search")
	class := strings.Join(classParts, " ")

	// P5.124: width/widthUnit → inline style on inside-wrapper.
	wrapperStyle := ""
	if w, ok := attrInt(block.Attrs, "width"); ok && w > 0 {
		unit := strings.TrimSpace(attrString(block.Attrs, "widthUnit"))
		if unit == "" {
			unit = "px"
		}
		wrapperStyle = fmt.Sprintf(` style="width: %d%s;"`, w, html.EscapeString(unit))
	}

	var field strings.Builder
	labelClass := "wp-block-search__label"
	if !showLabel {
		labelClass += " screen-reader-text"
	}
	field.WriteString(`<label class="`)
	field.WriteString(labelClass)
	field.WriteString(`" for="`)
	field.WriteString(html.EscapeString(inputID))
	field.WriteString(`" >`)
	field.WriteString(html.EscapeString(label))
	field.WriteString(`</label>`)
	field.WriteString(`<div class="wp-block-search__inside-wrapper"`)
	field.WriteString(wrapperStyle)
	field.WriteString(` >`)
	field.WriteString(`<input class="wp-block-search__input" id="`)
	field.WriteString(html.EscapeString(inputID))
	field.WriteString(`" placeholder="`)
	field.WriteString(html.EscapeString(placeholder))
	field.WriteString(`" value="`)
	field.WriteString(html.EscapeString(value))
	field.WriteString(`" type="search" name="s" required />`)
	// P5.124: attributes.query → hidden inputs (post_type etc.).
	field.WriteString(searchQueryHiddenInputs(block.Attrs))
	if showButton {
		btnClass := "wp-block-search__button wp-element-button"
		btnInner := html.EscapeString(buttonText)
		ariaLabel := buttonText
		btnType := "submit"
		extraAttrs := ""
		if useIcon {
			btnClass += " has-icon"
			btnInner = searchIconSVG
		}
		if buttonPos == "button-only" {
			// Expandable searchfield SSR defaults (interactivity directives omitted).
			ariaLabel = "Expand search field"
			btnType = "button"
			extraAttrs = ` aria-controls="` + html.EscapeString(inputID) + `" aria-expanded="false"`
		}
		field.WriteString(`<button aria-label="`)
		field.WriteString(html.EscapeString(ariaLabel))
		field.WriteString(`" class="`)
		field.WriteString(html.EscapeString(btnClass))
		field.WriteString(`" type="`)
		field.WriteString(btnType)
		field.WriteString(`"`)
		field.WriteString(extraAttrs)
		field.WriteString(` >`)
		field.WriteString(btnInner)
		field.WriteString(`</button>`)
	}
	field.WriteString(`</div>`)

	// P5.124: tagName / html5 search-element opt-in (ThemeSupports).
	tagName := strings.TrimSpace(attrString(block.Attrs, "tagName"))
	useSearchEl := tagName == "search"
	if tagName == "" && e.ThemeSupports != nil && e.ThemeSupports("html5", "search-element") {
		useSearchEl = true
	}
	var b strings.Builder
	if useSearchEl && tagName != "form" {
		b.WriteString(`<search class="`)
		b.WriteString(html.EscapeString(class))
		b.WriteString(`" >`)
		b.WriteString(`<form method="get" action="`)
		b.WriteString(html.EscapeString(home))
		b.WriteString(`">`)
		b.WriteString(field.String())
		b.WriteString(`</form></search>`)
	} else {
		b.WriteString(`<form role="search" method="get" action="`)
		b.WriteString(html.EscapeString(home))
		b.WriteString(`" class="`)
		b.WriteString(html.EscapeString(class))
		b.WriteString(`" >`)
		b.WriteString(field.String())
		b.WriteString(`</form>`)
	}
	return b.String()
}

// searchQueryHiddenInputs emits <input type="hidden"> for attributes.query (P5.124).
func searchQueryHiddenInputs(attrs map[string]any) string {
	if attrs == nil {
		return ""
	}
	raw, ok := attrs["query"]
	if !ok || raw == nil {
		return ""
	}
	m, ok := raw.(map[string]any)
	if !ok || len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Stable order for tests / diffs.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	var b strings.Builder
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		b.WriteString(`<input type="hidden" name="`)
		b.WriteString(html.EscapeString(k))
		b.WriteString(`" value="`)
		b.WriteString(html.EscapeString(fmt.Sprint(m[k])))
		b.WriteString(`" />`)
	}
	return b.String()
}

func (e *Env) renderCalendar(block Block) string {
	e = e.ensure()
	// P5.122: hide calendar when no published posts (block_core_calendar_has_published_posts).
	if e.HasPublishedPosts != nil && !e.HasPublishedPosts() {
		if e.IsLoggedIn != nil && e.IsLoggedIn() {
			return `<div>The calendar block is hidden because there are no published posts.</div>`
		}
		return ""
	}
	now := time.Now()
	year, month := now.Year(), now.Month()
	if m, ok := attrInt(block.Attrs, "month"); ok && m >= 1 && m <= 12 {
		month = time.Month(m)
	}
	if y, ok := attrInt(block.Attrs, "year"); ok && y >= 1970 {
		year = y
	}
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	daysInMonth := first.AddDate(0, 1, -1).Day()
	// WordPress calendar uses Sunday=0.
	startPad := int(first.Weekday())

	home := ""
	if e.HomeURL != nil {
		home = strings.TrimRight(e.HomeURL(), "/")
	}

	// P5.122: wrapper + table color classes (get_block_wrapper_attributes subset).
	wrapClasses := []string{"wp-block-calendar"}
	tableClasses := []string{"wp-calendar-table"}
	if hasLinkColorAttr(block.Attrs) {
		tableClasses = append(tableClasses, "has-link-color")
	}
	if bg := strings.TrimSpace(attrString(block.Attrs, "backgroundColor")); bg != "" {
		tableClasses = append(tableClasses, "has-background", "has-"+bg+"-background-color")
	}
	if tc := strings.TrimSpace(attrString(block.Attrs, "textColor")); tc != "" {
		tableClasses = append(tableClasses, "has-text-color", "has-"+tc+"-color")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		wrapClasses = append(wrapClasses, cn)
	}

	var b strings.Builder
	b.WriteString(`<div class="`)
	b.WriteString(html.EscapeString(strings.Join(wrapClasses, " ")))
	b.WriteString(`"`)
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(`><div class="wp-calendar-table"><table class="`)
	b.WriteString(html.EscapeString(strings.Join(tableClasses, " ")))
	b.WriteString(`"><caption>`)
	b.WriteString(html.EscapeString(first.Format("January 2006")))
	b.WriteString(`</caption><thead><tr>`)
	for _, d := range []string{"S", "M", "T", "W", "T", "F", "S"} {
		b.WriteString(`<th scope="col" title="`)
		b.WriteString(d)
		b.WriteString(`">`)
		b.WriteString(d)
		b.WriteString(`</th>`)
	}
	b.WriteString(`</tr></thead><tbody><tr>`)
	cell := 0
	for i := 0; i < startPad; i++ {
		b.WriteString(`<td colspan="1" class="pad">&nbsp;</td>`)
		cell++
	}
	for day := 1; day <= daysInMonth; day++ {
		if cell == 7 {
			b.WriteString(`</tr><tr>`)
			cell = 0
		}
		b.WriteString(`<td>`)
		if home != "" {
			href := fmt.Sprintf("%s/%04d/%02d/%02d/", home, year, int(month), day)
			b.WriteString(`<a href="`)
			b.WriteString(html.EscapeString(href))
			b.WriteString(`">`)
			b.WriteString(strconv.Itoa(day))
			b.WriteString(`</a>`)
		} else {
			b.WriteString(strconv.Itoa(day))
		}
		b.WriteString(`</td>`)
		cell++
	}
	for cell > 0 && cell < 7 {
		b.WriteString(`<td class="pad">&nbsp;</td>`)
		cell++
	}
	b.WriteString(`</tr></tbody></table></div></div>`)
	return b.String()
}

func (e *Env) renderArchives(block Block) string {
	e = e.ensure()
	showCounts := false
	if v, ok := block.Attrs["showPostCounts"].(bool); ok {
		showCounts = v
	}
	asDropdown := false
	if v, ok := block.Attrs["displayAsDropdown"].(bool); ok {
		asDropdown = v
	}
	// P5.122: type + showLabel + unique dropdown id + onChange script.
	archiveType := strings.TrimSpace(attrString(block.Attrs, "type"))
	if archiveType == "" {
		archiveType = "monthly"
	}
	showLabel := true
	if v, ok := block.Attrs["showLabel"].(bool); ok {
		showLabel = v
	}
	var items []ArchiveItem
	if e.ListArchives != nil {
		items = e.ListArchives(archiveType)
	}
	wrapExtra := archivesWrapperClasses(block.Attrs)
	if asDropdown {
		dropdownID := nextArchivesDropdownID()
		labelText := archivesDropdownLabel(archiveType)
		var b strings.Builder
		b.WriteString(`<div class="`)
		b.WriteString(html.EscapeString(strings.Join(append([]string{"wp-block-archives", "wp-block-archives-dropdown"}, wrapExtra...), " ")))
		b.WriteString(`"`)
		if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
			b.WriteString(` id="`)
			b.WriteString(html.EscapeString(anchor))
			b.WriteString(`"`)
		}
		b.WriteString(`>`)
		labelClass := "wp-block-archives__label"
		if !showLabel {
			labelClass += " screen-reader-text"
		}
		b.WriteString(`<label for="`)
		b.WriteString(html.EscapeString(dropdownID))
		b.WriteString(`" class="`)
		b.WriteString(labelClass)
		b.WriteString(`">Archives</label>`)
		b.WriteString(`<select id="`)
		b.WriteString(html.EscapeString(dropdownID))
		b.WriteString(`" name="archive-dropdown"><option value="">`)
		b.WriteString(html.EscapeString(labelText))
		b.WriteString(`</option>`)
		for _, it := range items {
			b.WriteString(`<option value="`)
			b.WriteString(html.EscapeString(it.URL))
			b.WriteString(`">`)
			b.WriteString(html.EscapeString(it.Label))
			if showCounts {
				b.WriteString(fmt.Sprintf(" (%d)", it.Count))
			}
			b.WriteString(`</option>`)
		}
		b.WriteString(`</select>`)
		b.WriteString(archivesDropdownScript(dropdownID, e))
		b.WriteString(`</div>`)
		return b.String()
	}
	if len(items) == 0 {
		var b strings.Builder
		b.WriteString(`<div class="`)
		b.WriteString(html.EscapeString(strings.Join(append([]string{"wp-block-archives", "wp-block-archives-list"}, wrapExtra...), " ")))
		b.WriteString(`">No archives to show.</div>`)
		return b.String()
	}
	var b strings.Builder
	b.WriteString(`<ul class="`)
	b.WriteString(html.EscapeString(strings.Join(append([]string{"wp-block-archives", "wp-block-archives-list"}, wrapExtra...), " ")))
	b.WriteString(`"`)
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	for _, it := range items {
		b.WriteString(`<li><a href="`)
		b.WriteString(html.EscapeString(it.URL))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(it.Label))
		b.WriteString(`</a>`)
		if showCounts {
			b.WriteString(fmt.Sprintf(" (%d)", it.Count))
		}
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
}

var archivesDropdownSeq atomic.Uint64

func nextArchivesDropdownID() string {
	n := archivesDropdownSeq.Add(1)
	return fmt.Sprintf("wp-block-archives-%d", n)
}

func archivesDropdownLabel(archiveType string) string {
	switch archiveType {
	case "yearly":
		return "Select Year"
	case "daily":
		return "Select Day"
	case "weekly":
		return "Select Week"
	case "postbypost", "alpha":
		return "Select Post"
	default:
		return "Select Month"
	}
}

func archivesWrapperClasses(attrs map[string]any) []string {
	var out []string
	if hasLinkColorAttr(attrs) {
		out = append(out, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		out = append(out, cn)
	}
	return out
}

// archivesDropdownScript mirrors block_core_archives_build_dropdown_script (P5.122).
func archivesDropdownScript(dropdownID string, e *Env) string {
	home := "/"
	if e != nil && e.HomeURL != nil {
		if h := strings.TrimSpace(e.HomeURL()); h != "" {
			home = h
		}
	}
	var b strings.Builder
	b.WriteString(`<script>`)
	b.WriteString(`( ( [ dropdownId, homeUrl ] ) => {`)
	b.WriteString(`const dropdown = document.getElementById( dropdownId );`)
	b.WriteString(`function onSelectChange() { setTimeout( () => {`)
	b.WriteString(`if ( 'escape' === dropdown.dataset.lastkey ) { return; }`)
	b.WriteString(`if ( dropdown.value ) { location.href = dropdown.value; }`)
	b.WriteString(`}, 250 ); }`)
	b.WriteString(`function onKeyUp( event ) {`)
	b.WriteString(`if ( 'Escape' === event.key ) { dropdown.dataset.lastkey = 'escape'; }`)
	b.WriteString(`else { delete dropdown.dataset.lastkey; } }`)
	b.WriteString(`function onClick() { delete dropdown.dataset.lastkey; }`)
	b.WriteString(`dropdown.addEventListener( 'keyup', onKeyUp );`)
	b.WriteString(`dropdown.addEventListener( 'click', onClick );`)
	b.WriteString(`dropdown.addEventListener( 'change', onSelectChange );`)
	b.WriteString(`} )( `)
	b.WriteString(fmt.Sprintf(`[%q,%q]`, dropdownID, home))
	b.WriteString(` );</script>`)
	return b.String()
}

func (e *Env) renderLatestPosts(block Block) string {
	e = e.ensure()
	n := 5
	if v, ok := attrInt(block.Attrs, "postsToShow"); ok && v > 0 {
		n = v
	}
	displayDate := false
	if v, ok := block.Attrs["displayPostDate"].(bool); ok {
		displayDate = v
	}
	displayAuthor := false
	if v, ok := block.Attrs["displayAuthor"].(bool); ok {
		displayAuthor = v
	}
	displayContent := false
	if v, ok := block.Attrs["displayPostContent"].(bool); ok {
		displayContent = v
	}
	contentRadio := strings.TrimSpace(attrString(block.Attrs, "displayPostContentRadio"))
	if contentRadio == "" {
		contentRadio = "excerpt"
	}
	excerptLen := 55
	if v, ok := attrInt(block.Attrs, "excerptLength"); ok && v > 0 {
		excerptLen = v
	}
	order := strings.ToLower(strings.TrimSpace(attrString(block.Attrs, "order")))
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	orderBy := strings.TrimSpace(attrString(block.Attrs, "orderBy"))
	if orderBy == "" {
		orderBy = "date"
	}
	var authorID int64
	if v, ok := attrInt(block.Attrs, "selectedAuthor"); ok && v > 0 {
		authorID = int64(v)
	}
	cats := latestPostsCategoryIDs(block.Attrs)
	args := QueryLoopArgs{
		PerPage:    n,
		PostType:   "post",
		Order:      order,
		OrderBy:    orderBy,
		Author:     authorID,
		CategoryIn: cats,
		Sticky:     "ignore",
	}
	var posts []QueryPost
	if e.ListPosts != nil {
		posts = e.ListPosts(args)
	}
	// P5.123: grid layout + has-dates/has-author + wrapper supports.
	classes := []string{"wp-block-latest-posts__list"}
	layoutType := "default"
	if strings.EqualFold(strings.TrimSpace(attrString(block.Attrs, "postLayout")), "grid") {
		layoutType = "grid"
	}
	if layout, ok := block.Attrs["layout"].(map[string]any); ok {
		if t := strings.TrimSpace(fmt.Sprint(layout["type"])); strings.EqualFold(t, "grid") {
			layoutType = "grid"
		}
	}
	if layoutType == "grid" {
		classes = append(classes, "is-grid")
		cols := 0
		if v, ok := attrInt(block.Attrs, "columns"); ok && v > 0 {
			cols = v
		}
		if layout, ok := block.Attrs["layout"].(map[string]any); ok {
			if v, ok := attrInt(layout, "columnCount"); ok && v > 0 {
				cols = v
			}
		}
		if cols > 0 {
			classes = append(classes, fmt.Sprintf("columns-%d", cols))
		}
	}
	if displayDate {
		classes = append(classes, "has-dates")
	}
	if displayAuthor {
		classes = append(classes, "has-author")
	}
	var b strings.Builder
	b.WriteString(`<ul class="wp-block-latest-posts `)
	b.WriteString(html.EscapeString(strings.Join(classes, " ")))
	b.WriteString(`">`)
	for _, p := range posts {
		title := strings.TrimSpace(p.Title)
		if title == "" {
			title = "(no title)"
		}
		b.WriteString(`<li>`)
		b.WriteString(`<a class="wp-block-latest-posts__post-title" href="`)
		b.WriteString(html.EscapeString(p.Link))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(title))
		b.WriteString(`</a>`)
		if displayAuthor && e.AuthorName != nil {
			if name := strings.TrimSpace(e.AuthorName(p.ID)); name != "" {
				b.WriteString(`<div class="wp-block-latest-posts__post-author">by `)
				b.WriteString(html.EscapeString(name))
				b.WriteString(`</div>`)
			}
		}
		if displayDate && p.Date != "" {
			b.WriteString(`<time datetime="`)
			b.WriteString(html.EscapeString(p.Date))
			b.WriteString(`" class="wp-block-latest-posts__post-date">`)
			b.WriteString(html.EscapeString(formatDisplayDate(p.Date)))
			b.WriteString(`</time>`)
		}
		if displayContent {
			switch contentRadio {
			case "full_post":
				content := p.Content
				if p.Password != "" {
					content = "This content is password protected."
				}
				b.WriteString(`<div class="wp-block-latest-posts__post-full-content">`)
				b.WriteString(content)
				b.WriteString(`</div>`)
			default: // excerpt
				ex := strings.TrimSpace(p.Excerpt)
				if ex == "" {
					ex = strings.TrimSpace(p.Content)
				}
				if p.Password != "" {
					ex = "This content is password protected."
				} else {
					ex = trimExcerptWords(stripTagsForExcerpt(ex), excerptLen)
				}
				b.WriteString(`<div class="wp-block-latest-posts__post-excerpt">`)
				b.WriteString(html.EscapeString(ex))
				b.WriteString(`</div>`)
			}
		}
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ul>`)
	return applySupports(b.String(), block.Attrs, "core/latest-posts")
}

func latestPostsCategoryIDs(attrs map[string]any) []int64 {
	if attrs == nil {
		return nil
	}
	raw, ok := attrs["categories"]
	if !ok || raw == nil {
		return nil
	}
	var out []int64
	switch v := raw.(type) {
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && n > 0 {
			out = append(out, n)
		}
	case float64:
		if int64(v) > 0 {
			out = append(out, int64(v))
		}
	case []any:
		for _, item := range v {
			switch t := item.(type) {
			case map[string]any:
				if id, ok := attrInt(t, "id"); ok && id > 0 {
					out = append(out, int64(id))
				}
			case float64:
				if int64(t) > 0 {
					out = append(out, int64(t))
				}
			case string:
				if n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil && n > 0 {
					out = append(out, n)
				}
			}
		}
	}
	return out
}

func stripTagsForExcerpt(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func (e *Env) renderLatestComments(block Block) string {
	e = e.ensure()
	n := 5
	if v, ok := attrInt(block.Attrs, "commentsToShow"); ok && v > 0 {
		n = v
	}
	displayDate := true
	if v, ok := block.Attrs["displayDate"].(bool); ok {
		displayDate = v
	}
	displayAvatar := true
	if v, ok := block.Attrs["displayAvatar"].(bool); ok {
		displayAvatar = v
	}
	displayExcerpt := true
	if v, ok := block.Attrs["displayExcerpt"].(bool); ok {
		displayExcerpt = v
	}
	displayContent := attrString(block.Attrs, "displayContent")
	if displayContent == "" {
		if displayExcerpt {
			displayContent = "excerpt"
		} else {
			displayContent = "none"
		}
	}
	var items []LatestCommentItem
	if e.ListLatestComments != nil {
		items = e.ListLatestComments(n)
	}
	classes := []string{"wp-block-latest-comments"}
	if displayAvatar {
		classes = append(classes, "has-avatars")
	}
	if displayDate {
		classes = append(classes, "has-dates")
	}
	if displayContent != "none" {
		classes = append(classes, "has-excerpts")
	}
	if len(items) == 0 {
		classes = append(classes, "no-comments")
		empty := fmt.Sprintf(`<div class="%s">No comments to show.</div>`, html.EscapeString(strings.Join(classes, " ")))
		return applySupports(empty, block.Attrs, "core/latest-comments")
	}
	var b strings.Builder
	b.WriteString(`<ol class="`)
	b.WriteString(html.EscapeString(strings.Join(classes, " ")))
	b.WriteString(`">`)
	for _, cm := range items {
		b.WriteString(`<li class="wp-block-latest-comments__comment">`)
		if displayAvatar {
			email := strings.TrimSpace(cm.AuthorEmail)
			hash := gravatarHash(email)
			src := fmt.Sprintf("https://secure.gravatar.com/avatar/%s?s=48&d=mm&r=g", hash)
			b.WriteString(`<img alt="" class="avatar avatar-48 photo wp-block-latest-comments__comment-avatar" height="48" src="`)
			b.WriteString(html.EscapeString(src))
			b.WriteString(`" width="48"/>`)
		}
		b.WriteString(`<article>`)
		b.WriteString(`<footer class="wp-block-latest-comments__comment-meta">`)
		author := html.EscapeString(cm.Author)
		if cm.AuthorURL != "" {
			b.WriteString(`<a class="wp-block-latest-comments__comment-author" href="`)
			b.WriteString(html.EscapeString(cm.AuthorURL))
			b.WriteString(`">`)
			b.WriteString(author)
			b.WriteString(`</a>`)
		} else {
			b.WriteString(`<span class="wp-block-latest-comments__comment-author">`)
			b.WriteString(author)
			b.WriteString(`</span>`)
		}
		b.WriteString(` on `)
		postTitle := strings.TrimSpace(cm.PostTitle)
		if postTitle == "" {
			postTitle = "(no title)"
		}
		b.WriteString(`<a class="wp-block-latest-comments__comment-link" href="`)
		b.WriteString(html.EscapeString(cm.PostURL))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(postTitle))
		b.WriteString(`</a>`)
		if displayDate && cm.Date != "" {
			b.WriteString(`<time datetime="`)
			b.WriteString(html.EscapeString(cm.Date))
			b.WriteString(`" class="wp-block-latest-comments__comment-date">`)
			b.WriteString(html.EscapeString(formatDisplayDate(cm.Date)))
			b.WriteString(`</time>`)
		}
		b.WriteString(`</footer>`)
		if displayContent != "none" && cm.Content != "" {
			excerpt := cm.Content
			if displayContent == "excerpt" && len(excerpt) > 100 {
				excerpt = excerpt[:100] + "…"
			}
			b.WriteString(`<div class="wp-block-latest-comments__comment-excerpt"><p>`)
			b.WriteString(html.EscapeString(excerpt))
			b.WriteString(`</p></div>`)
		}
		b.WriteString(`</article></li>`)
	}
	b.WriteString(`</ol>`)
	return applySupports(b.String(), block.Attrs, "core/latest-comments")
}

func (e *Env) renderTagCloud(block Block) string {
	e = e.ensure()
	showCounts := false
	if v, ok := block.Attrs["showTagCounts"].(bool); ok {
		showCounts = v
	}
	// P5.82: taxonomy + numberOfTags (wp_tag_cloud args from block attrs).
	taxonomy := strings.TrimSpace(attrString(block.Attrs, "taxonomy"))
	if taxonomy == "" {
		taxonomy = "post_tag"
	}
	number := 45
	if v, ok := attrInt(block.Attrs, "numberOfTags"); ok && v > 0 {
		number = v
	}
	var items []TagCloudItem
	if e.ListTagCloud != nil {
		items = e.ListTagCloud(taxonomy, number)
	}
	if len(items) == 0 {
		return ""
	}
	if len(items) > number {
		items = items[:number]
	}
	var b strings.Builder
	b.WriteString(`<p class="wp-block-tag-cloud">`)
	for i, it := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(`<a href="`)
		b.WriteString(html.EscapeString(it.URL))
		b.WriteString(`" class="tag-cloud-link">`)
		b.WriteString(html.EscapeString(it.Name))
		if showCounts {
			b.WriteString(fmt.Sprintf(" (%d)", it.Count))
		}
		b.WriteString(`</a>`)
	}
	b.WriteString(`</p>`)
	return b.String()
}

func attrString(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	if s, ok := attrs[key].(string); ok {
		return s
	}
	return ""
}

func attrInt(attrs map[string]any, key string) (int, bool) {
	if attrs == nil {
		return 0, false
	}
	switch v := attrs[key].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func formatDisplayDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("January 2, 2006")
		}
	}
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}
