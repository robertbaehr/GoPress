package server

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/blocks"
	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// Front is the minimal front controller / shim renderer (P3.3+).
// Theme PHP is out of scope; this emits small HTML documents for home and singular.
type Front struct{}

// ServeFront implements FrontHandler.
func (f *Front) ServeFront(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil {
		http.Error(w, "GoPress context unavailable\n", http.StatusServiceUnavailable)
		return
	}
	c.AuthenticateRequest()

	rules, err := c.LoadRewriteRules()
	if err != nil {
		http.Error(w, "rewrite rules: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	parsed := c.ParseRequest(rules)

	// Plain-permalink REST: ?rest_route=/wp/v2/… (P3.29). Prefer server-level
	// dispatch; this is a safety net when front is reached with the query var.
	if rr := strings.TrimSpace(parsed.QueryVars["rest_route"]); rr != "" {
		rs := &rest.Server{}
		rs.Handle(w, r, c)
		return
	}

	// redirect_canonical: enforce trailing slash on pretty permalinks (P3.15).
	if f.tryTrailingSlashRedirect(w, r, c, parsed) {
		return
	}

	if isFeedVars(parsed.QueryVars) {
		if isCommentsFeedVars(parsed.QueryVars) {
			if strings.EqualFold(parsed.QueryVars["feed"], "atom") {
				f.renderCommentsFeedAtom(w, r, c)
				return
			}
			f.renderCommentsFeedRSS2(w, r, c)
			return
		}
		f.renderFeed(w, r, c, parsed.QueryVars["feed"])
		return
	}

	// Front search results for /?s=term (P3.21).
	if term := strings.TrimSpace(parsed.QueryVars["s"]); term != "" || (c.Scope != nil && c.Scope.IsSearch) {
		f.renderSearch(w, c, term)
		return
	}

	if parsed.IsHome {
		f.renderHome(w, c)
		return
	}

	if isSingularVars(parsed.QueryVars) {
		if isEmbedVars(parsed.QueryVars) {
			f.renderEmbed(w, c, parsed.QueryVars)
			return
		}
		f.renderSingular(w, c, parsed.QueryVars)
		return
	}

	// Category archives: /category/{slug}/ (P3.25).
	if cat := parsed.QueryVars["category_name"]; cat != "" {
		f.renderCategoryArchive(w, c, cat)
		return
	}
	// Tag archives: /tag/{slug}/ (P3.24).
	if tag := parsed.QueryVars["tag"]; tag != "" {
		f.renderTagArchive(w, c, tag)
		return
	}

	// Date archives: /YYYY/ and /YYYY/MM/ (P3.22).
	if isDateArchiveVars(parsed.QueryVars) {
		f.renderDateArchive(w, c, parsed.QueryVars)
		return
	}

	// Author archives: /author/{nicename}/ (P3.23).
	if author := parsed.QueryVars["author_name"]; author != "" {
		f.renderAuthorArchive(w, c, author)
		return
	}

	// redirect_canonical lite: unmatched pretty paths whose last segment is a
	// published post slug (e.g. /blog/uncategorized/hello-world/ → date permalink).
	if parsed.Request != "" {
		if f.tryCanonicalPostRedirect(w, r, c, parsed.Request) {
			return
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "GoPress: front route not implemented (request=%q rule=%q)\n",
		parsed.Request, parsed.MatchedRule)
}

func isDateArchiveVars(vars map[string]string) bool {
	if vars == nil || vars["year"] == "" {
		return false
	}
	// Singular date permalinks carry name/p; those are handled earlier.
	if vars["name"] != "" || vars["p"] != "" || vars["page_id"] != "" || vars["pagename"] != "" {
		return false
	}
	return true
}

func isFeedVars(vars map[string]string) bool {
	if vars == nil {
		return false
	}
	feed := vars["feed"]
	if feed == "" {
		return false
	}
	// WP default feed is rss2; "feed" is the permalink alias for the default.
	switch strings.ToLower(feed) {
	case "feed", "rss", "rss2", "atom", "rdf":
		return true
	default:
		return false
	}
}

func (f *Front) renderFeed(w http.ResponseWriter, r *http.Request, c *wp.Context, feed string) {
	switch strings.ToLower(feed) {
	case "atom":
		f.renderFeedAtom(w, r, c)
	case "rdf":
		f.renderFeedRDF(w, r, c)
	default:
		f.renderFeedRSS2(w, r, c)
	}
}

func isCommentsFeedVars(vars map[string]string) bool {
	if vars == nil {
		return false
	}
	v := strings.TrimSpace(vars["withcomments"])
	return v == "1" || strings.EqualFold(v, "true")
}

// renderCommentsFeedRSS2 emits feed-rss2-comments.php shape (P3.27).
func (f *Front) renderCommentsFeedRSS2(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	desc := optionString(c, "blogdescription", "")
	comments, err := wp.ListRecentComments(c, 10)
	if err != nil {
		http.Error(w, "comments feed: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}

	// P3.58: self via feedSelfLink (request URI), same as RSS2/Atom posts feeds.
	self := feedSelfLink(r, c)

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"` + "\n")
	b.WriteString("\txmlns:content=\"http://purl.org/rss/1.0/modules/content/\"\n")
	b.WriteString("\txmlns:dc=\"http://purl.org/dc/elements/1.1/\"\n")
	b.WriteString("\txmlns:atom=\"http://www.w3.org/2005/Atom\"\n")
	b.WriteString("\txmlns:sy=\"http://purl.org/rss/1.0/modules/syndication/\"\n")
	b.WriteString("\t\n")
	b.WriteString("\t>\n")
	b.WriteString("<channel>\n")
	b.WriteString("\t<title>\n")
	b.WriteString("\tComments for ")
	b.WriteString(xmlEscape(blogname))
	b.WriteString("\t</title>\n")
	b.WriteString("\t<atom:link href=\"")
	b.WriteString(xmlEscape(self))
	b.WriteString("\" rel=\"self\" type=\"application/rss+xml\" />\n")
	b.WriteString("\t<link>")
	b.WriteString(xmlEscape(home))
	b.WriteString("</link>\n")
	b.WriteString("\t<description>")
	b.WriteString(xmlEscape(desc))
	b.WriteString("</description>\n")
	b.WriteString("\t<lastBuildDate>")
	b.WriteString(xmlEscape(commentsFeedBuildDate(c, comments)))
	b.WriteString("</lastBuildDate>\n")
	// P3.57: optional channel <ttl> when rss_ttl option is a positive integer.
	if ttl := wp.FeedTTLMinutes(c); ttl != "" {
		b.WriteString("\t<ttl>")
		b.WriteString(xmlEscape(ttl))
		b.WriteString("</ttl>\n")
	}
	b.WriteString("\t<sy:updatePeriod>\n")
	b.WriteString("\thourly\t</sy:updatePeriod>\n")
	b.WriteString("\t<sy:updateFrequency>\n")
	b.WriteString("\t1\t</sy:updateFrequency>\n")
	b.WriteString("\t")
	b.WriteString(wp.GetTheGenerator("rss2"))
	b.WriteString("\n")

	for _, cm := range comments {
		post, _ := wp.GetPostByID(c, cm.PostID)
		postTitle := ""
		link := fmt.Sprintf("%s/?p=%d#comment-%d", home, cm.PostID, cm.ID)
		if post != nil {
			postTitle = post.Title
			if plink := postPermalink(home, *post); plink != "" {
				link = strings.TrimRight(plink, "/") + "/#comment-" + strconv.FormatInt(cm.ID, 10)
			}
		}
		author := cm.Author
		if author == "" {
			author = "Anonymous"
		}
		title := "Comment on " + postTitle + " by " + author
		guid := fmt.Sprintf("%s/?p=%d#comment-%d", home, cm.PostID, cm.ID)
		contentHTML := commentFeedContent(cm.Content)
		descText := commentFeedDescription(cm.Content)

		b.WriteString("\t<item>\n")
		b.WriteString("\t\t<title>\n")
		b.WriteString("\t\t")
		b.WriteString(xmlEscape(title))
		b.WriteString("\t\t</title>\n")
		b.WriteString("\t\t<link>")
		b.WriteString(xmlEscape(link))
		b.WriteString("</link>\n\n")
		b.WriteString("\t\t<dc:creator><![CDATA[")
		b.WriteString(author)
		b.WriteString("]]></dc:creator>\n")
		b.WriteString("\t\t<pubDate>")
		b.WriteString(xmlEscape(feedRFC822(c, cm.DateGMT, cm.Date)))
		b.WriteString("</pubDate>\n")
		b.WriteString("\t\t<guid isPermaLink=\"false\">")
		b.WriteString(xmlEscape(guid))
		b.WriteString("</guid>\n\n")
		b.WriteString("\t\t\t\t\t<description><![CDATA[")
		b.WriteString(descText)
		b.WriteString("]]></description>\n")
		b.WriteString("\t\t\t<content:encoded><![CDATA[")
		b.WriteString(contentHTML)
		b.WriteString("]]></content:encoded>\n")
		b.WriteString("\t\t\n")
		b.WriteString("\t\t\t</item>\n")
	}
	b.WriteString("\t</channel>\n</rss>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "application/rss+xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func commentsFeedBuildDate(c *wp.Context, comments []wp.Comment) string {
	if len(comments) == 0 {
		// Stable empty-feed fallback (no wall clock; P3.55).
		return ""
	}
	// Newest comment first (ListRecentComments order).
	return feedRFC822(c, comments[0].DateGMT, comments[0].Date)
}

// renderCommentsFeedAtom emits feed-atom-comments.php shape (P3.34).
func (f *Front) renderCommentsFeedAtom(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	desc := optionString(c, "blogdescription", "")
	comments, err := wp.ListRecentComments(c, 10)
	if err != nil {
		http.Error(w, "comments feed: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}

	// P3.58: Atom comments self/id from request URI (get_self_link parity).
	self := feedSelfLink(r, c)

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><feed` + "\n")
	b.WriteString("\txmlns=\"http://www.w3.org/2005/Atom\"\n")
	b.WriteString("\txml:lang=\"")
	b.WriteString(xmlEscape(wp.BlogInfoLanguage(c)))
	b.WriteString("\"\n")
	b.WriteString("\txmlns:thr=\"http://purl.org/syndication/thread/1.0\"\n")
	b.WriteString("\t>\n")
	b.WriteString("\t<title type=\"text\">\n")
	b.WriteString("\tComments for ")
	b.WriteString(xmlEscape(blogname))
	b.WriteString("\t</title>\n")
	b.WriteString("\t<subtitle type=\"text\">")
	b.WriteString(xmlEscape(desc))
	b.WriteString("</subtitle>\n\n")
	b.WriteString("\t<updated>")
	if len(comments) > 0 {
		b.WriteString(xmlEscape(feedAtomDate(c, comments[0].DateGMT, comments[0].Date)))
	} else {
		b.WriteString(xmlEscape(feedAtomDate(c, "", "")))
	}
	b.WriteString("</updated>\n\n")
	b.WriteString("\t<link rel=\"alternate\" type=\"text/html\" href=\"")
	b.WriteString(xmlEscape(home))
	b.WriteString("\" />\n")
	b.WriteString("\t<link rel=\"self\" type=\"application/atom+xml\" href=\"")
	b.WriteString(xmlEscape(self))
	b.WriteString("\" />\n")
	b.WriteString("\t<id>")
	b.WriteString(xmlEscape(self))
	b.WriteString("</id>\n")
	b.WriteString(wp.GetTheGenerator("atom"))
	b.WriteString("\n")

	for _, cm := range comments {
		post, _ := wp.GetPostByID(c, cm.PostID)
		postTitle := ""
		postLink := fmt.Sprintf("%s/?p=%d", home, cm.PostID)
		postGUID := postLink
		link := fmt.Sprintf("%s/?p=%d#comment-%d", home, cm.PostID, cm.ID)
		if post != nil {
			postTitle = post.Title
			if plink := postPermalink(home, *post); plink != "" {
				postLink = strings.TrimRight(plink, "/") + "/"
				link = strings.TrimRight(plink, "/") + "/#comment-" + strconv.FormatInt(cm.ID, 10)
			}
			if post.GUID != "" {
				postGUID = post.GUID
			}
		}
		author := cm.Author
		if author == "" {
			author = "Anonymous"
		}
		title := "Comment on " + postTitle + " by " + author
		guid := fmt.Sprintf("%s/?p=%d#comment-%d", home, cm.PostID, cm.ID)
		contentHTML := commentFeedContent(cm.Content)

		b.WriteString("\t<entry>\n")
		b.WriteString("\t\t<title>\n")
		b.WriteString("\t\t")
		b.WriteString(xmlEscape(title))
		b.WriteString("\t\t</title>\n")
		b.WriteString("\t\t<link rel=\"alternate\" href=\"")
		b.WriteString(xmlEscape(link))
		b.WriteString("\" type=\"text/html\" />\n\n")
		b.WriteString("\t\t<author>\n")
		b.WriteString("\t\t\t<name>")
		b.WriteString(xmlEscape(author))
		b.WriteString("</name>\n")
		if cm.AuthorURL != "" {
			b.WriteString("\t\t\t<uri>")
			b.WriteString(xmlEscape(cm.AuthorURL))
			b.WriteString("</uri>\n")
		}
		b.WriteString("\t\t</author>\n\n")
		b.WriteString("\t\t<id>")
		b.WriteString(xmlEscape(guid))
		b.WriteString("</id>\n")
		b.WriteString("\t\t<updated>")
		b.WriteString(xmlEscape(feedAtomDate(c, cm.DateGMT, cm.Date)))
		b.WriteString("</updated>\n")
		b.WriteString("\t\t<published>")
		b.WriteString(xmlEscape(feedAtomDate(c, cm.DateGMT, cm.Date)))
		b.WriteString("</published>\n\n")
		b.WriteString("\t\t\t\t\t<content type=\"html\" xml:base=\"")
		b.WriteString(xmlEscape(link))
		b.WriteString("\"><![CDATA[")
		b.WriteString(contentHTML)
		b.WriteString("]]></content>\n")
		b.WriteString("\t\t\n")
		if cm.Parent == 0 {
			b.WriteString("\t\t\t\t\t<thr:in-reply-to ref=\"")
			b.WriteString(xmlEscape(postGUID))
			b.WriteString("\" href=\"")
			b.WriteString(xmlEscape(postLink))
			b.WriteString("\" type=\"text/html\" />\n")
		} else {
			parentGUID := fmt.Sprintf("%s/?p=%d#comment-%d", home, cm.PostID, cm.Parent)
			parentLink := fmt.Sprintf("%s/?p=%d#comment-%d", home, cm.PostID, cm.Parent)
			if post != nil {
				if plink := postPermalink(home, *post); plink != "" {
					parentLink = strings.TrimRight(plink, "/") + "/#comment-" + strconv.FormatInt(cm.Parent, 10)
				}
			}
			b.WriteString("\t\t\t\t\t<thr:in-reply-to ref=\"")
			b.WriteString(xmlEscape(parentGUID))
			b.WriteString("\" href=\"")
			b.WriteString(xmlEscape(parentLink))
			b.WriteString("\" type=\"text/html\" />\n")
		}
		b.WriteString("\t\t\t\t</entry>\n")
	}
	b.WriteString("</feed>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "application/atom+xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func commentFeedContent(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Minimal the_content-ish: paragraphs + br for newlines.
	escaped := html.EscapeString(raw)
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\n")
	parts := strings.Split(escaped, "\n\n")
	var b strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = strings.ReplaceAll(p, "\n", "<br />\n")
		b.WriteString("<p>")
		b.WriteString(p)
		b.WriteString("</p>\n")
	}
	return b.String()
}

func commentFeedDescription(raw string) string {
	// comment_text_rss: HTML-escaped plain text in CDATA.
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	return html.EscapeString(raw)
}

func isSingularVars(vars map[string]string) bool {
	if vars == nil {
		return false
	}
	for _, k := range []string{"p", "page_id", "name", "pagename", "attachment_id", "attachment"} {
		if v := vars[k]; v != "" {
			return true
		}
	}
	return false
}

func isEmbedVars(vars map[string]string) bool {
	if vars == nil {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(vars["embed"]))
	return v == "true" || v == "1"
}

func (f *Front) renderHome(w http.ResponseWriter, c *wp.Context) {
	// P5.5/P5.24: prefer block theme front-page/home/index hierarchy over shim.
	isFront := c != nil && c.Scope.IsFrontPage
	slugs := wp.HomeTemplateSlugs(isFront || (c != nil && c.Scope.IsHome))
	if tpl, ok := wp.ResolveBlockTemplate(c, slugs...); ok && strings.TrimSpace(tpl) != "" {
		f.renderBlockTemplate(w, c, tpl)
		return
	}
	f.renderHomeShim(w, c)
}

// renderSearch serves /?s= front search results (P3.21 / P3.36).
func (f *Front) renderSearch(w http.ResponseWriter, c *wp.Context, term string) {
	if c != nil && c.Scope != nil {
		c.Scope.IsSearch = true
		if term == "" {
			term = c.Scope.SearchQuery
		} else {
			c.Scope.SearchQuery = term
		}
	}
	// Prefer theme templates/search.html (then index) and ensure header/footer parts.
	slugs := wp.SearchTemplateSlugs()
	if tpl, ok := wp.ResolveBlockTemplate(c, slugs...); ok && strings.TrimSpace(tpl) != "" {
		f.renderSearchBlockTemplate(w, c, EnsureSearchTemplateParts(tpl), term)
		return
	}
	if wrapped, ok := wrapSearchShimWithParts(c, term); ok {
		f.renderSearchBlockTemplate(w, c, wrapped, term)
		return
	}
	f.renderSearchShim(w, c, term)
}

// EnsureSearchTemplateParts prepends/appends header/footer template parts when
// the resolved search template omits them (P3.36).
func EnsureSearchTemplateParts(template string) string {
	tpl := strings.TrimSpace(template)
	if tpl == "" {
		return `<!-- wp:template-part {"slug":"header"} /-->` + "\n\n" +
			`<!-- wp:query-title {"type":"search"} /-->` + "\n\n" +
			`<!-- wp:template-part {"slug":"footer"} /-->`
	}
	hasHeader := strings.Contains(tpl, `"slug":"header"`) || strings.Contains(tpl, `"slug": "header"`)
	hasFooter := strings.Contains(tpl, `"slug":"footer"`) || strings.Contains(tpl, `"slug": "footer"`)
	if hasHeader && hasFooter {
		return tpl
	}
	var b strings.Builder
	if !hasHeader {
		b.WriteString(`<!-- wp:template-part {"slug":"header"} /-->`)
		b.WriteString("\n\n")
	}
	b.WriteString(tpl)
	if !hasFooter {
		b.WriteString("\n\n")
		b.WriteString(`<!-- wp:template-part {"slug":"footer"} /-->`)
	}
	return b.String()
}

// wrapSearchShimWithParts builds a minimal search block document using theme parts.
func wrapSearchShimWithParts(c *wp.Context, term string) (string, bool) {
	if c == nil {
		return "", false
	}
	_, hasHeader := wp.ResolveTemplatePart(c, "header")
	_, hasFooter := wp.ResolveTemplatePart(c, "footer")
	if !hasHeader && !hasFooter {
		return "", false
	}
	_ = term
	return EnsureSearchTemplateParts(
		`<!-- wp:group {"tagName":"main","layout":{"type":"constrained"}} -->` +
			`<main class="wp-block-group"><!-- wp:query-title {"type":"search"} /-->` +
			`<!-- wp:search {"label":"Search","showLabel":false,"placeholder":"Type here...","buttonText":"Search"} /-->` +
			`<!-- wp:query {"query":{"perPage":10,"inherit":true}} -->` +
			`<div class="wp-block-query"><!-- wp:post-template --><!-- wp:post-title {"isLink":true} /--><!-- /wp:post-template --></div>` +
			`<!-- /wp:query --></main><!-- /wp:group -->`,
	), true
}

func (f *Front) renderSearchBlockTemplate(w http.ResponseWriter, c *wp.Context, template, term string) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.SearchQuery = term
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPost(body))

	bodyClasses := "search wp-embed-responsive"
	if term != "" {
		bodyClasses += " search-results"
	} else {
		bodyClasses += " search-no-results"
	}
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)
	safe = ensureMainSkipTarget(safe)

	title := blogname
	if term != "" {
		title = "Search Results for &#8220;" + html.EscapeString(term) + "&#8221; &#8211; " + html.EscapeString(blogname)
	} else {
		title = "Search &#8211; " + html.EscapeString(blogname)
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(title)
	b.WriteString("</title>\n")
	extraTitle, extraHref := "", ""
	if term != "" {
		// feed_links_extra search: "%1$s %2$s Search Results for &#8220;%3$s&#8221; Feed"
		extraTitle = html.EscapeString(blogname) + " &raquo; Search Results for &#8220;" + html.EscapeString(term) + "&#8221; Feed"
		extraHref = home + "/search/" + pathEscapeSearch(term) + "/feed/rss2/"
	}
	writeFrontHeadLinks(&b, home, blogname, "", "", "", extraTitle, extraHref, "")
	writeArchiveRelLinks(&b, c, home)
	b.WriteString(wp.EmojiHeadHTML())
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	b.WriteString("</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderDateArchive serves year/month(/day) front archives (P3.22).
func (f *Front) renderDateArchive(w http.ResponseWriter, c *wp.Context, vars map[string]string) {
	year := atoiFront(vars["year"])
	month := atoiFront(vars["monthnum"])
	day := atoiFront(vars["day"])
	if year <= 0 {
		f.render404(w, c)
		return
	}
	if c != nil && c.Scope != nil {
		c.Scope.IsArchive = true
		c.Scope.IsDate = true
		c.Scope.DateYear = year
		c.Scope.DateMonth = month
		c.Scope.DateDay = day
		c.Scope.IsYear = month == 0 && day == 0
		c.Scope.IsMonth = month > 0 && day == 0
		c.Scope.IsDay = month > 0 && day > 0
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 1,
		Year: year, MonthNum: month, Day: day,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if len(posts) == 0 {
		if c != nil && c.Scope != nil {
			c.Scope.Is404 = true
		}
		f.render404(w, c)
		return
	}
	title := dateArchiveTitle(year, month, day)
	slugs := wp.DateTemplateSlugs()
	if tpl, ok := wp.ResolveBlockTemplate(c, slugs...); ok && strings.TrimSpace(tpl) != "" {
		f.renderDateArchiveBlockTemplate(w, c, tpl, title)
		return
	}
	f.renderDateArchiveShim(w, c, title, year, month, day)
}

func dateArchiveTitle(year, month, day int) string {
	if day > 0 && month > 0 {
		return fmt.Sprintf("Day: %s %d, %d", time.Month(month).String(), day, year)
	}
	if month > 0 {
		return fmt.Sprintf("Month: %s %d", time.Month(month).String(), year)
	}
	return fmt.Sprintf("Year: %d", year)
}

func atoiFront(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func (f *Front) renderDateArchiveBlockTemplate(w http.ResponseWriter, c *wp.Context, template, archiveTitle string) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.ArchiveTitle = archiveTitle
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPost(body))
	safe = ensureMainSkipTarget(safe)

	bodyClasses := "archive date wp-embed-responsive"
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)

	pageTitle := html.EscapeString(archiveTitle) + " &#8211; " + html.EscapeString(blogname)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(pageTitle)
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "", "", "", "")
	writeArchiveRelLinks(&b, c, home)
	b.WriteString(wp.EmojiHeadHTML())
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	b.WriteString("</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderAuthorArchive serves /author/{nicename}/ front archives (P3.23).
func (f *Front) renderAuthorArchive(w http.ResponseWriter, c *wp.Context, nicename string) {
	nicename = strings.TrimSpace(nicename)
	if nicename == "" {
		f.render404(w, c)
		return
	}
	user, err := c.GetUserBy("slug", nicename)
	if err != nil || user == nil {
		// Also try login as nicename fallback.
		user, err = c.GetUserBy("login", nicename)
	}
	if err != nil || user == nil {
		if c != nil && c.Scope != nil {
			c.Scope.Is404 = true
		}
		f.render404(w, c)
		return
	}
	if c != nil && c.Scope != nil {
		c.Scope.IsArchive = true
		c.Scope.IsAuthor = true
		c.Scope.AuthorID = user.ID
		c.Scope.AuthorName = user.Nicename
		if c.Scope.AuthorName == "" {
			c.Scope.AuthorName = user.Login
		}
		c.Scope.QueriedObjectID = user.ID
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 1,
		Author: user.ID,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if len(posts) == 0 {
		if c != nil && c.Scope != nil {
			c.Scope.Is404 = true
		}
		f.render404(w, c)
		return
	}
	display := user.DisplayName
	if display == "" {
		display = user.Login
	}
	title := "Author: " + display
	slugs := wp.AuthorTemplateSlugs(user.Nicename, user.ID)
	if tpl, ok := wp.ResolveBlockTemplate(c, slugs...); ok && strings.TrimSpace(tpl) != "" {
		f.renderAuthorArchiveBlockTemplate(w, c, tpl, title, user)
		return
	}
	f.renderAuthorArchiveShim(w, c, title, user.ID)
}

func (f *Front) renderAuthorArchiveBlockTemplate(w http.ResponseWriter, c *wp.Context, template, archiveTitle string, user *wp.User) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.ArchiveTitle = archiveTitle
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPost(body))
	safe = ensureMainSkipTarget(safe)

	nicename := ""
	if user != nil {
		nicename = user.Nicename
		if nicename == "" {
			nicename = user.Login
		}
	}
	bodyClasses := "archive author wp-embed-responsive"
	if nicename != "" {
		bodyClasses += " author-" + nicename
	}
	if user != nil && user.ID > 0 {
		bodyClasses += " author-" + strconv.FormatInt(user.ID, 10)
	}
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)

	pageTitle := html.EscapeString(archiveTitle) + " &#8211; " + html.EscapeString(blogname)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(pageTitle)
	b.WriteString("</title>\n")
	extraTitle, extraHref, restPath := "", "", ""
	if user != nil {
		display := user.DisplayName
		if display == "" {
			display = user.Login
		}
		slug := nicename
		// feed_links_extra author: "%1$s %2$s Posts by %3$s Feed"
		extraTitle = html.EscapeString(blogname) + " &raquo; Posts by " + html.EscapeString(display) + " Feed"
		extraHref = home + "/author/" + slug + "/feed/"
		restPath = "/wp/v2/users/" + strconv.FormatInt(user.ID, 10)
	}
	writeFrontHeadLinks(&b, home, blogname, "", restPath, "", extraTitle, extraHref, "")
	writeArchiveRelLinks(&b, c, home)
	b.WriteString(wp.EmojiHeadHTML())
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	b.WriteString("</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderCategoryArchive serves /category/{slug}/ front archives (P3.25).
func (f *Front) renderCategoryArchive(w http.ResponseWriter, c *wp.Context, slug string) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		f.render404(w, c)
		return
	}
	// Nested category paths use the leaf slug for term lookup.
	leaf := slug
	if i := strings.LastIndex(slug, "/"); i >= 0 {
		leaf = slug[i+1:]
	}
	term, err := getTermBySlug(c, "category", leaf)
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if term == nil {
		if c != nil && c.Scope != nil {
			c.Scope.Is404 = true
		}
		f.render404(w, c)
		return
	}
	if c != nil && c.Scope != nil {
		c.Scope.IsArchive = true
		c.Scope.IsCategory = true
		c.Scope.CategorySlug = term.Slug
		c.Scope.CategoryID = term.ID
		c.Scope.QueriedObjectID = term.ID
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 1,
		Category: term.Slug,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if len(posts) == 0 {
		if c != nil && c.Scope != nil {
			c.Scope.Is404 = true
		}
		f.render404(w, c)
		return
	}
	title := "Category: " + term.Name
	slugs := wp.CategoryTemplateSlugs(term.Slug, term.ID)
	if tpl, ok := wp.ResolveBlockTemplate(c, slugs...); ok && strings.TrimSpace(tpl) != "" {
		f.renderCategoryArchiveBlockTemplate(w, c, tpl, title, term)
		return
	}
	f.renderCategoryArchiveShim(w, c, title, term.Slug)
}

func (f *Front) renderCategoryArchiveBlockTemplate(w http.ResponseWriter, c *wp.Context, template, archiveTitle string, term *termInfo) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.ArchiveTitle = archiveTitle
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPost(body))
	safe = ensureMainSkipTarget(safe)

	bodyClasses := "archive category wp-embed-responsive"
	if term != nil && term.Slug != "" {
		bodyClasses += " category-" + term.Slug
	}
	if term != nil && term.ID > 0 {
		bodyClasses += " category-" + strconv.FormatInt(term.ID, 10)
	}
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)

	pageTitle := html.EscapeString(archiveTitle) + " &#8211; " + html.EscapeString(blogname)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(pageTitle)
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "", "", "", "")
	writeArchiveRelLinks(&b, c, home)
	b.WriteString(wp.EmojiHeadHTML())
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	b.WriteString("</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderCategoryArchiveShim(w http.ResponseWriter, c *wp.Context, title, catSlug string) {
	blogname := optionString(c, "blogname", "GoPress")
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 10,
		Category: catSlug,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString(" &#8211; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "archive category")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1 class=\"page-title\">")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>\n<ul class=\"posts\">\n")
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a></li>\n")
	}
	b.WriteString("</ul>\n</body>\n</html>\n")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderTagArchive serves /tag/{slug}/ front archives (P3.24).
func (f *Front) renderTagArchive(w http.ResponseWriter, c *wp.Context, slug string) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		f.render404(w, c)
		return
	}
	term, err := getTermBySlug(c, "post_tag", slug)
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if term == nil {
		if c != nil && c.Scope != nil {
			c.Scope.Is404 = true
		}
		f.render404(w, c)
		return
	}
	if c != nil && c.Scope != nil {
		c.Scope.IsArchive = true
		c.Scope.IsTag = true
		c.Scope.TagSlug = term.Slug
		c.Scope.TagID = term.ID
		c.Scope.QueriedObjectID = term.ID
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 1,
		Tag: term.Slug,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if len(posts) == 0 {
		if c != nil && c.Scope != nil {
			c.Scope.Is404 = true
		}
		f.render404(w, c)
		return
	}
	title := "Tag: " + term.Name
	slugs := wp.TagTemplateSlugs(term.Slug, term.ID)
	if tpl, ok := wp.ResolveBlockTemplate(c, slugs...); ok && strings.TrimSpace(tpl) != "" {
		f.renderTagArchiveBlockTemplate(w, c, tpl, title, term)
		return
	}
	f.renderTagArchiveShim(w, c, title, term.Slug)
}

func (f *Front) renderTagArchiveBlockTemplate(w http.ResponseWriter, c *wp.Context, template, archiveTitle string, term *termInfo) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.ArchiveTitle = archiveTitle
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPost(body))
	safe = ensureMainSkipTarget(safe)

	bodyClasses := "archive tag wp-embed-responsive"
	if term != nil && term.Slug != "" {
		bodyClasses += " tag-" + term.Slug
	}
	if term != nil && term.ID > 0 {
		bodyClasses += " tag-" + strconv.FormatInt(term.ID, 10)
	}
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)

	pageTitle := html.EscapeString(archiveTitle) + " &#8211; " + html.EscapeString(blogname)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(pageTitle)
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "", "", "", "")
	writeArchiveRelLinks(&b, c, home)
	b.WriteString(wp.EmojiHeadHTML())
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	b.WriteString("</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderTagArchiveShim(w http.ResponseWriter, c *wp.Context, title, tagSlug string) {
	blogname := optionString(c, "blogname", "GoPress")
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 10,
		Tag: tagSlug,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString(" &#8211; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "archive tag")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1 class=\"page-title\">")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>\n<ul class=\"posts\">\n")
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a></li>\n")
	}
	b.WriteString("</ul>\n</body>\n</html>\n")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

type termInfo struct {
	ID   int64
	Name string
	Slug string
}

func getTermBySlug(c *wp.Context, taxonomy, slug string) (*termInfo, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT t.term_id, t.name, t.slug FROM "+c.DB.Terms()+" t "+
			"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
			"WHERE tt.taxonomy = %s AND t.slug = %s LIMIT 1",
		taxonomy, slug,
	)
	if err != nil {
		return nil, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	id, _ := strconv.ParseInt(fmt.Sprint(row["term_id"]), 10, 64)
	return &termInfo{
		ID:   id,
		Name: fmt.Sprint(row["name"]),
		Slug: fmt.Sprint(row["slug"]),
	}, nil
}

func (f *Front) renderAuthorArchiveShim(w http.ResponseWriter, c *wp.Context, title string, authorID int64) {
	blogname := optionString(c, "blogname", "GoPress")
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 10,
		Author: authorID,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString(" &#8211; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "archive author")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1 class=\"page-title\">")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>\n<ul class=\"posts\">\n")
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a></li>\n")
	}
	b.WriteString("</ul>\n</body>\n</html>\n")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderDateArchiveShim(w http.ResponseWriter, c *wp.Context, title string, year, month, day int) {
	blogname := optionString(c, "blogname", "GoPress")
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 10,
		Year: year, MonthNum: month, Day: day,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString(" &#8211; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "archive date")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1 class=\"page-title\">")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>\n<ul class=\"posts\">\n")
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a></li>\n")
	}
	b.WriteString("</ul>\n</body>\n</html>\n")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderSearchShim(w http.ResponseWriter, c *wp.Context, term string) {
	blogname := optionString(c, "blogname", "GoPress")
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 10, Search: term,
	})
	if err != nil {
		http.Error(w, "search: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Search Results for &#8220;")
	b.WriteString(html.EscapeString(term))
	b.WriteString("&#8221; &#8211; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "search search-results")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1 class=\"page-title\">Search results for: ")
	b.WriteString(html.EscapeString(term))
	b.WriteString("</h1>\n<ul class=\"posts\">\n")
	if len(posts) == 0 {
		b.WriteString("<li class=\"no-results\">No results found.</li>\n")
	}
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a></li>\n")
	}
	b.WriteString("</ul>\n</body>\n</html>\n")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderBlockTemplate parses and renders a block theme template document.
func (f *Front) renderBlockTemplate(w http.ResponseWriter, c *wp.Context, template string) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.IsFrontPage = func() bool { return true }
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPost(body))

	bodyClasses := "home blog wp-embed-responsive"
	paged := 1
	if c != nil && c.Scope != nil && c.Scope.Paged > 1 {
		paged = c.Scope.Paged
		bodyClasses += " paged paged-" + strconv.Itoa(paged)
	}
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)
	safe = ensureMainSkipTarget(safe)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(blogname))
	if paged > 1 {
		b.WriteString(" &#8211; Page ")
		b.WriteString(strconv.Itoa(paged))
	}
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "", "", "", "")
	b.WriteString(wp.EmojiHeadHTML()) // P3.12: wpemoji print scripts skipped
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	b.WriteString("</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func activeStylesheet(c *wp.Context) string {
	if c != nil && c.Theme != nil && c.Theme.Theme != "" {
		return c.Theme.Theme
	}
	return optionString(c, "stylesheet", optionString(c, "template", "twentytwentyfive"))
}

func (f *Front) renderHomeShim(w http.ResponseWriter, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	posts, err := latestPosts(c, 5)
	if err != nil {
		http.Error(w, "posts: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n")
	b.WriteString(wp.EmojiHeadHTML()) // P3.12: wpemoji print scripts skipped
	b.WriteString("</head>\n<body")
	if cls := wp.BodyClassWithAdminBar(c, ""); cls != "" {
		b.WriteString(` class="`)
		b.WriteString(html.EscapeString(cls))
		b.WriteString(`"`)
	}
	b.WriteString(">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</h1>\n")
	if c.CurrentUser != nil {
		b.WriteString("<p class=\"user\">Logged in as ")
		b.WriteString(html.EscapeString(c.CurrentUser.Login))
		b.WriteString("</p>\n")
	}
	b.WriteString("<ul class=\"posts\">\n")
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a></li>\n")
	}
	b.WriteString("</ul>\n</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderSingular(w http.ResponseWriter, c *wp.Context, vars map[string]string) {
	post, err := loadSingularPost(c, vars)
	if err != nil {
		http.Error(w, "post: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if post == nil {
		f.render404(w, c)
		return
	}

	// P5.17/P5.24: prefer block theme singular hierarchy when available.
	// Attachments keep the shim until attachment-* templates land.
	if post.Type == "page" || post.Type == "post" {
		slugs := wp.SingularTemplateSlugs(post.Type, post.Name, post.ID)
		if tpl, ok := wp.ResolveBlockTemplate(c, slugs...); ok && strings.TrimSpace(tpl) != "" {
			f.renderSingularBlockTemplate(w, c, post, tpl)
			return
		}
	}

	f.renderSingularShim(w, c, post)
}

func (f *Front) renderSingularBlockTemplate(w http.ResponseWriter, c *wp.Context, post *singularPost, template string) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.SetCurrentPost(&blocks.QueryPost{
		ID:               post.ID,
		Title:            post.Title,
		Content:          post.Content,
		Date:             post.Date,
		Link:             singularPermalink(home, post),
		FeaturedImageURL: post.FeaturedImageURL,
		FeaturedImageAlt: post.FeaturedImageAlt,
		Password:         post.Password,
		ClassList:        wp.GetPostClass(c, post.ID),
	})
	env.PasswordRequired = func(postID int64) bool {
		if postID != post.ID {
			return false
		}
		wpPost := &wp.Post{ID: post.ID, Title: post.Title, Type: post.Type, Name: post.Name, Password: post.Password}
		return wp.PostPasswordRequired(c, c.Req, wpPost)
	}
	// Slot marker survives Kses; real form is spliced in after (P5.18 / P5.19).
	const commentFormSlot = `<div class="gopress-comment-form-slot"></div>`
	env.RenderCommentForm = func(postID int64) string {
		if postID <= 0 {
			return ""
		}
		return commentFormSlot
	}
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPost(body))
	safe = strings.Replace(safe, commentFormSlot, commentFormHTML(c, post), 1)
	safe = ensureMainSkipTarget(safe)

	bodyClasses := "wp-singular wp-embed-responsive"
	if post.Type == "page" {
		bodyClasses += " page-template-default page page-id-" + strconv.FormatInt(post.ID, 10)
	} else {
		bodyClasses += " post-template-default single single-post postid-" + strconv.FormatInt(post.ID, 10) + " single-format-standard"
	}
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString(" &#8211; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n")
	perm := singularPermalink(home, post)
	if canon := wp.GetCanonicalURL(c, post.ID, 0, 0); canon != "" {
		perm = canon
	}
	restPath := "/wp/v2/posts/" + strconv.FormatInt(post.ID, 10)
	if post.Type == "page" {
		restPath = "/wp/v2/pages/" + strconv.FormatInt(post.ID, 10)
	}
	shortlink := wp.GetShortlink(c, post.ID, "post", 0)
	writeFrontHeadLinks(&b, home, blogname, perm, restPath, post.Title, "", "", shortlink)
	b.WriteString(wp.EmojiHeadHTML())
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	// trackback_rdf() / xmlns:trackback is intentionally omitted (P3.46): trackbacks are gone.
	b.WriteString("</body>\n</html>\n")

	setFrontSingularLinks(w, c, post)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderSingularShim(w http.ResponseWriter, c *wp.Context, post *singularPost) {
	blogname := optionString(c, "blogname", "GoPress")
	wpPost := &wp.Post{ID: post.ID, Title: post.Title, Content: post.Content, Type: post.Type, Name: post.Name, Password: post.Password}
	passwordRequired := wp.PostPasswordRequired(c, c.Req, wpPost)

	var safeContent string
	if passwordRequired {
		safeContent = wp.GetThePasswordForm(c, wpPost, "")
	} else {
		content := post.Content
		if post.Type == "attachment" && content == "" {
			if url := wp.GetAttachmentURL(c, post.ID); url != "" {
				content = `<p class="attachment"><a href="` + html.EscapeString(url) + `">` +
					html.EscapeString(post.Title) + `</a></p>`
			}
		}
		// the_content: blocks → shortcodes → texturize (P5.11).
		content = wp.TheContent(c, content)
		safeContent = wp.ProcessInteractivityHTML(wp.KsesPost(content))
	}

	bodyClass := "singular"
	if post.Type == "attachment" {
		bodyClass = "singular attachment"
	}
	if passwordRequired {
		bodyClass += " post-password-required"
	} else if strings.TrimSpace(post.Password) != "" {
		bodyClass += " post-password-protected"
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString(" &#8212; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, bodyClass)))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<article id=\"post-")
	b.WriteString(strconv.FormatInt(post.ID, 10))
	b.WriteString("\" class=\"post type-")
	b.WriteString(html.EscapeString(post.Type))
	b.WriteString("\">\n")
	b.WriteString("<h1 class=\"entry-title\">")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString("</h1>\n")
	if !passwordRequired && post.FeaturedImageURL != "" {
		b.WriteString("<figure class=\"post-thumbnail\">\n")
		b.WriteString(`<img src="`)
		b.WriteString(html.EscapeString(post.FeaturedImageURL))
		b.WriteString(`" alt="`)
		b.WriteString(html.EscapeString(post.FeaturedImageAlt))
		b.WriteString(`" class="attachment-post-thumbnail size-post-thumbnail"/>` + "\n")
		b.WriteString("</figure>\n")
	}
	b.WriteString("<div class=\"entry-content\">\n")
	b.WriteString(safeContent)
	b.WriteString("\n</div>\n</article>\n")
	if !passwordRequired && (post.Type == "post" || post.Type == "page") {
		f.writeCommentsSection(&b, c, post)
	}
	b.WriteString("</body>\n</html>\n")

	setFrontSingularLinks(w, c, post)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderEmbed serves the post embed endpoint (P3.18) — theme-compat/embed.php shape.
func (f *Front) renderEmbed(w http.ResponseWriter, c *wp.Context, vars map[string]string) {
	post, err := loadSingularPost(c, vars)
	if err != nil {
		http.Error(w, "post: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if post == nil {
		f.renderEmbed404(w, c)
		return
	}
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	perm := singularPermalink(home, post)
	excerpt := wp.TheContent(c, post.Content)
	excerpt = wp.ProcessInteractivityHTML(wp.KsesPost(excerpt))
	// Keep embed payloads small.
	if len(excerpt) > 400 {
		excerpt = excerpt[:400] + "&hellip;"
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\" class=\"no-js\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString(" &#8212; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n")
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "wp-embed")))
	b.WriteString("\">\n")
	b.WriteString("<div class=\"wp-embed\">\n")
	b.WriteString("<p class=\"wp-embed-heading\"><a href=\"")
	b.WriteString(html.EscapeString(perm))
	b.WriteString("\" target=\"_top\">")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString("</a></p>\n")
	b.WriteString("<div class=\"wp-embed-excerpt\">")
	b.WriteString(excerpt)
	b.WriteString("</div>\n")
	b.WriteString("<div class=\"wp-embed-footer\">\n")
	b.WriteString("<a href=\"")
	b.WriteString(html.EscapeString(home + "/"))
	b.WriteString("\" target=\"_top\">")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</a>\n")
	b.WriteString("</div>\n</div>\n</body>\n</html>\n")

	w.Header().Set("X-WP-embed", "true")
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderEmbed404(w http.ResponseWriter, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\" class=\"no-js\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n<title>")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"wp-embed\">\n")
	b.WriteString("<div class=\"wp-embed\"><p class=\"wp-embed-heading\">Oops! That embed could not be found.</p></div>\n")
	b.WriteString("</body>\n</html>\n")
	w.Header().Set("X-WP-embed", "true")
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(b.String()))
}

func singularPermalink(home string, post *singularPost) string {
	if post == nil {
		return home + "/"
	}
	if post.Type == "page" && post.Name != "" {
		return home + "/" + post.Name + "/"
	}
	if post.Type == "post" && post.Name != "" && post.Date != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", post.Date); err == nil {
			return fmt.Sprintf("%s/%04d/%02d/%02d/%s/", home, t.Year(), int(t.Month()), t.Day(), post.Name)
		}
	}
	if post.Name != "" {
		return home + "/?p=" + strconv.FormatInt(post.ID, 10)
	}
	return home + "/?p=" + strconv.FormatInt(post.ID, 10)
}

func (f *Front) writeCommentsSection(b *strings.Builder, c *wp.Context, post *singularPost) {
	comments, err := wp.ListApprovedComments(c, post.ID)
	if err != nil {
		comments = nil
	}
	b.WriteString("<section id=\"comments\" class=\"comments-area\">\n")
	b.WriteString("<h2 class=\"comments-title\">Comments</h2>\n")
	if len(comments) == 0 {
		b.WriteString("<p class=\"no-comments\">No comments yet.</p>\n")
	} else {
		b.WriteString("<ol class=\"comment-list\">\n")
		for _, cm := range comments {
			b.WriteString("<li id=\"comment-")
			b.WriteString(strconv.FormatInt(cm.ID, 10))
			b.WriteString("\" class=\"comment\">\n")
			b.WriteString("<article class=\"comment-body\">\n")
			b.WriteString("<footer class=\"comment-meta\"><b class=\"fn\">")
			b.WriteString(html.EscapeString(cm.Author))
			b.WriteString("</b></footer>\n")
			b.WriteString("<div class=\"comment-content\">\n")
			b.WriteString(wp.KsesData(cm.Content))
			b.WriteString("\n</div>\n</article>\n</li>\n")
		}
		b.WriteString("</ol>\n")
	}
	b.WriteString(commentFormHTML(c, post))
	b.WriteString("</section>\n")
}

// commentFormHTML mirrors WP comment_form() markup used under posts (P5.18).
func commentFormHTML(c *wp.Context, post *singularPost) string {
	if post == nil {
		return ""
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	action := "/wp-comments-post.php"
	if home != "" {
		action = home + "/wp-comments-post.php"
	}
	cancelHref := "#respond"
	if home != "" {
		// WP prints a root-relative cancel URL on front templates.
		perm := singularPermalink(home, post)
		cancelHref = strings.TrimPrefix(perm, home) + "#respond"
		if !strings.HasPrefix(cancelHref, "/") {
			cancelHref = "/" + cancelHref
		}
	}
	var b strings.Builder
	b.WriteString(`<div id="respond" class="comment-respond wp-block-post-comments-form">` + "\n")
	b.WriteString(`<h3 id="reply-title" class="comment-reply-title">Leave a Reply `)
	b.WriteString(`<small><a rel="nofollow" id="cancel-comment-reply-link" href="`)
	b.WriteString(html.EscapeString(cancelHref))
	b.WriteString(`" style="display:none;">Cancel reply</a></small></h3>`)
	b.WriteString(`<form action="`)
	b.WriteString(html.EscapeString(action))
	b.WriteString(`" method="post" id="commentform" class="comment-form">`)
	b.WriteString(`<p class="comment-notes"><span id="email-notes">Your email address will not be published.</span> `)
	b.WriteString(`<span class="required-field-message">Required fields are marked <span class="required">*</span></span></p>`)
	b.WriteString(`<p class="comment-form-comment"><label for="comment">Comment <span class="required">*</span></label> `)
	b.WriteString(`<textarea id="comment" name="comment" cols="45" rows="8" maxlength="65525" required></textarea></p>`)
	if c == nil || c.CurrentUser == nil {
		b.WriteString(`<p class="comment-form-author"><label for="author">Name <span class="required">*</span></label> `)
		b.WriteString(`<input id="author" name="author" type="text" value="" size="30" maxlength="245" autocomplete="name" required /></p>` + "\n")
		b.WriteString(`<p class="comment-form-email"><label for="email">Email <span class="required">*</span></label> `)
		b.WriteString(`<input id="email" name="email" type="email" value="" size="30" maxlength="100" aria-describedby="email-notes" autocomplete="email" required /></p>` + "\n")
		b.WriteString(`<p class="comment-form-url"><label for="url">Website</label> `)
		b.WriteString(`<input id="url" name="url" type="url" value="" size="30" maxlength="200" autocomplete="url" /></p>` + "\n")
		b.WriteString(`<p class="comment-form-cookies-consent"><input id="wp-comment-cookies-consent" name="wp-comment-cookies-consent" type="checkbox" value="yes" /> `)
		b.WriteString(`<label for="wp-comment-cookies-consent">Save my name, email, and website in this browser for the next time I comment.</label></p>` + "\n")
	}
	b.WriteString(`<p class="form-submit wp-block-button"><input name="submit" type="submit" id="submit" class="wp-block-button__link wp-element-button" value="Post Comment" /> `)
	b.WriteString(`<input type="hidden" name="comment_post_ID" value="`)
	b.WriteString(strconv.FormatInt(post.ID, 10))
	b.WriteString(`" id="comment_post_ID" />` + "\n")
	b.WriteString(`<input type="hidden" name="comment_parent" id="comment_parent" value="0" />` + "\n")
	b.WriteString(`</p></form></div><!-- #respond -->` + "\n")
	return b.String()
}

func (f *Front) renderFeedRSS2(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	desc := optionString(c, "blogdescription", "")
	authorID := feedAuthorFilterID(c)
	posts, err := latestFeedPosts(c, 10, authorID)
	if err != nil {
		http.Error(w, "feed: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}

	// P3.50: atom:link rel=self via get_self_link() (request URI).
	// P3.51: no PubSubHubbub atom:link rel="hub" (out of scope; never emit hub links).
	self := feedSelfLink(r, c)

	var b strings.Builder
	// Match WP feed-rss2.php channel shape (P3.13).
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"` + "\n")
	b.WriteString("\txmlns:content=\"http://purl.org/rss/1.0/modules/content/\"\n")
	b.WriteString("\txmlns:wfw=\"http://wellformedweb.org/CommentAPI/\"\n")
	b.WriteString("\txmlns:dc=\"http://purl.org/dc/elements/1.1/\"\n")
	b.WriteString("\txmlns:atom=\"http://www.w3.org/2005/Atom\"\n")
	b.WriteString("\txmlns:sy=\"http://purl.org/rss/1.0/modules/syndication/\"\n")
	b.WriteString("\txmlns:slash=\"http://purl.org/rss/1.0/modules/slash/\"\n")
	b.WriteString("\t>\n\n")
	b.WriteString("<channel>\n")
	b.WriteString("\t<title>")
	b.WriteString(feedChannelTitle(c, blogname, authorID))
	b.WriteString("</title>\n")
	b.WriteString("\t<atom:link href=\"")
	b.WriteString(xmlEscape(self))
	b.WriteString("\" rel=\"self\" type=\"application/rss+xml\" />\n")
	b.WriteString("\t<link>")
	b.WriteString(xmlEscape(home))
	b.WriteString("</link>\n")
	b.WriteString("\t<description>")
	b.WriteString(xmlEscape(desc))
	b.WriteString("</description>\n")
	b.WriteString("\t<lastBuildDate>")
	b.WriteString(xmlEscape(feedBuildDate(c, posts)))
	b.WriteString("</lastBuildDate>\n")
	b.WriteString("\t<language>")
	b.WriteString(xmlEscape(wp.BlogInfoLanguage(c)))
	b.WriteString("</language>\n")
	// P3.54: optional channel <copyright> when rss_copyright option is set.
	if copy := wp.FeedCopyright(c, feedCopyrightWhen(posts)); copy != "" {
		b.WriteString("\t<copyright>")
		b.WriteString(xmlEscape(copy))
		b.WriteString("</copyright>\n")
	}
	// P3.57: optional channel <ttl> when rss_ttl option is a positive integer.
	if ttl := wp.FeedTTLMinutes(c); ttl != "" {
		b.WriteString("\t<ttl>")
		b.WriteString(xmlEscape(ttl))
		b.WriteString("</ttl>\n")
	}
	b.WriteString("\t<sy:updatePeriod>\n")
	b.WriteString("\thourly\t</sy:updatePeriod>\n")
	b.WriteString("\t<sy:updateFrequency>\n")
	b.WriteString("\t1\t</sy:updateFrequency>\n")
	b.WriteString("\t")
	b.WriteString(wp.GetTheGenerator("rss2"))
	b.WriteString("\n")
	// P3.52: rss2_site_icon channel <image> when a custom site icon is set.
	if img := rss2SiteIconImage(c, blogname, home); img != "" {
		b.WriteString(img)
	}
	for i, p := range posts {
		link := postPermalink(home, p.Post)
		guid := feedItemGUID(home, p.ID, p.GUID)
		// WP feed-rss2.php indents the first item with one tab and later items with two.
		if i == 0 {
			b.WriteString("\t<item>\n")
		} else {
			b.WriteString("\t\t<item>\n")
		}
		b.WriteString("\t\t<title>")
		b.WriteString(xmlEscape(p.Title))
		b.WriteString("</title>\n")
		b.WriteString("\t\t<link>")
		b.WriteString(xmlEscape(link))
		b.WriteString("</link>\n")
		if p.CommentCount > 0 || p.CommentStatus == "open" {
			// P3.61: <comments> via comments_link_feed / get_comments_link.
			b.WriteString("\t\t\t\t\t<comments>")
			b.WriteString(xmlEscape(feedCommentsLink(c, link, p.CommentCount)))
			b.WriteString("</comments>\n")
			b.WriteString("\t\t\n")
		}
		b.WriteString("\t\t<dc:creator><![CDATA[")
		// P3.66: the_author() filter on item creator.
		b.WriteString(feedItemCreator(c, p.AuthorName))
		b.WriteString("]]></dc:creator>\n")
		b.WriteString("\t\t<pubDate>")
		b.WriteString(xmlEscape(feedRFC822(c, p.DateGMT, p.Date)))
		b.WriteString("</pubDate>\n")
		for _, cat := range p.Categories {
			b.WriteString("\t\t\t\t<category><![CDATA[")
			b.WriteString(cat)
			b.WriteString("]]></category>\n")
		}
		// P3.62: isPermaLink=false even when GUID is a pretty permalink URL.
		b.WriteString(rssGUIDTag(guid, link))
		b.WriteString("\t\t\t\t\t<description><![CDATA[")
		b.WriteString(p.ExcerptRSS)
		b.WriteString("]]></description>\n")
		b.WriteString("\t\t\t\t\t\t\t\t\t\t")
		// P3.63: content:encoded when available (skipped when rss_use_excerpt).
		if tag := rss2ContentEncodedTag(c, p); tag != "" {
			b.WriteString(tag)
		}
		b.WriteString("\t\t\t\t\t\n")
		if p.CommentCount > 0 || p.CommentStatus == "open" {
			b.WriteString("\t\t\t\t\t<wfw:commentRss>")
			b.WriteString(xmlEscape(strings.TrimRight(link, "/") + "/feed/"))
			b.WriteString("</wfw:commentRss>\n")
			b.WriteString("\t\t\t<slash:comments>")
			b.WriteString(strconv.FormatInt(p.CommentCount, 10))
			b.WriteString("</slash:comments>\n")
		}
		for _, enc := range p.Enclosures {
			b.WriteString(rssEnclosureTag(c, enc))
		}
		// P3.64: optional RSS 2.0 <source> when post meta provides attribution.
		if tag := rssItemSourceTag(c, p.ID); tag != "" {
			b.WriteString(tag)
		}
		b.WriteString("\t\t\n\t\t\n")
		b.WriteString("\t\t\t</item>\n")
	}
	b.WriteString("\t</channel>\n</rss>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "application/rss+xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderFeedAtom emits a WordPress-shaped Atom feed (feed-atom.php, P3.17).
func (f *Front) renderFeedAtom(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	desc := optionString(c, "blogdescription", "")
	authorID := feedAuthorFilterID(c)
	posts, err := latestFeedPosts(c, 10, authorID)
	if err != nil {
		http.Error(w, "feed: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}

	self := feedSelfLink(r, c)
	atomID := home + "/feed/atom/"

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><feed` + "\n")
	b.WriteString("\txmlns=\"http://www.w3.org/2005/Atom\"\n")
	b.WriteString("\txml:lang=\"")
	b.WriteString(xmlEscape(wp.BlogInfoLanguage(c)))
	b.WriteString("\"\n")
	b.WriteString("\txmlns:thr=\"http://purl.org/syndication/thread/1.0\"\n")
	b.WriteString("\t>\n")
	b.WriteString("\t<title type=\"text\">")
	b.WriteString(feedChannelTitle(c, blogname, authorID))
	b.WriteString("</title>\n")
	b.WriteString("\t<subtitle type=\"text\">")
	b.WriteString(xmlEscape(desc))
	b.WriteString("</subtitle>\n\n")
	b.WriteString("\t<updated>")
	if len(posts) > 0 {
		b.WriteString(xmlEscape(feedAtomDate(c, posts[0].ModifiedGMT, posts[0].Modified)))
	} else {
		b.WriteString(xmlEscape(feedAtomDate(c, "", "")))
	}
	b.WriteString("</updated>\n\n")
	b.WriteString("\t<link rel=\"alternate\" type=\"text/html\" href=\"")
	b.WriteString(xmlEscape(home))
	b.WriteString("\" />\n")
	b.WriteString("\t<id>")
	b.WriteString(xmlEscape(atomID))
	b.WriteString("</id>\n")
	// P3.51: self only — no PubSubHubbub <link rel="hub">.
	b.WriteString("\t<link rel=\"self\" type=\"application/atom+xml\" href=\"")
	b.WriteString(xmlEscape(self))
	b.WriteString("\" />\n\n")
	// P3.54: optional Atom <rights> when rss_copyright option is set.
	if copy := wp.FeedCopyright(c, feedCopyrightWhen(posts)); copy != "" {
		b.WriteString("\t<rights>")
		b.WriteString(xmlEscape(copy))
		b.WriteString("</rights>\n")
	}
	b.WriteString("\t")
	b.WriteString(wp.GetTheGenerator("atom"))
	b.WriteString("\n")

	for _, p := range posts {
		link := postPermalink(home, p.Post)
		guid := feedItemGUID(home, p.ID, p.GUID)
		b.WriteString("\t<entry>\n")
		b.WriteString("\t\t<author>\n")
		b.WriteString("\t\t\t<name>")
		// P3.66: the_author() filter on Atom entry author.
		b.WriteString(xmlEscape(feedItemCreator(c, p.AuthorName)))
		b.WriteString("</name>\n")
		if p.AuthorURL != "" {
			b.WriteString("\t\t\t\t\t\t\t<uri>")
			b.WriteString(xmlEscape(p.AuthorURL))
			b.WriteString("</uri>\n")
			b.WriteString("\t\t\t\t\t\t")
		}
		b.WriteString("</author>\n\n")
		b.WriteString("\t\t<title type=\"html\"><![CDATA[")
		b.WriteString(p.Title)
		b.WriteString("]]></title>\n")
		b.WriteString("\t\t<link rel=\"alternate\" type=\"text/html\" href=\"")
		b.WriteString(xmlEscape(link))
		b.WriteString("\" />\n\n")
		b.WriteString("\t\t<id>")
		b.WriteString(xmlEscape(guid))
		b.WriteString("</id>\n")
		b.WriteString("\t\t<updated>")
		b.WriteString(xmlEscape(feedAtomDate(c, p.ModifiedGMT, p.Modified)))
		b.WriteString("</updated>\n")
		b.WriteString("\t\t<published>")
		b.WriteString(xmlEscape(feedAtomDate(c, p.DateGMT, p.Date)))
		b.WriteString("</published>\n")
		for _, cat := range p.Categories {
			b.WriteString("\t\t<category scheme=\"")
			b.WriteString(xmlEscape(home))
			b.WriteString("\" term=\"")
			b.WriteString(xmlEscape(cat))
			b.WriteString("\" />\n")
		}
		b.WriteString("\t\t<summary type=\"html\"><![CDATA[")
		b.WriteString(p.ExcerptRSS)
		b.WriteString("]]></summary>\n\n")
		b.WriteString("\t\t\t\t\t<content type=\"html\" xml:base=\"")
		b.WriteString(xmlEscape(link))
		b.WriteString("\"><![CDATA[\n")
		b.WriteString(p.ContentRSS)
		if p.ContentRSS != "" && !strings.HasSuffix(p.ContentRSS, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("]]></content>\n")
		b.WriteString("\t\t\n")
		if p.CommentCount > 0 || p.CommentStatus == "open" {
			// P3.61: Atom always uses permalink#comments (feed-atom.php), not #respond.
			b.WriteString("\t\t\t\t\t<link rel=\"replies\" type=\"text/html\" href=\"")
			b.WriteString(xmlEscape(feedAtomCommentsHTMLLink(link)))
			b.WriteString("\" thr:count=\"")
			b.WriteString(strconv.FormatInt(p.CommentCount, 10))
			b.WriteString("\" />\n")
			b.WriteString("\t\t\t<link rel=\"replies\" type=\"application/atom+xml\" href=\"")
			b.WriteString(xmlEscape(strings.TrimRight(link, "/") + "/feed/atom/"))
			b.WriteString("\" thr:count=\"")
			b.WriteString(strconv.FormatInt(p.CommentCount, 10))
			b.WriteString("\" />\n")
			b.WriteString("\t\t\t<thr:total>")
			b.WriteString(strconv.FormatInt(p.CommentCount, 10))
			b.WriteString("</thr:total>\n")
		}
		for _, enc := range p.Enclosures {
			b.WriteString(atomEnclosureTag(c, enc))
		}
		b.WriteString("\t\t\t</entry>\n")
	}
	b.WriteString("</feed>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "application/atom+xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderFeedRDF emits a WordPress-shaped RDF/RSS 1.0 feed stub (feed-rdf.php, P3.33).
func (f *Front) renderFeedRDF(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	desc := optionString(c, "blogdescription", "")
	authorID := feedAuthorFilterID(c)
	posts, err := latestFeedPosts(c, 10, authorID)
	if err != nil {
		http.Error(w, "feed: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString("<rdf:RDF\n")
	b.WriteString("\txmlns=\"http://purl.org/rss/1.0/\"\n")
	b.WriteString("\txmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\"\n")
	b.WriteString("\txmlns:dc=\"http://purl.org/dc/elements/1.1/\"\n")
	b.WriteString("\txmlns:sy=\"http://purl.org/rss/1.0/modules/syndication/\"\n")
	b.WriteString("\txmlns:admin=\"http://webns.net/mvcb/\"\n")
	b.WriteString("\txmlns:content=\"http://purl.org/rss/1.0/modules/content/\"\n")
	b.WriteString("\t>\n")
	b.WriteString("<channel rdf:about=\"")
	b.WriteString(xmlEscape(home))
	b.WriteString("\">\n")
	b.WriteString("\t<title>")
	b.WriteString(feedChannelTitle(c, blogname, authorID))
	b.WriteString("</title>\n")
	b.WriteString("\t<link>")
	b.WriteString(xmlEscape(home))
	b.WriteString("</link>\n")
	b.WriteString("\t<description>")
	b.WriteString(xmlEscape(desc))
	b.WriteString("</description>\n")
	b.WriteString("\t<dc:date>")
	b.WriteString(xmlEscape(feedAtomDate(c, feedBuildDateGMT(posts), "")))
	b.WriteString("\t</dc:date>\n")
	b.WriteString("\t<sy:updatePeriod>\n")
	b.WriteString("\thourly\n")
	b.WriteString("\t</sy:updatePeriod>\n")
	b.WriteString("\t<sy:updateFrequency>\n")
	b.WriteString("\t1\n")
	b.WriteString("\t</sy:updateFrequency>\n")
	b.WriteString("\t<sy:updateBase>2000-01-01T12:00+00:00</sy:updateBase>\n")
	// P3.56: rdf_header → the_generator (admin:generatorAgent).
	b.WriteString("\t")
	b.WriteString(wp.GetTheGenerator("rdf"))
	b.WriteString("\n")
	b.WriteString("\t<items>\n")
	b.WriteString("\t\t<rdf:Seq>\n")
	for _, p := range posts {
		link := postPermalink(home, p.Post)
		b.WriteString("\t\t\t<rdf:li rdf:resource=\"")
		b.WriteString(xmlEscape(link))
		b.WriteString("\"/>\n")
	}
	b.WriteString("\t\t</rdf:Seq>\n")
	b.WriteString("\t</items>\n")
	b.WriteString("</channel>\n")
	for _, p := range posts {
		link := postPermalink(home, p.Post)
		b.WriteString("<item rdf:about=\"")
		b.WriteString(xmlEscape(link))
		b.WriteString("\">\n")
		b.WriteString("\t<title>")
		b.WriteString(xmlEscape(p.Title))
		b.WriteString("</title>\n")
		b.WriteString("\t<link>")
		b.WriteString(xmlEscape(link))
		b.WriteString("</link>\n")
		b.WriteString("\n")
		b.WriteString("\t<dc:creator><![CDATA[")
		// P3.66: the_author() filter on RDF item creator.
		b.WriteString(feedItemCreator(c, p.AuthorName))
		b.WriteString("]]></dc:creator>\n")
		b.WriteString("\t<dc:date>")
		b.WriteString(xmlEscape(feedAtomDate(c, p.DateGMT, p.Date)))
		b.WriteString("</dc:date>\n")
		for _, cat := range p.Categories {
			b.WriteString("\t\t\t\t<dc:subject><![CDATA[")
			b.WriteString(cat)
			b.WriteString("]]></dc:subject>\n")
		}
		b.WriteString("\n")
		b.WriteString("\t\t<description><![CDATA[")
		b.WriteString(p.ExcerptRSS)
		b.WriteString("]]></description>\n")
		// P3.63: RDF content:encoded when available (feed-rdf.php).
		if body, ok := rssContentEncodedBody(c, p); ok {
			b.WriteString("\t\t<content:encoded><![CDATA[")
			b.WriteString(body)
			b.WriteString("]]></content:encoded>\n")
		}
		b.WriteString("</item>\n")
	}
	b.WriteString("</rdf:RDF>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "application/rdf+xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func feedBuildDateGMT(posts []feedPost) string {
	return newestPostDateGMT(posts)
}

// newestPostDateGMT picks the max post_modified_gmt (else post_date_gmt) among
// feed posts — get_feed_build_date / get_lastpostmodified subset (P3.55).
func newestPostDateGMT(posts []feedPost) string {
	best := ""
	for _, p := range posts {
		for _, src := range []string{p.ModifiedGMT, p.DateGMT, p.Modified, p.Date} {
			if src != "" && src > best {
				best = src
			}
		}
	}
	return best
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

type feedEnclosure struct {
	URL    string
	Length string
	Type   string
}

type feedPost struct {
	wp.Post
	AuthorName   string
	AuthorURL    string
	Categories   []string
	CommentCount int64
	ExcerptRSS   string
	ContentRSS   string
	Enclosures   []feedEnclosure
}

func latestFeedPosts(c *wp.Context, limit int, authorID int64) ([]feedPost, error) {
	if limit <= 0 {
		limit = 10
	}
	args := wp.QueryArgs{PostType: "post", PostStatus: "publish", PostsPerPage: limit}
	if authorID > 0 {
		args.Author = authorID
	}
	posts, err := wp.GetPosts(c, args)
	if err != nil {
		return nil, err
	}
	out := make([]feedPost, 0, len(posts))
	for _, p := range posts {
		fp := feedPost{Post: p}
		fp.AuthorName, fp.AuthorURL = feedAuthor(c, p.Author)
		fp.Categories = feedCategoryNames(c, p.ID)
		fp.CommentCount = feedCommentCount(c, p.ID)
		fp.ExcerptRSS = feedExcerptRSS(p)
		// Feed content:encoded follows the_content_feed (no post-save KSES strip),
		// so data: URIs in fixture video blocks are preserved like WP.
		rendered := wp.TheContent(c, p.Content)
		fp.ContentRSS = strings.TrimLeft(rendered, " \t\r")
		if strings.TrimSpace(p.Password) == "" {
			fp.Enclosures = feedEnclosuresForPost(c, p.ID)
		}
		out = append(out, fp)
	}
	return out, nil
}

// rssUseExcerpt reports get_option('rss_use_excerpt') (P3.63).
func rssUseExcerpt(c *wp.Context) bool {
	if c == nil || c.Options == nil {
		return false
	}
	raw := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("rss_use_excerpt", "0")))
	if raw == "" || raw == "<nil>" {
		return false
	}
	return raw != "0" && !strings.EqualFold(raw, "false")
}

// rssContentEncodedBody returns the content:encoded CDATA body when the tag
// should be emitted (P3.63). Omitted when rss_use_excerpt is on. When full
// content is empty, falls back to the excerpt (WP feed-rss2.php).
func rssContentEncodedBody(c *wp.Context, p feedPost) (body string, ok bool) {
	if rssUseExcerpt(c) {
		return "", false
	}
	if len(p.ContentRSS) > 0 {
		return p.ContentRSS, true
	}
	return p.ExcerptRSS, true
}

// rss2ContentEncodedTag emits the RSS2 content:encoded element, or "" when omitted.
func rss2ContentEncodedTag(c *wp.Context, p feedPost) string {
	body, ok := rssContentEncodedBody(c, p)
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString("<content:encoded><![CDATA[")
	if !strings.HasPrefix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("]]></content:encoded>\n")
	return b.String()
}

// feedAuthorFilterID returns the author user ID when the request is an author feed (P3.60).
func feedAuthorFilterID(c *wp.Context) int64 {
	if c == nil || c.Scope == nil || !c.Scope.IsAuthor {
		return 0
	}
	if c.Scope.AuthorID > 0 {
		return c.Scope.AuthorID
	}
	slug := strings.TrimSpace(c.Scope.AuthorName)
	if slug == "" {
		return 0
	}
	u, err := c.GetUserBy("slug", slug)
	if err != nil || u == nil {
		u, err = c.GetUserBy("login", slug)
	}
	if err != nil || u == nil {
		return 0
	}
	c.Scope.AuthorID = u.ID
	return u.ID
}

// feedChannelTitle mirrors wp_title_rss / document title for feeds (P3.60).
// Author feeds: "{Display Name} &#8211; {Blogname}".
func feedChannelTitle(c *wp.Context, blogname string, authorID int64) string {
	if authorID <= 0 {
		return xmlEscape(blogname)
	}
	name, _ := feedAuthor(c, authorID)
	if name == "" {
		return xmlEscape(blogname)
	}
	return xmlEscape(name) + " &#8211; " + xmlEscape(blogname)
}

// feedEnclosuresForPost reads enclosure postmeta (rss_enclosure / atom_enclosure, P3.45 / P3.65).
// Meta value shape is "url\nlength\nmime …"; multiple meta rows each become an enclosure.
func feedEnclosuresForPost(c *wp.Context, postID int64) []feedEnclosure {
	if c == nil || c.Meta == nil || postID <= 0 {
		return nil
	}
	raw := c.Meta.GetMetadata(wp.MetaPost, postID, "enclosure", false)
	vals, ok := raw.([]string)
	if !ok || len(vals) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []feedEnclosure
	for _, enc := range vals {
		enc = strings.ReplaceAll(enc, "\r\n", "\n")
		enc = strings.ReplaceAll(enc, "\r", "\n")
		parts := strings.Split(enc, "\n")
		if len(parts) < 3 {
			continue
		}
		url := strings.TrimSpace(parts[0])
		length := strings.TrimSpace(parts[1])
		typeField := strings.TrimSpace(parts[2])
		if url == "" || typeField == "" {
			continue
		}
		mime := strings.Fields(typeField)
		if len(mime) == 0 {
			continue
		}
		if seen[url] {
			continue
		}
		seen[url] = true
		out = append(out, feedEnclosure{URL: url, Length: length, Type: mime[0]})
	}
	return out
}

func rssEnclosureTag(c *wp.Context, enc feedEnclosure) string {
	length := enc.Length
	if length == "" {
		length = "0"
	}
	tag := `<enclosure url="` + xmlEscape(enc.URL) + `" length="` + xmlEscape(length) + `" type="` + xmlEscape(enc.Type) + `" />` + "\n"
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("rss_enclosure", tag); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return tag
}

// feedItemSourceURLTitle reads optional RSS <source> attribution from post meta (P3.64).
// Keys: _rss_source_url (required) and _rss_source_title (optional; falls back to URL).
func feedItemSourceURLTitle(c *wp.Context, postID int64) (url, title string) {
	if c == nil || c.Meta == nil || postID <= 0 {
		return "", ""
	}
	url = strings.TrimSpace(asMetaString(c.Meta.GetMetadata(wp.MetaPost, postID, "_rss_source_url", true)))
	if url == "" {
		return "", ""
	}
	title = strings.TrimSpace(asMetaString(c.Meta.GetMetadata(wp.MetaPost, postID, "_rss_source_title", true)))
	if title == "" {
		title = url
	}
	return url, title
}

// rssItemSourceTag builds RSS 2.0 <source url="...">title</source> when meta is set (P3.64).
func rssItemSourceTag(c *wp.Context, postID int64) string {
	url, title := feedItemSourceURLTitle(c, postID)
	if url == "" {
		return ""
	}
	return `<source url="` + xmlEscape(url) + `">` + xmlEscape(title) + `</source>` + "\n"
}

func atomEnclosureTag(c *wp.Context, enc feedEnclosure) string {
	length := enc.Length
	if length == "" {
		length = "0"
	}
	tag := fmt.Sprintf("<link href=\"%s\" rel=\"enclosure\" length=\"%s\" type=\"%s\" />\n",
		xmlEscape(enc.URL), xmlEscape(length), xmlEscape(enc.Type))
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("atom_enclosure", tag); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return tag
}

func feedBuildDate(c *wp.Context, posts []feedPost) string {
	// P3.55: lastBuildDate from newest post modified/publish time — never wall clock.
	gmt := newestPostDateGMT(posts)
	if gmt == "" {
		return ""
	}
	return feedRFC822(c, gmt, "")
}

// feedRFC822 formats a MySQL DATETIME as RSS pubDate (P3.55 / P3.67).
// Prefer post_date_gmt as UTC (+0000). When GMT is empty/zero, convert local
// via the site timezone (timezone_string / gmt_offset) before emitting.
func feedRFC822(c *wp.Context, gmt, local string) string {
	if t, ok := feedParseUTC(c, gmt, local); ok {
		return t.UTC().Format("Mon, 02 Jan 2006 15:04:05 +0000")
	}
	// Stable fallback when timestamps are unparseable (P3.55).
	return ""
}

func feedParseUTC(c *wp.Context, gmt, local string) (time.Time, bool) {
	if t, ok := parseFeedMySQLUTC(gmt); ok {
		return t, true
	}
	local = strings.TrimSpace(local)
	if local == "" || local == "0000-00-00 00:00:00" {
		return time.Time{}, false
	}
	if tz := feedTimezoneName(c); tz != "" {
		if converted, err := wp.GetGMTFromDate(local, tz, "Y-m-d H:i:s"); err == nil {
			if t, ok := parseFeedMySQLUTC(converted); ok {
				return t, true
			}
		}
	}
	if offset := feedGMTOffsetHours(c); offset != 0 {
		zone := time.FixedZone("gmt_offset", int(offset*3600))
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", local, zone); err == nil {
			return t.UTC(), true
		}
	}
	// Last resort when site TZ is UTC / unset: treat local as UTC.
	return parseFeedMySQLUTC(local)
}

func parseFeedMySQLUTC(src string) (time.Time, bool) {
	src = strings.TrimSpace(src)
	if src == "" || src == "0000-00-00 00:00:00" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", src, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func feedTimezoneName(c *wp.Context) string {
	if c == nil || c.Options == nil {
		return ""
	}
	return strings.TrimSpace(optionString(c, "timezone_string", ""))
}

func feedGMTOffsetHours(c *wp.Context) float64 {
	if c == nil || c.Options == nil {
		return 0
	}
	raw := strings.TrimSpace(optionString(c, "gmt_offset", "0"))
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return n
}

func feedAuthor(c *wp.Context, authorID int64) (name, url string) {
	if c == nil || authorID <= 0 {
		return "", ""
	}
	u, err := c.GetUserBy("id", authorID)
	if err != nil || u == nil {
		return "", ""
	}
	// get_the_author() → display_name; fall back to login then nicename (P3.60).
	switch {
	case strings.TrimSpace(u.DisplayName) != "":
		name = u.DisplayName
	case strings.TrimSpace(u.Login) != "":
		name = u.Login
	case strings.TrimSpace(u.Nicename) != "":
		name = u.Nicename
	default:
		name = "Anonymous"
	}
	return name, u.URL
}

// feedItemCreator applies the_author for RSS/Atom/RDF item creators (P3.66).
func feedItemCreator(c *wp.Context, name string) string {
	if c == nil || c.Hooks == nil {
		return name
	}
	if v := c.Hooks.ApplyFilters("the_author", name); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return name
}

// feedAtomDate formats a MySQL DATETIME as Atom date (P3.55 / P3.67).
func feedAtomDate(c *wp.Context, gmt, local string) string {
	if t, ok := feedParseUTC(c, gmt, local); ok {
		return t.UTC().Format("2006-01-02T15:04:05Z")
	}
	// Stable fallback (P3.55).
	return ""
}

func feedSelfLink(r *http.Request, c *wp.Context) string {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	if r == nil || r.URL == nil {
		return home + "/feed/"
	}
	// Prefer absolute URL matching the request (WP get_self_link / self_link).
	path := r.URL.RequestURI()
	if path == "" {
		path = "/"
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return home + path
}

// feedCopyrightWhen picks a UTC time for auto copyright year (P3.54).
func feedCopyrightWhen(posts []feedPost) time.Time {
	if len(posts) == 0 {
		return time.Now().UTC()
	}
	raw := posts[0].DateGMT
	if raw == "" {
		raw = posts[0].Date
	}
	if t, err := time.Parse("2006-01-02 15:04:05", raw); err == nil {
		return t.UTC()
	}
	return time.Now().UTC()
}

// rss2SiteIconImage emits the RSS2 channel <image> block (rss2_site_icon, P3.52).
// Empty when no custom site icon is configured (matches get_site_icon_url fallback "").
func rss2SiteIconImage(c *wp.Context, title, home string) string {
	url := wp.GetSiteIconURL(c, 32, "")
	if url == "" {
		return ""
	}
	if strings.TrimSpace(title) == "" {
		title = optionString(c, "blogname", "GoPress")
	}
	var b strings.Builder
	b.WriteString("\n<image>\n")
	b.WriteString("\t<url>")
	b.WriteString(xmlEscape(url))
	b.WriteString("</url>\n")
	b.WriteString("\t<title>")
	b.WriteString(xmlEscape(title))
	b.WriteString("</title>\n")
	b.WriteString("\t<link>")
	b.WriteString(xmlEscape(home))
	b.WriteString("</link>\n")
	b.WriteString("\t<width>32</width>\n")
	b.WriteString("\t<height>32</height>\n")
	b.WriteString("</image> \n")
	return b.String()
}

// feedCategoryNames returns category + tag names for a post (get_the_category_rss, P3.59).
func feedCategoryNames(c *wp.Context, postID int64) []string {
	out := []string{}
	if c == nil || c.DB == nil || postID <= 0 {
		return out
	}
	// WP get_the_category_rss merges get_the_category() and get_the_tags().
	q, args, err := c.DB.Prepare(
		"SELECT t.name FROM "+c.DB.Terms()+" t "+
			"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
			"INNER JOIN "+c.DB.TermRelationships()+" tr ON tr.term_taxonomy_id = tt.term_taxonomy_id "+
			"WHERE tr.object_id = %d AND tt.taxonomy IN (%s, %s) ORDER BY t.name ASC",
		postID, "category", "post_tag",
	)
	if err != nil {
		return out
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return out
	}
	seen := map[string]bool{}
	for _, row := range rows {
		name := ""
		switch v := row["name"].(type) {
		case string:
			name = v
		case []byte:
			name = string(v)
		}
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// feedCommentsLink builds the RSS <comments> URL (comments_link_feed / get_comments_link, P3.61).
// Appends #comments when the post has comments, else #respond; runs comments_link_feed filter.
// feedItemGUID resolves the RSS/Atom item guid value (get_the_guid).
// Prefers the stored post GUID (often a pretty permalink URL on fixture/import
// data); otherwise falls back to home/?p={id}.
func feedItemGUID(home string, id int64, storedGUID string) string {
	if g := strings.TrimSpace(storedGUID); g != "" {
		return g
	}
	home = strings.TrimRight(home, "/")
	return home + "/?p=" + strconv.FormatInt(id, 10)
}

// feedGUIDIsPermaLink reports the isPermaLink attribute for RSS <guid>.
// WP feed-rss2.php hardcodes false even when the GUID is a pretty permalink URL
// (P3.62).
func feedGUIDIsPermaLink(guid, permalink string) bool {
	_ = guid
	_ = permalink
	return false
}

// rssGUIDTag emits <guid isPermaLink="…">value</guid> for RSS items (P3.62).
func rssGUIDTag(guid, permalink string) string {
	attr := "false"
	if feedGUIDIsPermaLink(guid, permalink) {
		attr = "true"
	}
	return "\t\t<guid isPermaLink=\"" + attr + "\">" + xmlEscape(guid) + "</guid>\n\n"
}

func feedCommentsLink(c *wp.Context, permalink string, commentCount int64) string {
	hash := "#respond"
	if commentCount > 0 {
		hash = "#comments"
	}
	link := permalink + hash
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("comments_link_feed", link); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return link
}

// feedAtomCommentsHTMLLink is the Atom HTML replies href (always #comments, P3.61 / feed-atom.php).
func feedAtomCommentsHTMLLink(permalink string) string {
	return permalink + "#comments"
}

func feedCommentCount(c *wp.Context, postID int64) int64 {
	if c == nil || c.DB == nil || postID <= 0 {
		return 0
	}
	q, args, err := c.DB.Prepare(
		"SELECT comment_count FROM "+c.DB.Posts()+" WHERE ID = %d LIMIT 1",
		postID,
	)
	if err != nil {
		return 0
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		return 0
	}
	switch v := row["comment_count"].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case []byte:
		n, _ := strconv.ParseInt(string(v), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}
	return 0
}

func feedExcerptRSS(p wp.Post) string {
	if strings.TrimSpace(p.Excerpt) != "" {
		return strings.TrimSpace(stripTagsSimple(p.Excerpt))
	}
	// Approximate WP excerpt_remove_blocks: drop media blocks, keep textual blocks.
	raw := p.Content
	raw = stripBlockType(raw, "video")
	raw = stripBlockType(raw, "audio")
	raw = stripBlockType(raw, "image")
	raw = stripBlockType(raw, "gallery")
	raw = stripBlockType(raw, "embed")
	raw = stripBlockType(raw, "text-columns")
	raw = stripBlockType(raw, "columns")
	raw = stripBlockType(raw, "terms-query")
	raw = stripBlockType(raw, "query")
	for {
		start := strings.Index(raw, "<!--")
		if start < 0 {
			break
		}
		end := strings.Index(raw[start:], "-->")
		if end < 0 {
			break
		}
		raw = raw[:start] + raw[start+end+3:]
	}
	text := strings.TrimSpace(stripTagsSimple(raw))
	text = strings.Join(strings.Fields(text), " ")
	return text
}

// stripBlockType removes <!-- wp:name ... --> ... <!-- /wp:name --> regions (and self-closing).
func stripBlockType(s, name string) string {
	for {
		open := "<!-- wp:" + name
		i := strings.Index(s, open)
		if i < 0 {
			// also core/name form is already "wp:name"
			return s
		}
		rest := s[i:]
		selfClose := strings.Index(rest, "/-->")
		close := strings.Index(rest, "<!-- /wp:"+name+" -->")
		if selfClose >= 0 && (close < 0 || selfClose < close) {
			// self-closing block comment
			end := i + selfClose + len("/-->")
			s = s[:i] + s[end:]
			continue
		}
		if close < 0 {
			return s
		}
		end := i + close + len("<!-- /wp:"+name+" -->")
		s = s[:i] + s[end:]
	}
}

func stripTagsSimple(s string) string {
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
	return b.String()
}

// tryTrailingSlashRedirect issues a 301 when a pretty content path is missing
// the trailing slash expected by permalink_structure (WP redirect_canonical subset).
// P3.38: also covers /page/N home pagination and emits an absolute Location.
func (f *Front) tryTrailingSlashRedirect(w http.ResponseWriter, r *http.Request, c *wp.Context, parsed *wp.ParseResult) bool {
	if r == nil || r.URL == nil || parsed == nil {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	path := r.URL.Path
	if path == "" || path == "/" || strings.HasSuffix(path, "/") {
		return false
	}
	// Query-style URLs (?p=, ?page_id=) do not use trailing slashes.
	if r.URL.RawQuery != "" && (strings.Contains(r.URL.RawQuery, "p=") ||
		strings.Contains(r.URL.RawQuery, "page_id=") ||
		strings.Contains(r.URL.RawQuery, "attachment_id=") ||
		strings.Contains(r.URL.RawQuery, "feed=")) {
		return false
	}
	if !permalinkUsesTrailingSlash(c) {
		return false
	}
	if !needsTrailingSlashCanonical(parsed) {
		return false
	}
	target := path + "/"
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, absoluteCanonicalURL(c, target), http.StatusMovedPermanently)
	return true
}

// absoluteCanonicalURL prefixes a site-relative redirect with home (P3.38).
func absoluteCanonicalURL(c *wp.Context, target string) string {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	if home == "" {
		return target
	}
	if !strings.HasPrefix(target, "/") {
		target = "/" + target
	}
	return home + target
}

func permalinkUsesTrailingSlash(c *wp.Context) bool {
	if c == nil || c.Options == nil {
		return true
	}
	structure := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("permalink_structure", "")))
	if structure == "" {
		return true
	}
	return strings.HasSuffix(structure, "/")
}

func needsTrailingSlashCanonical(parsed *wp.ParseResult) bool {
	if parsed == nil {
		return false
	}
	// Bare home "/" never needs a trailing-slash redirect.
	if parsed.IsHome {
		// P3.38: /page/N (home pagination) should redirect to /page/N/.
		if parsed.QueryVars != nil && parsed.QueryVars["paged"] != "" {
			return true
		}
		return false
	}
	if isSingularVars(parsed.QueryVars) || isFeedVars(parsed.QueryVars) {
		return true
	}
	for _, k := range []string{"category_name", "tag", "year", "monthnum", "author_name", "paged"} {
		if parsed.QueryVars[k] != "" {
			return true
		}
	}
	// Matched pretty request path with a rewrite rule (non-empty Request).
	return parsed.Request != "" && parsed.MatchedRule != ""
}

// tryCanonicalPostRedirect issues a 301 to the post's pretty permalink when the
// last path segment matches a published post name (WP redirect_canonical subset).
func (f *Front) tryCanonicalPostRedirect(w http.ResponseWriter, r *http.Request, c *wp.Context, request string) bool {
	parts := strings.Split(strings.Trim(request, "/"), "/")
	if len(parts) == 0 {
		return false
	}
	slug := parts[len(parts)-1]
	if slug == "" {
		return false
	}
	post, err := wp.GetPost(c, wp.QueryArgs{Name: slug, PostType: "post", PostStatus: "publish"})
	if err != nil || post == nil {
		return false
	}
	// Prefer date permalink when post_date is available.
	target := ""
	if post.Date != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", post.Date); err == nil {
			target = fmt.Sprintf("/%04d/%02d/%02d/%s/", t.Year(), int(t.Month()), t.Day(), post.Name)
		}
	}
	if target == "" {
		target = "/?p=" + strconv.FormatInt(post.ID, 10)
	}
	http.Redirect(w, r, absoluteCanonicalURL(c, target), http.StatusMovedPermanently)
	return true
}

func (f *Front) render404(w http.ResponseWriter, c *wp.Context) {
	// P3.16: prefer block theme templates/404.html when present.
	// P3.35: ensure header/footer template parts wrap the 404 content.
	if tpl, ok := wp.ResolveBlockTemplate(c, "404"); ok && strings.TrimSpace(tpl) != "" {
		f.render404BlockTemplate(w, c, Ensure404TemplateParts(tpl))
		return
	}
	// Fall back: still wrap shim content with theme header/footer parts when available.
	if wrapped, ok := wrap404ShimWithParts(c); ok {
		f.render404BlockTemplate(w, c, wrapped)
		return
	}
	f.render404Shim(w, c)
}

// Ensure404TemplateParts prepends/appends core/template-part header and footer
// blocks when the resolved 404 template omits them (P3.35).
func Ensure404TemplateParts(template string) string {
	tpl := strings.TrimSpace(template)
	if tpl == "" {
		return `<!-- wp:template-part {"slug":"header"} /-->` + "\n\n" +
			`<!-- wp:paragraph --><p>Page not found</p><!-- /wp:paragraph -->` + "\n\n" +
			`<!-- wp:template-part {"slug":"footer"} /-->`
	}
	hasHeader := strings.Contains(tpl, `"slug":"header"`) || strings.Contains(tpl, `"slug": "header"`)
	hasFooter := strings.Contains(tpl, `"slug":"footer"`) || strings.Contains(tpl, `"slug": "footer"`)
	if hasHeader && hasFooter {
		return tpl
	}
	var b strings.Builder
	if !hasHeader {
		b.WriteString(`<!-- wp:template-part {"slug":"header"} /-->`)
		b.WriteString("\n\n")
	}
	b.WriteString(tpl)
	if !hasFooter {
		b.WriteString("\n\n")
		b.WriteString(`<!-- wp:template-part {"slug":"footer"} /-->`)
	}
	return b.String()
}

// wrap404ShimWithParts builds a minimal 404 block document using theme parts.
func wrap404ShimWithParts(c *wp.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	_, hasHeader := wp.ResolveTemplatePart(c, "header")
	_, hasFooter := wp.ResolveTemplatePart(c, "footer")
	if !hasHeader && !hasFooter {
		return "", false
	}
	return Ensure404TemplateParts(
		`<!-- wp:group {"tagName":"main","layout":{"type":"constrained"}} -->` +
			`<main class="wp-block-group"><!-- wp:heading {"level":1} --><h1 class="wp-block-heading">Page not found</h1><!-- /wp:heading -->` +
			`<!-- wp:paragraph --><p>The requested content could not be found.</p><!-- /wp:paragraph --></main><!-- /wp:group -->`,
	), true
}

func (f *Front) render404BlockTemplate(w http.ResponseWriter, c *wp.Context, template string) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPost(body))
	safe = ensureMainSkipTarget(safe)

	bodyClasses := "error404 wp-embed-responsive"
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>Page not found &#8211; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "", "", "", "")
	b.WriteString(wp.EmojiHeadHTML())
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	b.WriteString("</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) render404Shim(w http.ResponseWriter, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Page not found &#8212; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "error404")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1>Page not found</h1>\n")
	b.WriteString("<p>The requested content could not be found.</p>\n")
	b.WriteString("</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(b.String()))
}

// pathEscapeSearch encodes a search term for /search/{term}/feed/ paths (P3.39).
func pathEscapeSearch(term string) string {
	return url.PathEscape(strings.TrimSpace(term))
}

// writeArchiveRelLinks emits <link rel="prev|next"> on paginated archives (P3.43).
// Not used on the blog home (corpus front-home) — only is_archive()-style views.
func writeArchiveRelLinks(b *strings.Builder, c *wp.Context, home string) {
	if b == nil || c == nil || c.Scope == nil {
		return
	}
	if !c.Scope.IsArchive && !c.Scope.IsSearch {
		return
	}
	base := wp.ArchiveBaseFromRequest(c, home)
	paged, maxPages := mainQueryPagination(c)
	b.WriteString(wp.ArchiveRelPrevNextHTML(base, paged, maxPages))
}

// frontRobotsMetaTag returns the <meta name='robots' …> line (wp_robots subset, P3.44).
// When blog_public=0: noindex, nofollow. Otherwise: max-image-preview:large.
func frontRobotsMetaTag(c *wp.Context) string {
	if !optionTruthy(c, "blog_public") {
		return "<meta name='robots' content='noindex, nofollow' />\n"
	}
	return "<meta name='robots' content='max-image-preview:large' />\n"
}

// writeFrontHeadLinks emits wp_head discovery links (feeds, api.w.org, oEmbed, canonical).
// Stylesheet <link> tags are omitted: the HTML normalizer strips them, and TT5 enqueues
// styles as inline <style> which we also normalize away (P5.19).
// extraFeedTitle/Href implement feed_links_extra for author/search archives (P3.39).
func writeFrontHeadLinks(b *strings.Builder, home, blogname, permalink, restPath, title, extraFeedTitle, extraFeedHref, shortlink string) {
	if home == "" || b == nil {
		return
	}
	b.WriteString("<link rel=\"alternate\" type=\"application/rss+xml\" title=\"")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &raquo; Feed\" href=\"")
	b.WriteString(html.EscapeString(home + "/feed/"))
	b.WriteString("\" />\n")
	b.WriteString("<link rel=\"alternate\" type=\"application/rss+xml\" title=\"")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &raquo; Comments Feed\" href=\"")
	b.WriteString(html.EscapeString(home + "/comments/feed/"))
	b.WriteString("\" />\n")
	if permalink != "" && title != "" {
		b.WriteString("<link rel=\"alternate\" type=\"application/rss+xml\" title=\"")
		b.WriteString(html.EscapeString(blogname))
		b.WriteString(" &raquo; ")
		b.WriteString(html.EscapeString(title))
		b.WriteString(" Comments Feed\" href=\"")
		b.WriteString(html.EscapeString(strings.TrimRight(permalink, "/") + "/feed/"))
		b.WriteString("\" />\n")
	}
	if extraFeedTitle != "" && extraFeedHref != "" {
		b.WriteString("<link rel=\"alternate\" type=\"application/rss+xml\" title=\"")
		b.WriteString(extraFeedTitle) // may include &#8220; entities like WP
		b.WriteString("\" href=\"")
		b.WriteString(html.EscapeString(extraFeedHref))
		b.WriteString("\" />\n")
	}
	// P3.20: wp_oembed_add_discovery_links — emit on singular permalinks.
	b.WriteString(wp.OEmbedDiscoveryLinks(home, permalink))
	b.WriteString("<link rel=\"https://api.w.org/\" href=\"")
	b.WriteString(html.EscapeString(home + "/wp-json/"))
	b.WriteString("\" />\n")
	if restPath != "" {
		b.WriteString("<link rel=\"alternate\" title=\"JSON\" type=\"application/json\" href=\"")
		b.WriteString(html.EscapeString(home + "/wp-json" + restPath))
		b.WriteString("\" />\n")
	}
	b.WriteString("<link rel=\"EditURI\" type=\"application/rsd+xml\" title=\"RSD\" href=\"")
	b.WriteString(html.EscapeString(home + "/xmlrpc.php?rsd"))
	b.WriteString("\" />\n")
	b.WriteString(wp.MetaGeneratorXHTML())
	b.WriteString("\n")
	// P3.42: rel=canonical (wp_get_canonical_url / rel_canonical) on singular.
	if permalink != "" {
		b.WriteString("<link rel=\"canonical\" href=\"")
		b.WriteString(html.EscapeString(permalink))
		b.WriteString("\" />\n")
	}
	if shortlink != "" {
		b.WriteString("<link rel='shortlink' href='")
		b.WriteString(html.EscapeString(shortlink))
		b.WriteString("' />\n")
	}
}

func ensureMainSkipTarget(htmlBody string) string {
	if strings.Contains(htmlBody, `id="wp--skip-link--target"`) || strings.Contains(htmlBody, `id='wp--skip-link--target'`) {
		return htmlBody
	}
	if idx := strings.Index(htmlBody, "<main"); idx >= 0 {
		return htmlBody[:idx] + `<main id="wp--skip-link--target"` + htmlBody[idx+len("<main"):]
	}
	return htmlBody
}

// setFrontAPILink mirrors rest_output_link_header (rel=https://api.w.org/).
func setFrontAPILink(w http.ResponseWriter, c *wp.Context) {
	siteURL := strings.TrimRight(optionString(c, "siteurl", ""), "/")
	home := strings.TrimRight(optionString(c, "home", siteURL), "/")
	if home == "" {
		home = siteURL
	}
	if home == "" {
		return
	}
	w.Header().Set("Link", fmt.Sprintf(`<%s/wp-json/>; rel="https://api.w.org/"`, home))
}

// setFrontSingularLinks mirrors WP Link headers for singular views (P5.17).
// X-Pingback is intentionally omitted: XML-RPC/pingbacks are out of scope (P3.41).
func setFrontSingularLinks(w http.ResponseWriter, c *wp.Context, post *singularPost) {
	siteURL := strings.TrimRight(optionString(c, "siteurl", ""), "/")
	home := strings.TrimRight(optionString(c, "home", siteURL), "/")
	if home == "" {
		home = siteURL
	}
	if home == "" || post == nil {
		return
	}
	// Never advertise xmlrpc.php via X-Pingback (gone with Tier-2 XML-RPC).
	w.Header().Del("X-Pingback")
	w.Header().Add("Link", fmt.Sprintf(`<%s/wp-json/>; rel="https://api.w.org/"`, home))
	restType := "posts"
	if post.Type == "page" {
		restType = "pages"
	}
	if post.Type == "post" || post.Type == "page" {
		w.Header().Add("Link", fmt.Sprintf(
			`<%s/wp-json/wp/v2/%s/%d>; rel="alternate"; title="JSON"; type="application/json"`,
			home, restType, post.ID))
		if short := wp.GetShortlink(c, post.ID, "post", 0); short != "" {
			w.Header().Add("Link", fmt.Sprintf(`<%s>; rel=shortlink`, short))
		}
	}
}

type postListItem struct {
	ID    int64
	Title string
}

type singularPost struct {
	ID               int64
	Title            string
	Content          string
	Type             string
	Name             string
	Status           string
	Date             string
	Password         string
	FeaturedImageURL string
	FeaturedImageAlt string
}

func loadSingularPost(c *wp.Context, vars map[string]string) (*singularPost, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if idStr := vars["attachment_id"]; idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			return nil, nil
		}
		return singularFromQuery(c, wp.QueryArgs{P: id, PostType: "attachment", PostStatus: "inherit"})
	}
	if slug := vars["attachment"]; slug != "" {
		return singularFromQuery(c, wp.QueryArgs{Name: slug, PostType: "attachment", PostStatus: "inherit"})
	}
	if idStr := vars["p"]; idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			return nil, nil
		}
		return singularFromQuery(c, wp.QueryArgs{P: id, PostType: "post"})
	}
	if idStr := vars["page_id"]; idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			return nil, nil
		}
		return singularFromQuery(c, wp.QueryArgs{P: id, PostType: "page"})
	}
	if name := vars["name"]; name != "" {
		return singularFromQuery(c, wp.QueryArgs{Name: name, PostType: "post"})
	}
	if pagename := vars["pagename"]; pagename != "" {
		return singularFromQuery(c, wp.QueryArgs{PageName: pagename, PostType: "page"})
	}
	return nil, nil
}

func singularFromQuery(c *wp.Context, args wp.QueryArgs) (*singularPost, error) {
	p, err := wp.GetPost(c, args)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	thumbURL := wp.GetPostThumbnailURL(c, p.ID)
	alt := ""
	if thumbID := wp.GetPostThumbnailID(c, p.ID); thumbID > 0 && c.Meta != nil {
		alt = asMetaString(c.Meta.GetMetadata(wp.MetaPost, thumbID, "_wp_attachment_image_alt", true))
	}
	return &singularPost{
		ID:               p.ID,
		Title:            p.Title,
		Content:          p.Content,
		Type:             p.Type,
		Name:             p.Name,
		Status:           p.Status,
		Date:             p.Date,
		Password:         p.Password,
		FeaturedImageURL: thumbURL,
		FeaturedImageAlt: alt,
	}, nil
}

func asMetaString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		s := fmt.Sprint(t)
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	}
}

func latestPosts(c *wp.Context, limit int) ([]postListItem, error) {
	if limit <= 0 {
		limit = 5
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostType: "post", PostStatus: "publish", PostsPerPage: limit})
	if err != nil {
		return nil, err
	}
	out := make([]postListItem, 0, len(posts))
	for _, p := range posts {
		out = append(out, postListItem{ID: p.ID, Title: p.Title})
	}
	return out, nil
}

// mainQueryPagination returns paged + max_num_pages for inherit query blocks (P5.44).
func mainQueryPagination(c *wp.Context) (paged, maxPages int) {
	paged = 1
	if c != nil && c.Scope != nil && c.Scope.Paged > 1 {
		paged = c.Scope.Paged
	}
	ppp := 10
	if v := optionString(c, "posts_per_page", "10"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ppp = n
		}
	}
	args := wp.QueryArgs{PostType: "post", PostStatus: "publish"}
	if c != nil && c.Scope != nil {
		if c.Scope.IsSearch {
			args.Search = c.Scope.SearchQuery
		}
		if c.Scope.IsDate {
			args.Year = c.Scope.DateYear
			args.MonthNum = c.Scope.DateMonth
			args.Day = c.Scope.DateDay
		}
		if c.Scope.IsAuthor {
			args.Author = c.Scope.AuthorID
		}
		if c.Scope.IsTag {
			args.Tag = c.Scope.TagSlug
		}
		if c.Scope.IsCategory {
			args.Category = c.Scope.CategorySlug
		}
	}
	total, err := wp.CountPosts(c, args)
	if err != nil || total <= 0 {
		maxPages = 1
		return paged, maxPages
	}
	maxPages = (total + ppp - 1) / ppp
	if maxPages < 1 {
		maxPages = 1
	}
	return paged, maxPages
}

func optionString(c *wp.Context, name, fallback string) string {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case string:
		if t == "" {
			return fallback
		}
		return t
	default:
		s := fmt.Sprint(t)
		if s == "" || s == "<nil>" {
			return fallback
		}
		return s
	}
}

// blockRenderEnv wires dynamic block resolvers (template parts, navigation, query).
func blockRenderEnv(c *wp.Context) *blocks.Env {
	paged, maxPages := mainQueryPagination(c)
	return &blocks.Env{
		Paged:    paged,
		MaxPages: maxPages,
		GetAttachmentURL: func(id int64) string {
			return wp.GetAttachmentURL(c, id)
		},
		LoadTemplatePart: func(slug string) (string, bool) {
			return wp.ResolveTemplatePart(c, slug)
		},
		LoadNavigation: func(ref int64) (string, bool) {
			return wp.LoadNavigationContent(c, ref)
		},
		ListPages: func() []blocks.NavLink {
			pages, err := wp.ListPublishedPages(c, 50)
			if err != nil {
				return nil
			}
			out := make([]blocks.NavLink, 0, len(pages))
			for _, p := range pages {
				out = append(out, blocks.NavLink{
					Label: p.Title,
					URL:   wp.PagePermalink(c, p),
				})
			}
			return out
		},
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			limit := q.PerPage
			if limit <= 0 {
				limit = 10
			}
			// WP: inherit:true uses the main query's posts_per_page (P5.17).
			if q.Inherit {
				if v := optionString(c, "posts_per_page", "10"); v != "" {
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						limit = n
					}
				}
			}
			postType := q.PostType
			if postType == "" {
				postType = "post"
			}
			search := ""
			year, month, day := 0, 0, 0
			var authorID int64
			// Inherit search results on /?s= (P3.21).
			if q.Inherit && c != nil && c.Scope != nil && c.Scope.IsSearch {
				search = c.Scope.SearchQuery
			}
			// Inherit date archive filters (P3.22).
			if q.Inherit && c != nil && c.Scope != nil && c.Scope.IsDate {
				year = c.Scope.DateYear
				month = c.Scope.DateMonth
				day = c.Scope.DateDay
			}
			// Inherit author archive filters (P3.23).
			if q.Inherit && c != nil && c.Scope != nil && c.Scope.IsAuthor {
				authorID = c.Scope.AuthorID
			}
			tagSlug := ""
			catSlug := ""
			// Inherit tag archive filters (P3.24).
			if q.Inherit && c != nil && c.Scope != nil && c.Scope.IsTag {
				tagSlug = c.Scope.TagSlug
			}
			// Inherit category archive filters (P3.25).
			if q.Inherit && c != nil && c.Scope != nil && c.Scope.IsCategory {
				catSlug = c.Scope.CategorySlug
			}
			offset := q.Offset
			paged := 1
			if q.Inherit && c != nil && c.Scope != nil && c.Scope.Paged > 1 {
				paged = c.Scope.Paged
				offset = (paged-1)*limit + q.Offset
			}
			// Sticky posts are prepended on page 1 only (WP main query / P3.26).
			stickyFirst := postType == "post" && search == "" && year == 0 && authorID == 0 && tagSlug == "" && catSlug == "" && paged <= 1
			posts, err := wp.GetPosts(c, wp.QueryArgs{
				PostType:     postType,
				PostStatus:   "publish",
				PostsPerPage: limit,
				Offset:       offset,
				Search:       search,
				Year:         year,
				MonthNum:     month,
				Day:          day,
				Author:       authorID,
				Tag:          tagSlug,
				Category:     catSlug,
				StickyFirst:  stickyFirst,
			})
			if err != nil {
				return nil
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			out := make([]blocks.QueryPost, 0, len(posts))
			for _, p := range posts {
				thumbURL := wp.GetPostThumbnailURL(c, p.ID)
				alt := ""
				if thumbID := wp.GetPostThumbnailID(c, p.ID); thumbID > 0 && c.Meta != nil {
					alt = asMetaString(c.Meta.GetMetadata(wp.MetaPost, thumbID, "_wp_attachment_image_alt", true))
				}
				classes := wp.GetPostClass(c, p.ID)
				if q.Inherit {
					classes = wp.AppendStickyClass(c, p.ID, classes)
				}
				out = append(out, blocks.QueryPost{
					ID:               p.ID,
					Title:            p.Title,
					Content:          p.Content,
					Excerpt:          p.Excerpt,
					Date:             p.Date,
					Link:             postPermalink(home, p),
					FeaturedImageURL: thumbURL,
					FeaturedImageAlt: alt,
					ClassList:        classes,
				})
			}
			return out
		},
		LoadPattern: func(slug string) (string, bool) {
			if err := wp.EnsurePatterns(c); err != nil {
				return "", false
			}
			if c.Patterns == nil {
				return "", false
			}
			p, ok := c.Patterns.Get(slug)
			if !ok {
				return "", false
			}
			return p.Content, true
		},
		LoadBlock: func(ref int64) (string, bool) {
			if c == nil || ref <= 0 {
				return "", false
			}
			row, err := wp.GetPost(c, wp.QueryArgs{P: ref, PostType: "wp_block", PostStatus: "publish"})
			if err != nil || row == nil {
				return "", false
			}
			if strings.TrimSpace(row.Password) != "" {
				return "", false
			}
			return row.Content, true
		},
		LoadFootnotes: func(postID int64) string {
			if c == nil || c.Meta == nil || postID <= 0 {
				return ""
			}
			return asMetaString(c.Meta.GetMetadata(wp.MetaPost, postID, "footnotes", true))
		},
		GetPostMeta: func(postID int64, key string) string {
			if c == nil || c.Meta == nil || postID <= 0 || key == "" {
				return ""
			}
			if strings.HasPrefix(key, "_") {
				return ""
			}
			return asMetaString(c.Meta.GetMetadata(wp.MetaPost, postID, key, true))
		},
		RenderCommentForm: func(postID int64) string {
			if postID <= 0 {
				return ""
			}
			return commentFormHTML(c, &singularPost{ID: postID, Type: "post"})
		},
		SiteName: func() string {
			return optionString(c, "blogname", "GoPress")
		},
		SiteTagline: func() string {
			return optionString(c, "blogdescription", "")
		},
		SiteLogoURL: func() string {
			id := int64(0)
			if c != nil && c.Options != nil {
				switch v := c.Options.GetOption("site_logo", 0).(type) {
				case int64:
					id = v
				case int:
					id = int64(v)
				case float64:
					id = int64(v)
				case string:
					id, _ = strconv.ParseInt(strings.TrimSpace(v), 10, 64)
				}
			}
			if id <= 0 {
				return ""
			}
			return wp.GetAttachmentURL(c, id)
		},
		HomeURL: func() string {
			return strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		},
		IsFrontPage: func() bool {
			// Front block templates set this via env flag when rendering home.
			return false
		},
		// P5.74: core/loginout resolvers.
		IsLoggedIn: func() bool {
			return c != nil && c.CurrentUser != nil
		},
		CurrentURL: func() string {
			if c == nil || c.Req == nil || c.Req.URL == nil {
				return ""
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			path := c.Req.URL.RequestURI()
			if home == "" {
				return path
			}
			scheme := "http"
			if c.Req.TLS != nil || strings.EqualFold(c.Req.Header.Get("X-Forwarded-Proto"), "https") {
				scheme = "https"
			}
			host := c.Req.Host
			if host == "" {
				return home + path
			}
			return scheme + "://" + host + path
		},
		LoginURL: func(redirect string) string {
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			u := home + "/wp-login.php"
			if redirect != "" {
				u += "?redirect_to=" + url.QueryEscape(redirect)
			}
			return u
		},
		LogoutURL: func(redirect string) string {
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			u := home + "/wp-login.php?action=logout"
			if redirect != "" {
				u += "&redirect_to=" + url.QueryEscape(redirect)
			}
			if c != nil {
				if n := c.CreateNonce("log-out"); n != "" {
					u += "&_wpnonce=" + url.QueryEscape(n)
				}
			}
			return u
		},
		FilterPostHTML: func(htmlBody string) string {
			return wp.Texturize(htmlBody)
		},
		ListComments: func(postID int64) []blocks.CommentItem {
			rows, err := wp.ListApprovedComments(c, postID)
			if err != nil {
				return nil
			}
			out := make([]blocks.CommentItem, 0, len(rows))
			for _, cm := range rows {
				out = append(out, blocks.CommentItem{
					ID:          cm.ID,
					Author:      cm.Author,
					AuthorEmail: cm.AuthorEmail,
					AuthorURL:   cm.AuthorURL,
					Content:     wp.KsesPost(cm.Content),
					Date:        cm.Date,
				})
			}
			return out
		},
		AdjacentPost: func(postID int64, previous bool) *blocks.QueryPost {
			// previous = older (date ASC toward past); next = newer.
			op, order := "<", "DESC"
			if !previous {
				op, order = ">", "ASC"
			}
			p, err := wp.GetPostByID(c, postID)
			if err != nil || p == nil {
				return nil
			}
			q, args, err := c.DB.Prepare(
				"SELECT ID, post_title, post_name, post_date FROM "+c.DB.Posts()+
					" WHERE post_type = %s AND post_status = %s AND post_date "+op+" %s "+
					"ORDER BY post_date "+order+" LIMIT 1",
				"post", "publish", p.Date,
			)
			if err != nil {
				return nil
			}
			row, err := c.DB.GetRow(q, args...)
			if err != nil || row == nil {
				return nil
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			id := rowInt64DB(row["ID"])
			name := rowStringDB(row["post_name"])
			date := rowStringDB(row["post_date"])
			title := rowStringDB(row["post_title"])
			link := home + "/?p=" + strconv.FormatInt(id, 10)
			if name != "" && date != "" {
				if t, err := time.Parse("2006-01-02 15:04:05", date); err == nil {
					link = fmt.Sprintf("%s/%04d/%02d/%02d/%s/", home, t.Year(), int(t.Month()), t.Day(), name)
				}
			}
			return &blocks.QueryPost{ID: id, Title: title, Link: link, Date: date}
		},
		AuthorName: func(postID int64) string {
			u := authorUserForPost(c, postID)
			if u == nil {
				return ""
			}
			return u.DisplayName
		},
		AuthorEmail: func(postID int64) string {
			u := authorUserForPost(c, postID)
			if u == nil {
				return ""
			}
			return u.Email
		},
		AuthorBio: func(postID int64) string {
			u := authorUserForPost(c, postID)
			if u == nil || c == nil || c.Meta == nil {
				return ""
			}
			return asMetaString(c.Meta.GetMetadata(wp.MetaUser, u.ID, "description", true))
		},
		AuthorLink: func(postID int64) string {
			u := authorUserForPost(c, postID)
			if u == nil {
				return ""
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			slug := u.Nicename
			if slug == "" {
				slug = u.Login
			}
			if home == "" || slug == "" {
				return ""
			}
			return home + "/author/" + slug + "/"
		},
		PostTerms: func(postID int64, taxonomy string) []blocks.NavLink {
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			q, args, err := c.DB.Prepare(
				"SELECT t.name, t.slug, tt.taxonomy FROM "+c.DB.Terms()+" t "+
					"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
					"INNER JOIN "+c.DB.TermRelationships()+" tr ON tr.term_taxonomy_id = tt.term_taxonomy_id "+
					"WHERE tr.object_id = %d AND tt.taxonomy = %s",
				postID, taxonomy,
			)
			if err != nil {
				return nil
			}
			rows, err := c.DB.GetResults(q, args...)
			if err != nil {
				return nil
			}
			out := make([]blocks.NavLink, 0, len(rows))
			for _, row := range rows {
				name := rowStringDB(row["name"])
				slug := rowStringDB(row["slug"])
				tax := rowStringDB(row["taxonomy"])
				base := "category"
				if tax == "post_tag" {
					base = "tag"
				}
				out = append(out, blocks.NavLink{
					Label: name,
					URL:   home + "/" + base + "/" + slug + "/",
				})
			}
			return out
		},
		// P5.42: widget-style dynamic blocks.
		ListArchives: func() []blocks.ArchiveItem {
			if c == nil || c.DB == nil {
				return nil
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			q, args, err := c.DB.Prepare(
				"SELECT YEAR(post_date) AS y, MONTH(post_date) AS m, COUNT(*) AS n FROM "+c.DB.Posts()+
					" WHERE post_type = %s AND post_status = %s "+
					"GROUP BY YEAR(post_date), MONTH(post_date) ORDER BY post_date DESC LIMIT %d",
				"post", "publish", 24,
			)
			if err != nil {
				return nil
			}
			rows, err := c.DB.GetResults(q, args...)
			if err != nil {
				return nil
			}
			out := make([]blocks.ArchiveItem, 0, len(rows))
			for _, row := range rows {
				y := int(rowInt64DB(row["y"]))
				m := int(rowInt64DB(row["m"]))
				if y <= 0 || m <= 0 || m > 12 {
					continue
				}
				label := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC).Format("January 2006")
				out = append(out, blocks.ArchiveItem{
					Label: label,
					URL:   fmt.Sprintf("%s/%04d/%02d/", home, y, m),
					Count: int(rowInt64DB(row["n"])),
				})
			}
			return out
		},
		ListTagCloud: func() []blocks.TagCloudItem {
			if c == nil || c.DB == nil {
				return nil
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			q, args, err := c.DB.Prepare(
				"SELECT t.name, t.slug, tt.count FROM "+c.DB.Terms()+" t "+
					"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
					"WHERE tt.taxonomy = %s AND tt.count > 0 ORDER BY tt.count DESC, t.name ASC LIMIT %d",
				"post_tag", 45,
			)
			if err != nil {
				return nil
			}
			rows, err := c.DB.GetResults(q, args...)
			if err != nil {
				return nil
			}
			out := make([]blocks.TagCloudItem, 0, len(rows))
			for _, row := range rows {
				slug := rowStringDB(row["slug"])
				out = append(out, blocks.TagCloudItem{
					Name:  rowStringDB(row["name"]),
					URL:   home + "/tag/" + slug + "/",
					Count: int(rowInt64DB(row["count"])),
				})
			}
			return out
		},
		ListLatestComments: func(limit int) []blocks.LatestCommentItem {
			rows, err := wp.ListRecentComments(c, limit)
			if err != nil {
				return nil
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			out := make([]blocks.LatestCommentItem, 0, len(rows))
			for _, cm := range rows {
				title, link := "", home+"/"
				if p, err := wp.GetPostByID(c, cm.PostID); err == nil && p != nil {
					title = p.Title
					link = postPermalink(home, *p)
				}
				out = append(out, blocks.LatestCommentItem{
					ID:        cm.ID,
					Author:    cm.Author,
					AuthorURL: cm.AuthorURL,
					Content:   strings.TrimSpace(stripTagsSimple(cm.Content)),
					Date:      cm.Date,
					PostID:    cm.PostID,
					PostTitle: title,
					PostURL:   link + "#comment-" + strconv.FormatInt(cm.ID, 10),
				})
			}
			return out
		},
	}
}

func authorUserForPost(c *wp.Context, postID int64) *wp.User {
	if c == nil || postID <= 0 {
		return nil
	}
	p, err := wp.GetPostByID(c, postID)
	if err != nil || p == nil || p.Author <= 0 {
		return nil
	}
	u, err := c.GetUserBy("id", p.Author)
	if err != nil || u == nil {
		return nil
	}
	return u
}

func rowStringDB(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case nil:
		return ""
	default:
		s := fmt.Sprint(t)
		if s == "<nil>" {
			return ""
		}
		return s
	}
}

func rowInt64DB(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case []byte:
		n, _ := strconv.ParseInt(string(t), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(fmt.Sprint(t), 10, 64)
		return n
	}
}
