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

// LatestCommentItem is a site-wide comment for core/latest-comments (P5.42).
type LatestCommentItem struct {
	ID        int64
	Author    string
	AuthorURL string
	Content   string
	Date      string
	PostID    int64
	PostTitle string
	PostURL   string
}

var searchInputSeq atomic.Uint64

func nextSearchInputID() string {
	n := searchInputSeq.Add(1)
	return fmt.Sprintf("wp-block-search__input-%d", n)
}

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

	class := "wp-block-search__button-outside wp-block-search__text-button wp-block-search"
	switch buttonPos {
	case "button-inside":
		class = "wp-block-search__button-inside wp-block-search__text-button wp-block-search"
	case "button-only":
		class = "wp-block-search__button-only wp-block-search__text-button wp-block-search"
	case "no-button":
		class = "wp-block-search__no-button wp-block-search"
	}

	var b strings.Builder
	b.WriteString(`<form role="search" method="get" action="`)
	b.WriteString(html.EscapeString(home))
	b.WriteString(`" class="`)
	b.WriteString(html.EscapeString(class))
	b.WriteString(`" >`)
	labelClass := "wp-block-search__label"
	if !showLabel {
		labelClass += " screen-reader-text"
	}
	b.WriteString(`<label class="`)
	b.WriteString(labelClass)
	b.WriteString(`" for="`)
	b.WriteString(html.EscapeString(inputID))
	b.WriteString(`" >`)
	b.WriteString(html.EscapeString(label))
	b.WriteString(`</label>`)
	b.WriteString(`<div class="wp-block-search__inside-wrapper" >`)
	b.WriteString(`<input class="wp-block-search__input" id="`)
	b.WriteString(html.EscapeString(inputID))
	b.WriteString(`" placeholder="`)
	b.WriteString(html.EscapeString(placeholder))
	b.WriteString(`" value="`)
	b.WriteString(html.EscapeString(value))
	b.WriteString(`" type="search" name="s" required />`)
	if showButton {
		b.WriteString(`<button aria-label="`)
		b.WriteString(html.EscapeString(buttonText))
		b.WriteString(`" class="wp-block-search__button wp-element-button" type="submit" >`)
		b.WriteString(html.EscapeString(buttonText))
		b.WriteString(`</button>`)
	}
	b.WriteString(`</div></form>`)
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
	var posts []QueryPost
	if e.ListPosts != nil {
		posts = e.ListPosts(QueryLoopArgs{PerPage: n, PostType: "post"})
	}
	var b strings.Builder
	b.WriteString(`<ul class="wp-block-latest-posts wp-block-latest-posts__list">`)
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
		if displayDate && p.Date != "" {
			b.WriteString(`<time datetime="`)
			b.WriteString(html.EscapeString(p.Date))
			b.WriteString(`" class="wp-block-latest-posts__post-date">`)
			b.WriteString(html.EscapeString(formatDisplayDate(p.Date)))
			b.WriteString(`</time>`)
		}
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
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
	if displayDate {
		classes = append(classes, "has-dates")
	}
	if displayContent != "none" {
		classes = append(classes, "has-excerpts")
	}
	if len(items) == 0 {
		classes = append(classes, "no-comments")
		return fmt.Sprintf(`<div class="%s">No comments to show.</div>`, strings.Join(classes, " "))
	}
	var b strings.Builder
	b.WriteString(`<ol class="`)
	b.WriteString(strings.Join(classes, " "))
	b.WriteString(`">`)
	for _, cm := range items {
		b.WriteString(`<li class="wp-block-latest-comments__comment"><article>`)
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
	return b.String()
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
