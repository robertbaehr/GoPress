package blocks

import (
	"html"
	"net/url"
	"strconv"
	"strings"
)

// renderRSS renders core/rss (P5.79).
//
// Remote HTTP fetch is blocked (SSRF / no SimplePie). Same-site feed URLs
// (/feed/, /feed/rss2/, ?feed=…) render a link list from ListPosts. Pointing
// the block at the site home/site URL shows the WP loop-prevention notice.
func (e *Env) renderRSS(block Block) string {
	e = e.ensure()
	feedURL := strings.TrimSpace(attrString(block.Attrs, "feedURL"))
	if feedURL == "" {
		return ""
	}

	if rssFeedIsSiteRoot(feedURL, e) {
		return `<div class="components-placeholder"><div class="notice notice-error">` +
			`Adding an RSS feed to this site’s homepage is not supported, as it could lead to a loop that slows down your site. Try using another block, like the <strong>Latest Posts</strong> block, to list posts from the site.` +
			`</div></div>`
	}

	if !rssFeedIsLocal(feedURL, e) {
		return `<div class="components-placeholder"><div class="notice notice-error"><strong>RSS Error:</strong> ` +
			html.EscapeString("Remote feeds are blocked.") +
			`</div></div>`
	}

	n := 5
	if v, ok := attrInt(block.Attrs, "itemsToShow"); ok && v > 0 {
		n = v
	}
	displayDate := false
	if v, ok := block.Attrs["displayDate"].(bool); ok {
		displayDate = v
	}
	displayAuthor := false
	if v, ok := block.Attrs["displayAuthor"].(bool); ok {
		displayAuthor = v
	}
	displayExcerpt := false
	if v, ok := block.Attrs["displayExcerpt"].(bool); ok {
		displayExcerpt = v
	}
	openInNewTab := false
	if v, ok := block.Attrs["openInNewTab"].(bool); ok {
		openInNewTab = v
	}
	rel := strings.TrimSpace(attrString(block.Attrs, "rel"))
	layout := attrString(block.Attrs, "blockLayout")
	if layout == "" {
		layout = "list"
	}
	columns := 2
	if v, ok := attrInt(block.Attrs, "columns"); ok && v > 0 {
		columns = v
	}

	var posts []QueryPost
	if e.ListPosts != nil {
		posts = e.ListPosts(QueryLoopArgs{PerPage: n, PostType: "post"})
	}
	if len(posts) == 0 {
		return `<div class="components-placeholder"><div class="notice notice-error">` +
			`An error has occurred, which probably means the feed is down. Try again later.` +
			`</div></div>`
	}

	linkAttrs := ""
	if openInNewTab {
		linkAttrs += ` target="_blank"`
	}
	if rel != "" {
		linkAttrs += ` rel="` + html.EscapeString(rel) + `"`
	}

	var items strings.Builder
	for _, p := range posts {
		title := strings.TrimSpace(p.Title)
		if title == "" {
			title = "(no title)"
		}
		titleHTML := html.EscapeString(title)
		if p.Link != "" {
			titleHTML = `<a href='` + html.EscapeString(p.Link) + `'` + linkAttrs + `>` + titleHTML + `</a>`
		}
		items.WriteString(`<li class='wp-block-rss__item'>`)
		items.WriteString(`<div class='wp-block-rss__item-title'>`)
		items.WriteString(titleHTML)
		items.WriteString(`</div>`)
		if displayDate && p.Date != "" {
			items.WriteString(`<time datetime="`)
			items.WriteString(html.EscapeString(p.Date))
			items.WriteString(`" class="wp-block-rss__item-publish-date">`)
			items.WriteString(html.EscapeString(formatDisplayDate(p.Date)))
			items.WriteString(`</time> `)
		}
		if displayAuthor && e.AuthorName != nil {
			if name := strings.TrimSpace(e.AuthorName(p.ID)); name != "" {
				items.WriteString(`<span class="wp-block-rss__item-author">by `)
				items.WriteString(html.EscapeString(name))
				items.WriteString(`</span>`)
			}
		}
		if displayExcerpt {
			excerpt := strings.TrimSpace(p.Excerpt)
			if excerpt == "" {
				excerpt = strings.TrimSpace(stripTags(p.Content))
			}
			if excerpt != "" {
				items.WriteString(`<div class="wp-block-rss__item-excerpt">`)
				items.WriteString(html.EscapeString(excerpt))
				items.WriteString(`</div>`)
			}
		}
		items.WriteString(`</li>`)
	}

	var classes []string
	classes = append(classes, "wp-block-rss")
	if layout == "grid" {
		classes = append(classes, "is-grid", "columns-"+strconv.Itoa(columns))
	}
	if displayDate {
		classes = append(classes, "has-dates")
	}
	if displayAuthor {
		classes = append(classes, "has-authors")
	}
	if displayExcerpt {
		classes = append(classes, "has-excerpts")
	}

	return `<ul class="` + strings.Join(classes, " ") + `">` + items.String() + `</ul>`
}

func rssFeedIsSiteRoot(feedURL string, e *Env) bool {
	trimmed := strings.TrimRight(strings.TrimSpace(feedURL), "/")
	if trimmed == "" {
		return false
	}
	for _, base := range rssSiteBases(e) {
		if strings.EqualFold(trimmed, strings.TrimRight(base, "/")) {
			return true
		}
	}
	return false
}

func rssFeedIsLocal(feedURL string, e *Env) bool {
	u, err := url.Parse(strings.TrimSpace(feedURL))
	if err != nil || u == nil {
		return false
	}
	// Relative same-origin feed paths.
	if u.Host == "" {
		path := strings.ToLower(u.Path)
		if strings.Contains(path, "/feed") || u.Query().Get("feed") != "" {
			return true
		}
		return false
	}
	host := strings.ToLower(u.Hostname())
	local := false
	for _, base := range rssSiteBases(e) {
		if bu, err := url.Parse(base); err == nil && bu != nil && bu.Hostname() != "" {
			if strings.EqualFold(host, bu.Hostname()) {
				local = true
				break
			}
		}
	}
	if !local {
		return false
	}
	path := strings.ToLower(u.Path)
	if strings.Contains(path, "/feed") || u.Query().Get("feed") != "" {
		return true
	}
	// Bare site root already handled as loop; other same-host non-feed URLs are not local feeds.
	return false
}

func rssSiteBases(e *Env) []string {
	var out []string
	if e != nil && e.HomeURL != nil {
		if s := strings.TrimSpace(e.HomeURL()); s != "" {
			out = append(out, s)
		}
	}
	return out
}
