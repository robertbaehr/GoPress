package server

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"sort"
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
	// P3.113: channel title/description via get_bloginfo.
	blogname := wp.BlogInfo(c, "name", "raw")
	if blogname == "" {
		blogname = "GoPress"
	}
	home := strings.TrimRight(wp.BlogInfo(c, "url", "raw"), "/")
	desc := wp.BlogInfo(c, "description", "raw")
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
	b.WriteString(wp.TheGenerator(c, "rss2"))
	b.WriteString("\n")

	for _, cm := range comments {
		post, _ := wp.GetPostByID(c, cm.PostID)
		postTitle := ""
		storedGUID := ""
		link := fmt.Sprintf("%s/?p=%d#comment-%d", home, cm.PostID, cm.ID)
		if post != nil {
			postTitle = post.Title
			storedGUID = post.GUID
			if plink := postPermalink(home, *post); plink != "" {
				link = strings.TrimRight(plink, "/") + "/#comment-" + strconv.FormatInt(cm.ID, 10)
			}
		}
		author := cm.Author
		if author == "" {
			author = "Anonymous"
		}
		title := "Comment on " + postTitle + " by " + author
		// P3.77: get_comment_guid = get_the_guid(post) + #comment-ID.
		guid := commentFeedGUID(c, home, cm.PostID, cm.ID, storedGUID)
		text := commentFeedGetText(c, home, cm)
		contentHTML := commentFeedContent(text)
		descText := commentFeedDescription(text)

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
	b.WriteString(wp.TheGenerator(c, "atom"))
	b.WriteString("\n")

	for _, cm := range comments {
		post, _ := wp.GetPostByID(c, cm.PostID)
		postTitle := ""
		postLink := fmt.Sprintf("%s/?p=%d", home, cm.PostID)
		storedGUID := ""
		link := fmt.Sprintf("%s/?p=%d#comment-%d", home, cm.PostID, cm.ID)
		if post != nil {
			postTitle = post.Title
			storedGUID = post.GUID
			if plink := postPermalink(home, *post); plink != "" {
				postLink = strings.TrimRight(plink, "/") + "/"
				link = strings.TrimRight(plink, "/") + "/#comment-" + strconv.FormatInt(cm.ID, 10)
			}
		}
		// P3.88: thr:in-reply-to ref uses get_the_guid / get_comment_guid.
		postGUID := feedItemGUID(c, home, cm.PostID, storedGUID)
		if postGUID == "" {
			postGUID = postLink
		}
		author := cm.Author
		if author == "" {
			author = "Anonymous"
		}
		// P3.88: the_title_rss on the post title inside "Comment on … by …".
		title := "Comment on " + applyTheTitleRSS(c, postTitle) + " by " + author
		guid := commentFeedGUID(c, home, cm.PostID, cm.ID, storedGUID)
		contentHTML := ""
		if post != nil && wp.PostPasswordRequired(c, r, post) {
			// feed-atom-comments.php: password form when post_password_required.
			contentHTML = commentFeedContent(wp.GetThePasswordForm(c, post, postLink))
		} else {
			contentHTML = commentFeedContent(commentFeedGetText(c, home, cm))
		}

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
			parentGUID := commentFeedGUID(c, home, cm.PostID, cm.Parent, storedGUID)
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

// commentFeedGUID mirrors get_comment_guid (P3.77).
func commentFeedGUID(c *wp.Context, home string, postID, commentID int64, storedGUID string) string {
	base := feedItemGUID(c, home, postID, storedGUID)
	if base == "" {
		base = fmt.Sprintf("%s/?p=%d", home, postID)
	}
	return base + "#comment-" + strconv.FormatInt(commentID, 10)
}

// commentFeedGetText mirrors get_comment_text for comment feeds (P3.77):
// child comments get an "In reply to {author}." HTML prefix.
func commentFeedGetText(c *wp.Context, home string, cm wp.Comment) string {
	text := cm.Content
	if cm.Parent <= 0 {
		return text
	}
	parent, err := wp.GetCommentByID(c, cm.Parent)
	if err != nil || parent == nil {
		return text
	}
	parentLink := fmt.Sprintf("%s/?p=%d#comment-%d", home, parent.PostID, parent.ID)
	if post, _ := wp.GetPostByID(c, parent.PostID); post != nil {
		if plink := postPermalink(home, *post); plink != "" {
			parentLink = strings.TrimRight(plink, "/") + "/#comment-" + strconv.FormatInt(parent.ID, 10)
		}
	}
	name := parent.Author
	if name == "" {
		name = "Anonymous"
	}
	prefix := "In reply to <a href=\"" + html.EscapeString(parentLink) + "\">" + html.EscapeString(name) + "</a>."
	return prefix + "\n\n" + text
}

func commentFeedContent(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// comment_text / wpautop: preserve existing HTML (reply prefix, anchors).
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	parts := strings.Split(raw, "\n\n")
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
	// comment_text_rss: HTML-escaped text in CDATA (entities for tags/quotes).
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

// renderSearch serves /?s= front search results (P3.21 / P3.36 / P3.82).
func (f *Front) renderSearch(w http.ResponseWriter, c *wp.Context, term string) {
	if c != nil {
		if c.Scope == nil {
			c.Scope = &wp.RequestScope{}
		}
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
		tpl = applySearchTemplateContentFilter(c, tpl)
		f.renderSearchBlockTemplate(w, c, EnsureSearchTemplateParts(tpl), term)
		return
	}
	if wrapped, ok := wrapSearchShimWithParts(c, term); ok {
		f.renderSearchBlockTemplate(w, c, wrapped, term)
		return
	}
	f.renderSearchShim(w, c, term)
}

// applySearchTemplateContentFilter lets hooks adjust resolved search block markup (P3.82).
func applySearchTemplateContentFilter(c *wp.Context, tpl string) string {
	if c == nil || c.Hooks == nil {
		return tpl
	}
	if v := c.Hooks.ApplyFilters("search_template_content", tpl); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return tpl
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
// P3.82: richer post-template (date + excerpt) and query-no-results fallback.
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
			`<div class="wp-block-query"><!-- wp:post-template -->` +
			`<!-- wp:post-title {"isLink":true} /-->` +
			`<!-- wp:post-date /-->` +
			`<!-- wp:post-excerpt /-->` +
			`<!-- /wp:post-template -->` +
			`<!-- wp:query-no-results --><!-- wp:paragraph --><p>No results found.</p><!-- /wp:paragraph --><!-- /wp:query-no-results -->` +
			`</div><!-- /wp:query --></main><!-- /wp:group -->`,
	), true
}

func (f *Front) renderSearchBlockTemplate(w http.ResponseWriter, c *wp.Context, template, term string) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.SearchQuery = term
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPostFiltered(c, body))

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

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html")
	b.WriteString(wp.LanguageAttributesHTML(c))
	b.WriteString(">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(frontSearchDocumentTitle(c, blogname, term))
	b.WriteString("</title>\n")
	extraTitle, extraHref := "", ""
	if term != "" {
		// feed_links_extra search: "%1$s %2$s Search Results for &#8220;%3$s&#8221; Feed"
		extraTitle = html.EscapeString(blogname) + " &raquo; Search Results for &#8220;" + html.EscapeString(term) + "&#8221; Feed"
		extraHref = home + "/search/" + pathEscapeSearch(term) + "/feed/rss2/"
	}
	writeFrontHeadLinks(&b, c, home, blogname, "", "", "", extraTitle, extraHref, "", wp.SiteIconMetaTags(c))
	writeArchiveRelLinks(&b, c, home)
	b.WriteString(wp.EmojiHeadHTMLCtx(c))
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	writeFrontDocumentClose(&b, c)

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// frontSearchDocumentTitle builds the search <title> and honors document_title_parts (P3.82).
func frontSearchDocumentTitle(c *wp.Context, blogname, term string) string {
	titlePart := "Search"
	if strings.TrimSpace(term) != "" {
		titlePart = "Search Results for \u201C" + term + "\u201D"
	}
	parts := map[string]string{
		"title":   titlePart,
		"page":    "",
		"tagline": "",
		"site":    blogname,
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("document_title_parts", parts); v != nil {
			if m, ok := v.(map[string]string); ok {
				parts = m
			}
		}
	}
	title := strings.TrimSpace(parts["title"])
	if title == "" {
		title = "Search"
	}
	site := strings.TrimSpace(parts["site"])
	out := html.EscapeString(title)
	if site != "" {
		out += " &#8211; " + html.EscapeString(site)
	}
	return out
}

// renderDateArchive serves year/month(/day) front archives (P3.22 / P3.86).
func (f *Front) renderDateArchive(w http.ResponseWriter, c *wp.Context, vars map[string]string) {
	year := atoiFront(vars["year"])
	month := atoiFront(vars["monthnum"])
	day := atoiFront(vars["day"])
	if year <= 0 {
		f.render404(w, c)
		return
	}
	if c != nil && c.Scope == nil {
		c.Scope = &wp.RequestScope{}
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
		tpl = applyDateTemplateContentFilter(c, tpl)
		f.renderDateArchiveBlockTemplate(w, c, tpl, title, year, month, day)
		return
	}
	f.renderDateArchiveShim(w, c, title, year, month, day)
}

// applyDateTemplateContentFilter lets hooks adjust resolved date block markup (P3.86).
func applyDateTemplateContentFilter(c *wp.Context, tpl string) string {
	if c == nil || c.Hooks == nil {
		return tpl
	}
	if v := c.Hooks.ApplyFilters("date_template_content", tpl); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return tpl
}

// frontDateDocumentTitle builds the date archive <title> and honors document_title_parts (P3.86).
func frontDateDocumentTitle(c *wp.Context, blogname, archiveTitle string) string {
	parts := map[string]string{
		"title":   archiveTitle,
		"page":    "",
		"tagline": "",
		"site":    blogname,
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("document_title_parts", parts); v != nil {
			if m, ok := v.(map[string]string); ok {
				parts = m
			}
		}
	}
	title := strings.TrimSpace(parts["title"])
	if title == "" {
		title = "Archives"
	}
	site := strings.TrimSpace(parts["site"])
	out := html.EscapeString(title)
	if site != "" {
		out += " &#8211; " + html.EscapeString(site)
	}
	return out
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

func (f *Front) renderDateArchiveBlockTemplate(w http.ResponseWriter, c *wp.Context, template, archiveTitle string, year, month, day int) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.ArchiveTitle = archiveTitle
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPostFiltered(c, body))
	safe = ensureMainSkipTarget(safe)

	bodyClasses := dateArchiveBodyClass(year, month, day) + " wp-embed-responsive"
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html")
	b.WriteString(wp.LanguageAttributesHTML(c))
	b.WriteString(">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(frontDateDocumentTitle(c, blogname, archiveTitle))
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, c, home, blogname, "", "", "", "", "", "", wp.SiteIconMetaTags(c))
	writeArchiveRelLinks(&b, c, home)
	b.WriteString(wp.EmojiHeadHTMLCtx(c))
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	writeFrontDocumentClose(&b, c)

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func dateArchiveBodyClass(year, month, day int) string {
	classes := "archive date"
	if day > 0 && month > 0 {
		classes += " date-day"
	} else if month > 0 {
		classes += " date-month"
	} else {
		classes += " date-year"
	}
	if year > 0 {
		classes += " date-" + strconv.Itoa(year)
	}
	if month > 0 {
		classes += fmt.Sprintf(" date-%04d%02d", year, month)
	}
	if day > 0 {
		classes += fmt.Sprintf(" date-%04d%02d%02d", year, month, day)
	}
	return classes
}

// renderAuthorArchive serves /author/{nicename}/ front archives (P3.23 / P3.83).
func (f *Front) renderAuthorArchive(w http.ResponseWriter, c *wp.Context, nicename string) {
	nicename = strings.TrimSpace(nicename)
	if nicename == "" {
		f.render404(w, c)
		return
	}
	if c != nil && c.Scope == nil {
		c.Scope = &wp.RequestScope{}
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
		tpl = applyAuthorTemplateContentFilter(c, tpl)
		f.renderAuthorArchiveBlockTemplate(w, c, tpl, title, user)
		return
	}
	f.renderAuthorArchiveShim(w, c, title, user)
}

// applyAuthorTemplateContentFilter lets hooks adjust resolved author block markup (P3.83).
func applyAuthorTemplateContentFilter(c *wp.Context, tpl string) string {
	if c == nil || c.Hooks == nil {
		return tpl
	}
	if v := c.Hooks.ApplyFilters("author_template_content", tpl); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return tpl
}

// frontAuthorDocumentTitle builds the author archive <title> and honors document_title_parts (P3.83).
func frontAuthorDocumentTitle(c *wp.Context, blogname, archiveTitle string) string {
	parts := map[string]string{
		"title":   archiveTitle,
		"page":    "",
		"tagline": "",
		"site":    blogname,
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("document_title_parts", parts); v != nil {
			if m, ok := v.(map[string]string); ok {
				parts = m
			}
		}
	}
	title := strings.TrimSpace(parts["title"])
	if title == "" {
		title = "Author"
	}
	site := strings.TrimSpace(parts["site"])
	out := html.EscapeString(title)
	if site != "" {
		out += " &#8211; " + html.EscapeString(site)
	}
	return out
}

func (f *Front) renderAuthorArchiveBlockTemplate(w http.ResponseWriter, c *wp.Context, template, archiveTitle string, user *wp.User) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.ArchiveTitle = archiveTitle
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPostFiltered(c, body))
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

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html")
	b.WriteString(wp.LanguageAttributesHTML(c))
	b.WriteString(">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(frontAuthorDocumentTitle(c, blogname, archiveTitle))
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
	writeFrontHeadLinks(&b, c, home, blogname, "", restPath, "", extraTitle, extraHref, "", wp.SiteIconMetaTags(c))
	writeArchiveRelLinks(&b, c, home)
	b.WriteString(wp.EmojiHeadHTMLCtx(c))
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	writeFrontDocumentClose(&b, c)

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderCategoryArchive serves /category/{slug}/ front archives (P3.25 / P3.84).
func (f *Front) renderCategoryArchive(w http.ResponseWriter, c *wp.Context, slug string) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		f.render404(w, c)
		return
	}
	if c != nil && c.Scope == nil {
		c.Scope = &wp.RequestScope{}
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
		tpl = applyCategoryTemplateContentFilter(c, tpl)
		f.renderCategoryArchiveBlockTemplate(w, c, tpl, title, term)
		return
	}
	f.renderCategoryArchiveShim(w, c, title, term)
}

// applyCategoryTemplateContentFilter lets hooks adjust resolved category block markup (P3.84).
func applyCategoryTemplateContentFilter(c *wp.Context, tpl string) string {
	if c == nil || c.Hooks == nil {
		return tpl
	}
	if v := c.Hooks.ApplyFilters("category_template_content", tpl); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return tpl
}

// frontCategoryDocumentTitle builds the category archive <title> and honors document_title_parts (P3.84).
func frontCategoryDocumentTitle(c *wp.Context, blogname, archiveTitle string) string {
	parts := map[string]string{
		"title":   archiveTitle,
		"page":    "",
		"tagline": "",
		"site":    blogname,
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("document_title_parts", parts); v != nil {
			if m, ok := v.(map[string]string); ok {
				parts = m
			}
		}
	}
	title := strings.TrimSpace(parts["title"])
	if title == "" {
		title = "Category"
	}
	site := strings.TrimSpace(parts["site"])
	out := html.EscapeString(title)
	if site != "" {
		out += " &#8211; " + html.EscapeString(site)
	}
	return out
}

func (f *Front) renderCategoryArchiveBlockTemplate(w http.ResponseWriter, c *wp.Context, template, archiveTitle string, term *termInfo) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.ArchiveTitle = archiveTitle
	// P5.76: expose term fields for core/term-description (+ term-data bindings).
	env.TermContext = termBindingContext("category", term)
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPostFiltered(c, body))
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

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html")
	b.WriteString(wp.LanguageAttributesHTML(c))
	b.WriteString(">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(frontCategoryDocumentTitle(c, blogname, archiveTitle))
	b.WriteString("</title>\n")
	// P3.103: feed_links_extra category alternate.
	extraTitle, extraHref := termArchiveFeedLink(c, blogname, home, "category", term)
	writeFrontHeadLinks(&b, c, home, blogname, "", "", "", extraTitle, extraHref, "", wp.SiteIconMetaTags(c))
	writeArchiveRelLinks(&b, c, home)
	b.WriteString(wp.EmojiHeadHTMLCtx(c))
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	writeFrontDocumentClose(&b, c)

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderCategoryArchiveShim(w http.ResponseWriter, c *wp.Context, title string, term *termInfo) {
	blogname := optionString(c, "blogname", "GoPress")
	catSlug := ""
	if term != nil {
		catSlug = term.Slug
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 10,
		Category: catSlug,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	bodyClass := "archive category"
	if term != nil {
		if term.Slug != "" {
			bodyClass += " category-" + term.Slug
		}
		if term.ID > 0 {
			bodyClass += " category-" + strconv.FormatInt(term.ID, 10)
		}
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(frontCategoryDocumentTitle(c, blogname, title))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, bodyClass)))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<main id=\"wp--skip-link--target\">\n")
	b.WriteString("<h1 class=\"page-title\">")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>\n<ul class=\"posts\">\n")
	if len(posts) == 0 {
		b.WriteString("<li class=\"no-results\">No posts found in this category.</li>\n")
	}
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a>")
		// P3.124: classic shim uses get_the_excerpt (password + trim + filters).
		if ex := strings.TrimSpace(wp.GetTheExcerpt(c, &p)); ex != "" {
			b.WriteString("<p class=\"entry-summary\">")
			b.WriteString(html.EscapeString(ex))
			b.WriteString("</p>")
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n</main>\n</body>\n</html>\n")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderTagArchive serves /tag/{slug}/ front archives (P3.24 / P3.85).
func (f *Front) renderTagArchive(w http.ResponseWriter, c *wp.Context, slug string) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		f.render404(w, c)
		return
	}
	if c != nil && c.Scope == nil {
		c.Scope = &wp.RequestScope{}
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
		tpl = applyTagTemplateContentFilter(c, tpl)
		f.renderTagArchiveBlockTemplate(w, c, tpl, title, term)
		return
	}
	f.renderTagArchiveShim(w, c, title, term)
}

// applyTagTemplateContentFilter lets hooks adjust resolved tag block markup (P3.85).
func applyTagTemplateContentFilter(c *wp.Context, tpl string) string {
	if c == nil || c.Hooks == nil {
		return tpl
	}
	if v := c.Hooks.ApplyFilters("tag_template_content", tpl); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return tpl
}

// frontTagDocumentTitle builds the tag archive <title> and honors document_title_parts (P3.85).
func frontTagDocumentTitle(c *wp.Context, blogname, archiveTitle string) string {
	parts := map[string]string{
		"title":   archiveTitle,
		"page":    "",
		"tagline": "",
		"site":    blogname,
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("document_title_parts", parts); v != nil {
			if m, ok := v.(map[string]string); ok {
				parts = m
			}
		}
	}
	title := strings.TrimSpace(parts["title"])
	if title == "" {
		title = "Tag"
	}
	site := strings.TrimSpace(parts["site"])
	out := html.EscapeString(title)
	if site != "" {
		out += " &#8211; " + html.EscapeString(site)
	}
	return out
}

func (f *Front) renderTagArchiveBlockTemplate(w http.ResponseWriter, c *wp.Context, template, archiveTitle string, term *termInfo) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.ArchiveTitle = archiveTitle
	// P5.76: expose term fields for core/term-description (+ term-data bindings).
	env.TermContext = termBindingContext("post_tag", term)
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPostFiltered(c, body))
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

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html")
	b.WriteString(wp.LanguageAttributesHTML(c))
	b.WriteString(">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(frontTagDocumentTitle(c, blogname, archiveTitle))
	b.WriteString("</title>\n")
	// P3.103: feed_links_extra tag alternate.
	extraTitle, extraHref := termArchiveFeedLink(c, blogname, home, "post_tag", term)
	writeFrontHeadLinks(&b, c, home, blogname, "", "", "", extraTitle, extraHref, "", wp.SiteIconMetaTags(c))
	writeArchiveRelLinks(&b, c, home)
	b.WriteString(wp.EmojiHeadHTMLCtx(c))
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	writeFrontDocumentClose(&b, c)

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderTagArchiveShim(w http.ResponseWriter, c *wp.Context, title string, term *termInfo) {
	blogname := optionString(c, "blogname", "GoPress")
	tagSlug := ""
	if term != nil {
		tagSlug = term.Slug
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 10,
		Tag: tagSlug,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	bodyClass := "archive tag"
	if term != nil {
		if term.Slug != "" {
			bodyClass += " tag-" + term.Slug
		}
		if term.ID > 0 {
			bodyClass += " tag-" + strconv.FormatInt(term.ID, 10)
		}
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(frontTagDocumentTitle(c, blogname, title))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, bodyClass)))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<main id=\"wp--skip-link--target\">\n")
	b.WriteString("<h1 class=\"page-title\">")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>\n<ul class=\"posts\">\n")
	if len(posts) == 0 {
		b.WriteString("<li class=\"no-results\">No posts found in this tag.</li>\n")
	}
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a>")
		// P3.124: classic shim uses get_the_excerpt (password + trim + filters).
		if ex := strings.TrimSpace(wp.GetTheExcerpt(c, &p)); ex != "" {
			b.WriteString("<p class=\"entry-summary\">")
			b.WriteString(html.EscapeString(ex))
			b.WriteString("</p>")
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n</main>\n</body>\n</html>\n")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

type termInfo struct {
	ID          int64
	Name        string
	Slug        string
	Description string // term_taxonomy.description for core/term-description (P5.76).
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
		"SELECT t.term_id, t.name, t.slug, tt.description FROM "+c.DB.Terms()+" t "+
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
		ID:          id,
		Name:        fmt.Sprint(row["name"]),
		Slug:        fmt.Sprint(row["slug"]),
		Description: asMetaString(row["description"]),
	}, nil
}

// termBindingContext builds blocks.TermBindingContext for archive templates (P5.76).
func termBindingContext(taxonomy string, term *termInfo) *blocks.TermBindingContext {
	if term == nil || term.ID <= 0 {
		return nil
	}
	return &blocks.TermBindingContext{
		ID:          term.ID,
		Taxonomy:    taxonomy,
		Name:        term.Name,
		Slug:        term.Slug,
		Description: term.Description,
	}
}

func (f *Front) renderAuthorArchiveShim(w http.ResponseWriter, c *wp.Context, title string, user *wp.User) {
	blogname := optionString(c, "blogname", "GoPress")
	authorID := int64(0)
	if user != nil {
		authorID = user.ID
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 10,
		Author: authorID,
	})
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	bodyClass := "archive author"
	if user != nil {
		nicename := user.Nicename
		if nicename == "" {
			nicename = user.Login
		}
		if nicename != "" {
			bodyClass += " author-" + nicename
		}
		if user.ID > 0 {
			bodyClass += " author-" + strconv.FormatInt(user.ID, 10)
		}
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(frontAuthorDocumentTitle(c, blogname, title))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, bodyClass)))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<main id=\"wp--skip-link--target\">\n")
	b.WriteString("<h1 class=\"page-title\">")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>\n<ul class=\"posts\">\n")
	if len(posts) == 0 {
		b.WriteString("<li class=\"no-results\">No posts by this author.</li>\n")
	}
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a>")
		// P3.124: classic shim uses get_the_excerpt (password + trim + filters).
		if ex := strings.TrimSpace(wp.GetTheExcerpt(c, &p)); ex != "" {
			b.WriteString("<p class=\"entry-summary\">")
			b.WriteString(html.EscapeString(ex))
			b.WriteString("</p>")
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n</main>\n</body>\n</html>\n")
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
	bodyClass := dateArchiveBodyClass(year, month, day)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(frontDateDocumentTitle(c, blogname, title))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, bodyClass)))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<main id=\"wp--skip-link--target\">\n")
	b.WriteString("<h1 class=\"page-title\">")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>\n<ul class=\"posts\">\n")
	if len(posts) == 0 {
		b.WriteString("<li class=\"no-results\">No posts found for this date.</li>\n")
	}
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a>")
		// P3.124: classic shim uses get_the_excerpt (password + trim + filters).
		if ex := strings.TrimSpace(wp.GetTheExcerpt(c, &p)); ex != "" {
			b.WriteString("<p class=\"entry-summary\">")
			b.WriteString(html.EscapeString(ex))
			b.WriteString("</p>")
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n</main>\n</body>\n</html>\n")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderSearchShim(w http.ResponseWriter, c *wp.Context, term string) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	if home == "" {
		home = "/"
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType: "post", PostStatus: "publish", PostsPerPage: 10, Search: term,
	})
	if err != nil {
		http.Error(w, "search: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	bodyClass := "search search-results"
	if term == "" || len(posts) == 0 {
		bodyClass = "search search-no-results"
		if term != "" {
			bodyClass = "search search-results search-no-results"
		}
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(frontSearchDocumentTitle(c, blogname, term))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, bodyClass)))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<main id=\"wp--skip-link--target\">\n")
	b.WriteString("<h1 class=\"page-title\">Search results for: ")
	b.WriteString(html.EscapeString(term))
	b.WriteString("</h1>\n")
	// P3.82: classic shim search form for refining the query.
	action := home
	if !strings.HasSuffix(action, "/") {
		action += "/"
	}
	b.WriteString(`<form role="search" method="get" action="`)
	b.WriteString(html.EscapeString(action))
	b.WriteString(`" class="search-form wp-block-search">`)
	b.WriteString(`<label><span class="screen-reader-text">Search</span>`)
	b.WriteString(`<input type="search" name="s" value="`)
	b.WriteString(html.EscapeString(term))
	b.WriteString(`" placeholder="Search"/></label>`)
	b.WriteString(`<input type="submit" value="Search"/>`)
	b.WriteString("</form>\n")
	b.WriteString("<ul class=\"posts\">\n")
	if len(posts) == 0 {
		b.WriteString("<li class=\"no-results\">No results found.</li>\n")
	}
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a>")
		if ex := strings.TrimSpace(wp.GetTheExcerpt(c, &p)); ex != "" {
			b.WriteString("<p class=\"excerpt\">")
			b.WriteString(html.EscapeString(ex))
			b.WriteString("</p>")
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n</main>\n</body>\n</html>\n")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderBlockTemplate parses and renders a block theme template document.
func (f *Front) renderBlockTemplate(w http.ResponseWriter, c *wp.Context, template string) {
	// P3.113: name / url / language via get_bloginfo.
	blogname := wp.BlogInfo(c, "name", "raw")
	if blogname == "" {
		blogname = "GoPress"
	}
	home := strings.TrimRight(wp.BlogInfo(c, "url", "raw"), "/")
	charset := wp.BlogInfo(c, "charset", "raw")
	if charset == "" {
		charset = "UTF-8"
	}
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	env.IsFrontPage = func() bool {
		if c != nil && c.Scope != nil {
			return c.Scope.IsFrontPage
		}
		return true
	}
	env.IsHome = func() bool {
		return c != nil && c.Scope != nil && c.Scope.IsHome
	}
	env.IsPaged = func() bool {
		return c != nil && c.Scope != nil && c.Scope.Paged > 1
	}
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPostFiltered(c, body))

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
	b.WriteString("<!DOCTYPE html>\n<html")
	b.WriteString(wp.LanguageAttributesHTML(c))
	b.WriteString(">\n<head>\n")
	b.WriteString("<meta charset=\"")
	b.WriteString(html.EscapeString(charset))
	b.WriteString("\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	// P3.126: block theme <title> via wp_get_document_title.
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(wp.GetDocumentTitle(c)))
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, c, home, blogname, "", "", "", "", "", "", wp.SiteIconMetaTags(c))
	b.WriteString(wp.EmojiHeadHTMLCtx(c)) // P3.12: wpemoji print scripts skipped
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	writeFrontDocumentClose(&b, c)

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset="+charset)
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
	b.WriteString(wp.EmojiHeadHTMLCtx(c)) // P3.12: wpemoji print scripts skipped
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

	// P3.109: when attachment pages are disabled, redirect_canonical sends
	// attachment requests to the media file URL (wp_attachment_pages_enabled=0).
	if post.Type == "attachment" && f.tryAttachmentFileRedirect(w, c, post) {
		return
	}

	// P5.17/P5.24: prefer block theme singular hierarchy when available.
	// Attachments keep the shim until attachment-* templates land.
	if post.Type == "page" || post.Type == "post" {
		// P3.80: static front page uses front-page → home → index hierarchy.
		var slugs []string
		if c != nil && c.Scope != nil && c.Scope.IsFrontPage {
			slugs = wp.HomeTemplateSlugs(true)
		} else {
			slugs = wp.SingularTemplateSlugs(post.Type, post.Name, post.ID)
		}
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
	perm := singularPermalink(home, post)
	env.SetCurrentPost(&blocks.QueryPost{
		ID:               post.ID,
		Title:            post.Title,
		Content:          post.Content,
		Date:             post.Date,
		Link:             perm,
		FeaturedImageURL: post.FeaturedImageURL,
		FeaturedImageAlt: post.FeaturedImageAlt,
		Password:         post.Password,
		ClassList:        wp.GetPostClass(c, post.ID),
	})
	// P3.108: wire page_comments / cpage into comment-template + comments-pagination.
	applyCommentPagingEnv(env, c, post.ID, perm)
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
	safe := wp.ProcessInteractivityHTML(wp.KsesPostFiltered(c, body))
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
	b.WriteString("<!DOCTYPE html>\n<html")
	b.WriteString(wp.LanguageAttributesHTML(c))
	b.WriteString(">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString(" &#8211; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n")
	page, cpage := wp.CanonicalPageFromRequest(c)
	if canon := wp.GetCanonicalURL(c, post.ID, page, cpage); canon != "" {
		perm = canon
	}
	restPath := "/wp/v2/posts/" + strconv.FormatInt(post.ID, 10)
	if post.Type == "page" {
		restPath = "/wp/v2/pages/" + strconv.FormatInt(post.ID, 10)
	}
	shortlink := wp.GetShortlink(c, post.ID, "post", 0)
	// P3.128: singular feed_links_extra post comments alternate (gated; not the
	// legacy title+permalink path). Pass empty title so writeFrontHeadLinks
	// does not double-emit.
	extraTitle, extraHref := singularCommentsFeedLink(c, home, blogname, perm, post)
	writeFrontHeadLinks(&b, c, home, blogname, perm, restPath, "", extraTitle, extraHref, shortlink, wp.SiteIconMetaTags(c))
	b.WriteString(wp.EmojiHeadHTMLCtx(c))
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	// trackback_rdf() / xmlns:trackback is intentionally omitted (P3.46): trackbacks are gone.
	writeFrontDocumentClose(&b, c)

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
		if post.Type == "attachment" {
			// P3.109: prepend_attachment subset — medium image linked to the file.
			content = wp.PrependAttachment(c, post.ID, content)
		}
		// the_content: blocks → texturize → classic wpautop → shortcodes → capital_P (P5.11 / P3.123).
		content = wp.GetTheContent(content, "", false)
		content = wp.TheContentDisplay(c, content)
		safeContent = wp.ProcessInteractivityHTML(wp.KsesPostFiltered(c, content))
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

	// P3.125: classic shim <title> via wp_title + blogname (theme filter style).
	if c != nil && c.Scope != nil {
		c.Scope.QueriedObjectID = post.ID
		if post.Type == "page" {
			c.Scope.IsPage = true
		} else {
			c.Scope.IsSingle = true
		}
		c.Scope.IsSingular = true
	}
	docTitle := strings.TrimSpace(wp.GetWPTitle(c, "\u2014", "right"))
	if docTitle == "" {
		docTitle = post.Title + " \u2014 "
	}
	docTitle = docTitle + blogname

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(docTitle))
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
	if c.Scope == nil {
		c.Scope = wp.NewRequestScope()
	}
	c.Scope.IsEmbed = true
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
	excerpt = wp.ProcessInteractivityHTML(wp.KsesPostFiltered(c, excerpt))
	// Keep embed payloads small.
	if len(excerpt) > 400 {
		excerpt = excerpt[:400] + "&hellip;"
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html")
	b.WriteString(wp.LanguageAttributesHTML(c))
	b.WriteString(" class=\"no-js\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	// P3.90: embed_head includes wp_robots (noindex on embeds).
	b.WriteString(frontRobotsMetaTag(c))
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
	b.WriteString("<!DOCTYPE html>\n<html")
	b.WriteString(wp.LanguageAttributesHTML(c))
	b.WriteString(" class=\"no-js\">\n<head>\n")
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

// applyCommentPagingEnv sets Env comment paging fields from options + request (P3.108).
func applyCommentPagingEnv(env *blocks.Env, c *wp.Context, postID int64, permalink string) {
	if env == nil {
		return
	}
	n := 0
	if rows, err := wp.ListApprovedComments(c, postID); err == nil {
		// Count top-level comments (matches comment-template paging).
		for _, cm := range rows {
			if cm.Parent == 0 {
				n++
			}
		}
		if n == 0 {
			n = len(rows)
		}
	}
	paging := wp.CommentPagingForPost(c, n)
	env.PageComments = paging.Enabled
	env.CommentsPerPage = paging.PerPage
	env.CommentPage = paging.Page
	env.CommentTotalPages = paging.Total
	env.CommentPermalink = permalink
}

func (f *Front) writeCommentsSection(b *strings.Builder, c *wp.Context, post *singularPost) {
	comments, err := wp.ListApprovedComments(c, post.ID)
	if err != nil {
		comments = nil
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	perm := singularPermalink(home, post)
	// P3.108: page_comments + cpage slice + comment navigation.
	paging := wp.CommentPagingForPost(c, len(comments))
	pageComments := wp.SliceCommentPage(comments, paging)
	b.WriteString("<section id=\"comments\" class=\"comments-area\">\n")
	b.WriteString("<h2 class=\"comments-title\">Comments</h2>\n")
	if len(comments) == 0 {
		b.WriteString("<p class=\"no-comments\">No comments yet.</p>\n")
	} else {
		b.WriteString(wp.CommentPaginationHTML(perm, paging))
		b.WriteString("<ol class=\"comment-list\">\n")
		for _, cm := range pageComments {
			b.WriteString("<li id=\"comment-")
			b.WriteString(strconv.FormatInt(cm.ID, 10))
			b.WriteString("\" class=\"comment\">\n")
			b.WriteString("<article class=\"comment-body\">\n")
			b.WriteString("<footer class=\"comment-meta\"><b class=\"fn\">")
			b.WriteString(html.EscapeString(cm.Author))
			b.WriteString("</b></footer>\n")
			b.WriteString("<div class=\"comment-content\">\n")
			b.WriteString(wp.KsesData(cm.Content))
			b.WriteString("\n</div>\n")
			// P3.97: reply link (?replytocom=ID#respond).
			replyHref := perm
			if strings.Contains(replyHref, "?") {
				replyHref += "&replytocom=" + strconv.FormatInt(cm.ID, 10) + "#respond"
			} else {
				replyHref += "?replytocom=" + strconv.FormatInt(cm.ID, 10) + "#respond"
			}
			b.WriteString(`<div class="reply"><a rel="nofollow" class="comment-reply-link" href="`)
			b.WriteString(html.EscapeString(replyHref))
			b.WriteString(`" data-commentid="`)
			b.WriteString(strconv.FormatInt(cm.ID, 10))
			b.WriteString(`" data-postid="`)
			b.WriteString(strconv.FormatInt(post.ID, 10))
			b.WriteString(`" aria-label="Reply to `)
			b.WriteString(html.EscapeString(cm.Author))
			b.WriteString(`">Reply</a></div>` + "\n")
			b.WriteString("</article>\n</li>\n")
		}
		b.WriteString("</ol>\n")
		b.WriteString(wp.CommentPaginationHTML(perm, paging))
	}
	b.WriteString(commentFormHTML(c, post))
	b.WriteString("</section>\n")
}

// commentReplyToID resolves ?replytocom= for the current request (P3.97 / _get_comment_reply_id).
func commentReplyToID(c *wp.Context, post *singularPost) (int64, string) {
	if c == nil || c.Req == nil || post == nil {
		return 0, ""
	}
	raw := strings.TrimSpace(c.Req.URL.Query().Get("replytocom"))
	if raw == "" {
		return 0, ""
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, ""
	}
	cm, err := wp.GetCommentByID(c, id)
	if err != nil || cm == nil {
		return 0, ""
	}
	if cm.PostID != post.ID || cm.Approved != "1" {
		return 0, ""
	}
	return cm.ID, cm.Author
}

// commentFormHTML mirrors WP comment_form() markup used under posts (P5.18 / P3.97).
func commentFormHTML(c *wp.Context, post *singularPost) string {
	if post == nil {
		return ""
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	action := "/wp-comments-post.php"
	if home != "" {
		action = home + "/wp-comments-post.php"
	}
	replyID, replyAuthor := commentReplyToID(c, post)
	cancelHref := "#respond"
	if home != "" {
		// WP prints a root-relative cancel URL on front templates (strip replytocom).
		perm := singularPermalink(home, post)
		cancelHref = strings.TrimPrefix(perm, home) + "#respond"
		if !strings.HasPrefix(cancelHref, "/") {
			cancelHref = "/" + cancelHref
		}
	}
	title := "Leave a Reply"
	cancelStyle := ` style="display:none;"`
	if replyID > 0 {
		title = "Leave a Reply to " + html.EscapeString(replyAuthor)
		cancelStyle = ""
	}
	var b strings.Builder
	b.WriteString(`<div id="respond" class="comment-respond wp-block-post-comments-form">` + "\n")
	b.WriteString(`<h3 id="reply-title" class="comment-reply-title">`)
	b.WriteString(title)
	b.WriteString(` <small><a rel="nofollow" id="cancel-comment-reply-link" href="`)
	b.WriteString(html.EscapeString(cancelHref))
	b.WriteString(`"`)
	b.WriteString(cancelStyle)
	b.WriteString(`>Cancel reply</a></small></h3>`)
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
	b.WriteString(`<input type="hidden" name="comment_parent" id="comment_parent" value="`)
	b.WriteString(strconv.FormatInt(replyID, 10))
	b.WriteString(`" />` + "\n")
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

	// P3.69 / P3.70: precompute Media RSS items; emit xmlns:media only when present
	// so bare fixture feeds match WP (no Media RSS namespace by default).
	mediaByPost := make([][]feedMediaContent, len(posts))
	thumbByPost := make([]string, len(posts))
	hasMedia := false
	for i, p := range posts {
		mediaByPost[i] = feedMediaContentForPost(c, p)
		thumbByPost[i] = feedThumbnailURLForPost(c, p)
		if len(mediaByPost[i]) > 0 || thumbByPost[i] != "" {
			hasMedia = true
		}
	}

	var b strings.Builder
	// Match WP feed-rss2.php channel shape (P3.13).
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"` + "\n")
	b.WriteString("\txmlns:content=\"http://purl.org/rss/1.0/modules/content/\"\n")
	b.WriteString("\txmlns:wfw=\"http://wellformedweb.org/CommentAPI/\"\n")
	b.WriteString("\txmlns:dc=\"http://purl.org/dc/elements/1.1/\"\n")
	b.WriteString("\txmlns:atom=\"http://www.w3.org/2005/Atom\"\n")
	b.WriteString("\txmlns:sy=\"http://purl.org/rss/1.0/modules/syndication/\"\n")
	b.WriteString("\txmlns:slash=\"http://purl.org/rss/1.0/modules/slash/\"\n")
	if hasMedia {
		b.WriteString("\txmlns:media=\"http://search.yahoo.com/mrss/\"\n")
	}
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
	b.WriteString(wp.TheGenerator(c, "rss2"))
	b.WriteString("\n")
	// P3.52: rss2_site_icon channel <image> when a custom site icon is set.
	if img := rss2SiteIconImage(c, blogname, home); img != "" {
		b.WriteString(img)
	}
	for i, p := range posts {
		link := postPermalink(home, p.Post)
		guid := feedItemGUID(c, home, p.ID, p.GUID)
		// WP feed-rss2.php indents the first item with one tab and later items with two.
		if i == 0 {
			b.WriteString("\t<item>\n")
		} else {
			b.WriteString("\t\t<item>\n")
		}
		b.WriteString("\t\t<title>")
		// P3.75: the_title_rss filter on item title.
		b.WriteString(xmlEscape(applyTheTitleRSS(c, p.Title)))
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
			// P3.71: domain attribute is the taxonomy (WXR-style).
			b.WriteString("\t\t\t\t<category domain=\"")
			b.WriteString(xmlEscape(cat.Domain))
			b.WriteString("\"><![CDATA[")
			b.WriteString(cat.Name)
			b.WriteString("]]></category>\n")
		}
		// P3.62: isPermaLink=false even when GUID is a pretty permalink URL.
		b.WriteString(rssGUIDTag(guid, link))
		b.WriteString("\t\t\t\t\t<description><![CDATA[")
		// P3.74: the_excerpt_rss filter on item description.
		b.WriteString(applyTheExcerptRSS(c, p.ExcerptRSS))
		b.WriteString("]]></description>\n")
		b.WriteString("\t\t\t\t\t\t\t\t\t\t")
		// P3.63: content:encoded when available (skipped when rss_use_excerpt).
		if tag := rss2ContentEncodedTag(c, p); tag != "" {
			b.WriteString(tag)
		}
		b.WriteString("\t\t\t\t\t\n")
		if p.CommentCount > 0 || p.CommentStatus == "open" {
			// P3.72: wfw:commentRss via get_post_comments_feed_link (rss2).
			b.WriteString("\t\t\t\t\t<wfw:commentRss>")
			b.WriteString(xmlEscape(wp.GetPostCommentsFeedLink(c, link, "rss2")))
			b.WriteString("</wfw:commentRss>\n")
			b.WriteString("\t\t\t<slash:comments>")
			b.WriteString(strconv.FormatInt(p.CommentCount, 10))
			b.WriteString("</slash:comments>\n")
		}
		for _, enc := range p.Enclosures {
			b.WriteString(rssEnclosureTag(c, enc))
		}
		// P3.69: Media RSS <media:content> from featured image + enclosures.
		for _, mc := range mediaByPost[i] {
			b.WriteString(rssMediaContentTag(mc))
		}
		// P3.70: Media RSS <media:thumbnail> for the featured image.
		if tag := rssMediaThumbnailTag(thumbByPost[i]); tag != "" {
			b.WriteString(tag)
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

	mediaByPost := make([][]feedMediaContent, len(posts))
	thumbByPost := make([]string, len(posts))
	hasMedia := false
	for i, p := range posts {
		mediaByPost[i] = feedMediaContentForPost(c, p)
		thumbByPost[i] = feedThumbnailURLForPost(c, p)
		if len(mediaByPost[i]) > 0 || thumbByPost[i] != "" {
			hasMedia = true
		}
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><feed` + "\n")
	b.WriteString("\txmlns=\"http://www.w3.org/2005/Atom\"\n")
	b.WriteString("\txml:lang=\"")
	b.WriteString(xmlEscape(wp.BlogInfoLanguage(c)))
	b.WriteString("\"\n")
	b.WriteString("\txmlns:thr=\"http://purl.org/syndication/thread/1.0\"\n")
	if hasMedia {
		b.WriteString("\txmlns:media=\"http://search.yahoo.com/mrss/\"\n")
	}
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
	b.WriteString(wp.TheGenerator(c, "atom"))
	b.WriteString("\n")

	for i, p := range posts {
		link := postPermalink(home, p.Post)
		guid := feedItemGUID(c, home, p.ID, p.GUID)
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
		// P3.75: the_title_rss filter on Atom entry title.
		b.WriteString(applyTheTitleRSS(c, p.Title))
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
			b.WriteString(xmlEscape(cat.Name))
			b.WriteString("\" />\n")
		}
		b.WriteString("\t\t<summary type=\"html\"><![CDATA[")
		// P3.74: the_excerpt_rss filter on Atom summary.
		b.WriteString(applyTheExcerptRSS(c, p.ExcerptRSS))
		b.WriteString("]]></summary>\n\n")
		b.WriteString("\t\t\t\t\t<content type=\"html\" xml:base=\"")
		b.WriteString(xmlEscape(link))
		b.WriteString("\"><![CDATA[\n")
		// P3.73: Atom content via the_content_feed('atom').
		atomBody := applyTheContentFeed(c, p.ContentRSS, "atom")
		b.WriteString(atomBody)
		if atomBody != "" && !strings.HasSuffix(atomBody, "\n") {
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
			// P3.72: get_post_comments_feed_link( …, 'atom' ).
			b.WriteString(xmlEscape(wp.GetPostCommentsFeedLink(c, link, "atom")))
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
		// P3.69: Media RSS media:content on Atom entries too.
		for _, mc := range mediaByPost[i] {
			b.WriteString(rssMediaContentTag(mc))
		}
		// P3.70: Media RSS media:thumbnail on Atom entries too.
		if tag := rssMediaThumbnailTag(thumbByPost[i]); tag != "" {
			b.WriteString(tag)
		}
		b.WriteString("\t\t\t</entry>\n")
	}
	b.WriteString("</feed>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "application/atom+xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// renderFeedRDF emits a WordPress-shaped RDF/RSS 1.0 feed stub (feed-rdf.php, P3.33 / P3.89).
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

	// P3.89: rss_tag_pre before the RDF root (feed-rdf.php).
	if c != nil && c.Hooks != nil {
		c.Hooks.DoAction("rss_tag_pre", "rdf")
	}

	updatePeriod := "hourly"
	updateFrequency := "1"
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("rss_update_period", updatePeriod); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				updatePeriod = strings.TrimSpace(s)
			}
		}
		if v := c.Hooks.ApplyFilters("rss_update_frequency", updateFrequency); v != nil {
			switch x := v.(type) {
			case string:
				if strings.TrimSpace(x) != "" {
					updateFrequency = strings.TrimSpace(x)
				}
			case int:
				updateFrequency = strconv.Itoa(x)
			case int64:
				updateFrequency = strconv.FormatInt(x, 10)
			}
		}
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
	// P3.89: rdf_ns / wp_feed_namespaces('rdf') for extra xmlns attrs.
	b.WriteString(feedRDFExtraNamespaces(c))
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
	b.WriteString("\t")
	b.WriteString(xmlEscape(updatePeriod))
	b.WriteString("\n")
	b.WriteString("\t</sy:updatePeriod>\n")
	b.WriteString("\t<sy:updateFrequency>\n")
	b.WriteString("\t")
	b.WriteString(xmlEscape(updateFrequency))
	b.WriteString("\n")
	b.WriteString("\t</sy:updateFrequency>\n")
	b.WriteString("\t<sy:updateBase>2000-01-01T12:00+00:00</sy:updateBase>\n")
	// P3.56: rdf_header → the_generator (admin:generatorAgent).
	b.WriteString("\t")
	b.WriteString(wp.TheGenerator(c, "rdf"))
	b.WriteString("\n")
	// P3.89: rdf_header action after generator (plugins append channel markup).
	if c != nil && c.Hooks != nil {
		c.Hooks.DoAction("rdf_header")
	}
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
		// P3.75: the_title_rss filter on RDF item title.
		b.WriteString(xmlEscape(applyTheTitleRSS(c, p.Title)))
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
			b.WriteString(cat.Name)
			b.WriteString("]]></dc:subject>\n")
		}
		b.WriteString("\n")
		b.WriteString("\t\t<description><![CDATA[")
		// P3.74: the_excerpt_rss filter on RDF description.
		b.WriteString(applyTheExcerptRSS(c, p.ExcerptRSS))
		b.WriteString("]]></description>\n")
		// P3.63 / P3.73: RDF content:encoded via the_content_feed('rdf').
		if body, ok := rssContentEncodedBodyTyped(c, p, "rdf"); ok {
			b.WriteString("\t\t<content:encoded><![CDATA[")
			b.WriteString(body)
			b.WriteString("]]></content:encoded>\n")
		}
		// P3.89: rdf_item at the end of each item.
		if c != nil && c.Hooks != nil {
			c.Hooks.DoAction("rdf_item")
		}
		b.WriteString("</item>\n")
	}
	b.WriteString("</rdf:RDF>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "application/rdf+xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// feedRDFExtraNamespaces collects extra xmlns attrs from wp_feed_namespaces / rdf_ns (P3.89).
func feedRDFExtraNamespaces(c *wp.Context) string {
	if c == nil || c.Hooks == nil {
		return ""
	}
	extra := ""
	if v := c.Hooks.ApplyFilters("wp_feed_namespaces", "", "rdf"); v != nil {
		if s, ok := v.(string); ok {
			extra = s
		}
	}
	// rdf_ns historically printed xmlns attrs directly; capture via filter-shaped hooks
	// that append to a shared string is awkward, so plugins should use wp_feed_namespaces.
	c.Hooks.DoAction("rdf_ns")
	if strings.TrimSpace(extra) == "" {
		return ""
	}
	if !strings.HasSuffix(extra, "\n") {
		extra += "\n"
	}
	return extra
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

// feedMediaContent is a Media RSS media:content descriptor (P3.69).
type feedMediaContent struct {
	URL    string
	Type   string
	Medium string
	Length string
}

// feedCategory is one RSS/Atom/RDF item category (get_the_category_rss, P3.59/P3.71).
type feedCategory struct {
	Name   string
	Domain string // taxonomy slug: "category" or "post_tag" (WXR-style domain)
}

type feedPost struct {
	wp.Post
	AuthorName   string
	AuthorURL    string
	Categories   []feedCategory
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
		fp.Categories = feedCategories(c, p.ID)
		fp.CommentCount = feedCommentCount(c, p.ID)
		fp.ExcerptRSS = feedExcerptRSS(p)
		// Feed content runs the_content then the_content_feed at emit time (P3.73).
		// Skip post-save KSES so data: URIs in fixture video blocks match WP.
		rendered := wp.TheContent(c, p.Content)
		fp.ContentRSS = strings.TrimLeft(rendered, " \t\r")
		if strings.TrimSpace(p.Password) == "" {
			fp.Enclosures = feedEnclosuresForPost(c, p.ID)
		}
		out = append(out, fp)
	}
	return out, nil
}

// applyTheContentFeed mirrors get_the_content_feed post-processing (P3.73):
// escape CDATA terminators, then apply the_content_feed filter.
func applyTheContentFeed(c *wp.Context, content, feedType string) string {
	content = strings.ReplaceAll(content, "]]>", "]]&gt;")
	if feedType == "" {
		feedType = "rss2"
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("the_content_feed", content, feedType); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return content
}

// applyTheExcerptRSS mirrors the_excerpt_rss() (P3.74): escape CDATA
// terminators, then apply the_excerpt_rss filter.
func applyTheExcerptRSS(c *wp.Context, excerpt string) string {
	excerpt = strings.ReplaceAll(excerpt, "]]>", "]]&gt;")
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("the_excerpt_rss", excerpt); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return excerpt
}

// applyTheTitleRSS mirrors get_the_title_rss() (P3.75): apply the_title_rss
// then escape CDATA terminators for Atom HTML titles.
func applyTheTitleRSS(c *wp.Context, title string) string {
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("the_title_rss", title); v != nil {
			if s, ok := v.(string); ok {
				title = s
			} else {
				s := strings.TrimSpace(fmt.Sprint(v))
				if s != "" && s != "<nil>" {
					title = s
				}
			}
		}
	}
	return strings.ReplaceAll(title, "]]>", "]]&gt;")
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
// should be emitted (P3.63 / P3.73). Omitted when rss_use_excerpt is on. When
// full content is empty, falls back to the excerpt (WP feed-rss2.php).
func rssContentEncodedBody(c *wp.Context, p feedPost) (body string, ok bool) {
	return rssContentEncodedBodyTyped(c, p, "rss2")
}

func rssContentEncodedBodyTyped(c *wp.Context, p feedPost, feedType string) (body string, ok bool) {
	if rssUseExcerpt(c) {
		return "", false
	}
	raw := p.ContentRSS
	if len(raw) == 0 {
		raw = p.ExcerptRSS
	}
	return applyTheContentFeed(c, raw, feedType), true
}

// rss2ContentEncodedTag emits the RSS2 content:encoded element, or "" when omitted.
func rss2ContentEncodedTag(c *wp.Context, p feedPost) string {
	body, ok := rssContentEncodedBodyTyped(c, p, "rss2")
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

// feedMediaContentForPost builds Media RSS media:content entries from the
// featured image and enclosure meta (P3.69).
func feedMediaContentForPost(c *wp.Context, p feedPost) []feedMediaContent {
	var out []feedMediaContent
	seen := map[string]bool{}
	if thumb := strings.TrimSpace(wp.GetPostThumbnailURL(c, p.ID)); thumb != "" {
		mime := attachmentMimeType(c, wp.GetPostThumbnailID(c, p.ID))
		medium := mediaMediumFromType(mime)
		if medium == "" {
			medium = "image"
		}
		out = append(out, feedMediaContent{URL: thumb, Type: mime, Medium: medium})
		seen[thumb] = true
	}
	for _, enc := range p.Enclosures {
		url := strings.TrimSpace(enc.URL)
		if url == "" || seen[url] {
			continue
		}
		seen[url] = true
		mime := strings.TrimSpace(enc.Type)
		out = append(out, feedMediaContent{
			URL:    url,
			Type:   mime,
			Medium: mediaMediumFromType(mime),
			Length: enc.Length,
		})
	}
	return out
}

// feedThumbnailURLForPost returns the featured image URL for media:thumbnail (P3.70).
func feedThumbnailURLForPost(c *wp.Context, p feedPost) string {
	return strings.TrimSpace(wp.GetPostThumbnailURL(c, p.ID))
}

func rssMediaThumbnailTag(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	return `<media:thumbnail url="` + xmlEscape(url) + `" />` + "\n"
}

func attachmentMimeType(c *wp.Context, attachmentID int64) string {
	if c == nil || c.DB == nil || attachmentID <= 0 {
		return ""
	}
	q, args, err := c.DB.Prepare(
		"SELECT post_mime_type FROM "+c.DB.Posts()+" WHERE ID = %d",
		attachmentID,
	)
	if err != nil {
		return ""
	}
	v, err := c.DB.GetVar(q, args...)
	if err != nil || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case []byte:
		return strings.TrimSpace(string(t))
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func mediaMediumFromType(mime string) string {
	mime = strings.ToLower(strings.TrimSpace(mime))
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "audio/"):
		return "audio"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	default:
		return ""
	}
}

func rssMediaContentTag(mc feedMediaContent) string {
	if strings.TrimSpace(mc.URL) == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<media:content url="`)
	b.WriteString(xmlEscape(mc.URL))
	b.WriteString(`"`)
	if mc.Type != "" {
		b.WriteString(` type="`)
		b.WriteString(xmlEscape(mc.Type))
		b.WriteString(`"`)
	}
	if mc.Medium != "" {
		b.WriteString(` medium="`)
		b.WriteString(xmlEscape(mc.Medium))
		b.WriteString(`"`)
	}
	if mc.Length != "" && mc.Length != "0" {
		b.WriteString(` fileSize="`)
		b.WriteString(xmlEscape(mc.Length))
		b.WriteString(`"`)
	}
	b.WriteString(` />` + "\n")
	return b.String()
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

// feedCategories returns category + tag terms for a post (get_the_category_rss, P3.59/P3.71).
// Domain is the taxonomy slug (category / post_tag), matching WXR category domain=.
func feedCategories(c *wp.Context, postID int64) []feedCategory {
	out := []feedCategory{}
	if c == nil || c.DB == nil || postID <= 0 {
		return out
	}
	// WP get_the_category_rss merges get_the_category() and get_the_tags().
	// Categories first, then tags (stable unique-by-name like PHP array_unique).
	q, args, err := c.DB.Prepare(
		"SELECT t.name, tt.taxonomy FROM "+c.DB.Terms()+" t "+
			"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
			"INNER JOIN "+c.DB.TermRelationships()+" tr ON tr.term_taxonomy_id = tt.term_taxonomy_id "+
			"WHERE tr.object_id = %d AND tt.taxonomy IN (%s, %s) "+
			"ORDER BY FIELD(tt.taxonomy, %s, %s), t.name ASC",
		postID, "category", "post_tag", "category", "post_tag",
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
		name := strings.TrimSpace(asFeedString(row["name"]))
		domain := strings.TrimSpace(asFeedString(row["taxonomy"]))
		if name == "" || seen[name] {
			continue
		}
		if domain != "category" && domain != "post_tag" {
			domain = "category"
		}
		seen[name] = true
		out = append(out, feedCategory{Name: name, Domain: domain})
	}
	return out
}

// feedCategoryNames returns unique category + tag names (P3.59).
func feedCategoryNames(c *wp.Context, postID int64) []string {
	cats := feedCategories(c, postID)
	out := make([]string, 0, len(cats))
	for _, cat := range cats {
		out = append(out, cat.Name)
	}
	return out
}

func asFeedString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

// feedCommentsLink builds the RSS <comments> URL (comments_link_feed / get_comments_link, P3.61).
// Appends #comments when the post has comments, else #respond; runs comments_link_feed filter.
// feedItemGUID resolves the RSS/Atom item guid value via get_the_guid (P3.62 / P3.76).
// Prefers the stored post GUID (often a pretty permalink URL on fixture/import
// data); otherwise falls back to home/?p={id}. Runs the get_the_guid filter.
func feedItemGUID(c *wp.Context, home string, id int64, storedGUID string) string {
	guid := strings.TrimSpace(storedGUID)
	if guid == "" {
		home = strings.TrimRight(home, "/")
		guid = home + "/?p=" + strconv.FormatInt(id, 10)
	}
	// P3.76: apply_filters( 'get_the_guid', $post_guid, $post_id ).
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_the_guid", guid, id); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				guid = s
			} else {
				s := strings.TrimSpace(fmt.Sprint(v))
				if s != "" && s != "<nil>" {
					guid = s
				}
			}
		}
	}
	return guid
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

// postCommentsFeedLink builds get_post_comments_feed_link for pretty permalinks (P3.72 / P3.148).
// Delegates to wp.GetPostCommentsFeedLink (filter + default rss2).
func postCommentsFeedLink(permalink, feed string) string {
	return wp.GetPostCommentsFeedLink(nil, permalink, feed)
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
	var n int64
	switch v := row["comment_count"].(type) {
	case int64:
		n = v
	case int:
		n = int64(v)
	case []byte:
		n, _ = strconv.ParseInt(string(v), 10, 64)
	case string:
		n, _ = strconv.ParseInt(v, 10, 64)
	}
	// P3.68: get_comments_number filter (used for slash:comments / thr:count).
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_comments_number", n, postID); v != nil {
			switch t := v.(type) {
			case int64:
				return t
			case int:
				return int64(t)
			case string:
				if parsed, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil {
					return parsed
				}
			}
		}
	}
	return n
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

// tryTrailingSlashRedirect issues a 301 when a pretty content path disagrees
// with permalink_structure trailing-slash policy (WP redirect_canonical subset).
// P3.38: also covers /page/N home pagination and emits an absolute Location.
// P3.79: also strips a trailing slash when the structure does not use one, and
// collapses repeated slashes in the request path.
func (f *Front) tryTrailingSlashRedirect(w http.ResponseWriter, r *http.Request, c *wp.Context, parsed *wp.ParseResult) bool {
	if r == nil || r.URL == nil || parsed == nil {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	path := r.URL.Path
	if path == "" {
		return false
	}
	// P3.79: collapse // in path (keep leading slash).
	if cleaned := collapsePathSlashes(path); cleaned != path {
		target := cleaned
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, absoluteCanonicalURL(c, target), http.StatusMovedPermanently)
		return true
	}
	// Query-style URLs (?p=, ?page_id=) do not use trailing slashes.
	if r.URL.RawQuery != "" && (strings.Contains(r.URL.RawQuery, "p=") ||
		strings.Contains(r.URL.RawQuery, "page_id=") ||
		strings.Contains(r.URL.RawQuery, "attachment_id=") ||
		strings.Contains(r.URL.RawQuery, "feed=")) {
		return false
	}
	usesSlash := permalinkUsesTrailingSlash(c)
	if path == "/" {
		return false
	}
	if !needsTrailingSlashCanonical(parsed) {
		return false
	}
	hasSlash := strings.HasSuffix(path, "/")
	var target string
	switch {
	case usesSlash && !hasSlash:
		target = path + "/"
	case !usesSlash && hasSlash:
		// P3.79: strip trailing slash when structure omits it.
		target = strings.TrimRight(path, "/")
		if target == "" {
			target = "/"
		}
	default:
		return false
	}
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, absoluteCanonicalURL(c, target), http.StatusMovedPermanently)
	return true
}

// collapsePathSlashes replaces repeated "/" in a URL path with a single slash (P3.79).
func collapsePathSlashes(path string) string {
	if path == "" || !strings.Contains(path, "//") {
		return path
	}
	var b strings.Builder
	b.Grow(len(path))
	prevSlash := false
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '/' {
			if prevSlash {
				continue
			}
			prevSlash = true
		} else {
			prevSlash = false
		}
		b.WriteByte(c)
	}
	out := b.String()
	if out == "" {
		return "/"
	}
	return out
}

// tryAttachmentFileRedirect issues a 301 to the media file when attachment
// pages are disabled (wp_attachment_pages_enabled=0, P3.109 / redirect_canonical).
func (f *Front) tryAttachmentFileRedirect(w http.ResponseWriter, c *wp.Context, post *singularPost) bool {
	if post == nil || post.Type != "attachment" {
		return false
	}
	if wp.AttachmentPagesEnabled(c) {
		return false
	}
	url := wp.GetAttachmentURL(c, post.ID)
	if strings.TrimSpace(url) == "" {
		return false
	}
	// Match WP redirect_canonical headers (UTF-8 charset; avoid net/http's utf-8).
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.Header().Set("Location", url)
	w.Header().Set("X-Redirect-By", "WordPress")
	w.WriteHeader(http.StatusMovedPermanently)
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
	// P3.81: always mark the query as 404 before rendering (body_class / conditionals).
	if c != nil {
		if c.Scope == nil {
			c.Scope = &wp.RequestScope{}
		}
		c.Scope.Is404 = true
	}
	// P3.16: prefer block theme templates/404.html when present.
	// P3.35: ensure header/footer template parts wrap the 404 content.
	if tpl, ok := wp.ResolveBlockTemplate(c, "404"); ok && strings.TrimSpace(tpl) != "" {
		tpl = apply404TemplateContentFilter(c, tpl)
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

// apply404TemplateContentFilter lets hooks adjust resolved 404 block markup (P3.81).
func apply404TemplateContentFilter(c *wp.Context, tpl string) string {
	if c == nil || c.Hooks == nil {
		return tpl
	}
	if v := c.Hooks.ApplyFilters("404_template_content", tpl); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return tpl
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
// P3.81: includes a search block and home link like common theme 404 patterns.
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
			`<!-- wp:paragraph --><p>The requested content could not be found. Try a search or return home.</p><!-- /wp:paragraph -->` +
			`<!-- wp:search {"label":"Search","buttonText":"Search"} /-->` +
			`<!-- wp:paragraph --><p><a href="/">Home</a></p><!-- /wp:paragraph -->` +
			`</main><!-- /wp:group -->`,
	), true
}

func (f *Front) render404BlockTemplate(w http.ResponseWriter, c *wp.Context, template string) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	stylesheet := activeStylesheet(c)
	env := blockRenderEnv(c)
	body := blocks.RenderWith(env, template)
	safe := wp.ProcessInteractivityHTML(wp.KsesPostFiltered(c, body))
	safe = ensureMainSkipTarget(safe)

	bodyClasses := "error404 wp-embed-responsive"
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html")
	b.WriteString(wp.LanguageAttributesHTML(c))
	b.WriteString(">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString(frontRobotsMetaTag(c))
	b.WriteString("<title>")
	b.WriteString(front404DocumentTitle(c, blogname))
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, c, home, blogname, "", "", "", "", "", "", wp.SiteIconMetaTags(c))
	b.WriteString(wp.EmojiHeadHTMLCtx(c))
	b.WriteString("</head>\n<body class=\"")
	b.WriteString(html.EscapeString(bodyClasses))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<a class=\"skip-link screen-reader-text\" id=\"wp-skip-link\" href=\"#wp--skip-link--target\">Skip to content</a>")
	b.WriteString("<div class=\"wp-site-blocks\">")
	b.WriteString(safe)
	b.WriteString("</div>\n")
	writeFrontDocumentClose(&b, c)

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) render404Shim(w http.ResponseWriter, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	if home == "" {
		home = "/"
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(front404DocumentTitle(c, blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "error404")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<main id=\"wp--skip-link--target\">\n")
	b.WriteString("<h1>Page not found</h1>\n")
	b.WriteString("<p>The requested content could not be found. Try a search or return home.</p>\n")
	// P3.81: classic shim search form (parity with block theme search pattern).
	action := home
	if !strings.HasSuffix(action, "/") {
		action += "/"
	}
	b.WriteString(`<form role="search" method="get" action="`)
	b.WriteString(html.EscapeString(action))
	b.WriteString(`" class="search-form wp-block-search">`)
	b.WriteString(`<label><span class="screen-reader-text">Search</span>`)
	b.WriteString(`<input type="search" name="s" value="" placeholder="Search"/></label>`)
	b.WriteString(`<input type="submit" value="Search"/>`)
	b.WriteString("</form>\n")
	b.WriteString(`<p><a href="`)
	b.WriteString(html.EscapeString(home))
	b.WriteString(`">Home</a></p>`)
	b.WriteString("\n</main>\n")
	writeFrontDocumentClose(&b, c)

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(b.String()))
}

// front404DocumentTitle builds the 404 <title> and honors document_title_parts (P3.81).
func front404DocumentTitle(c *wp.Context, blogname string) string {
	parts := map[string]string{
		"title":   "Page not found",
		"page":    "",
		"tagline": "",
		"site":    blogname,
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("document_title_parts", parts); v != nil {
			if m, ok := v.(map[string]string); ok {
				parts = m
			}
		}
	}
	title := strings.TrimSpace(parts["title"])
	if title == "" {
		title = "Page not found"
	}
	site := strings.TrimSpace(parts["site"])
	out := html.EscapeString(title)
	if site != "" {
		out += " &#8211; " + html.EscapeString(site)
	}
	return out
}

// pathEscapeSearch encodes a search term for /search/{term}/feed/ paths (P3.39).
func pathEscapeSearch(term string) string {
	return url.PathEscape(strings.TrimSpace(term))
}

// writeArchiveRelLinks emits <link rel="prev|next"> on paginated archives (P3.43 / P3.107 / P3.145).
// Not used on the blog home (corpus front-home) — only is_archive()-style views.
// P3.107: search archives preserve ?s= on paged prev/next (WP /page/2/?s=term).
// P3.145: get_pagenum_link filter + EscURL on hrefs via ArchiveRelPrevNextHTMLQueryCtx.
func writeArchiveRelLinks(b *strings.Builder, c *wp.Context, home string) {
	if b == nil || c == nil || c.Scope == nil {
		return
	}
	if !c.Scope.IsArchive && !c.Scope.IsSearch {
		return
	}
	base := wp.ArchiveBaseFromRequest(c, home)
	paged, maxPages := mainQueryPagination(c)
	q := wp.ArchiveSearchQuerySuffix(c)
	b.WriteString(wp.ArchiveRelPrevNextHTMLQueryCtx(c, base, paged, maxPages, q))
}

// frontRobotsMetaTag returns the <meta name='robots' …> line (wp_robots, P3.44 / P3.90 / P3.104 / P3.142 / P3.143).
// Applies the wp_robots filter after default directives matching WP core callbacks:
// noindex/nofollow when blog_public=0; wp_robots_no_robots (noindex+follow) on
// search/embed when public; max-image-preview:large when public.
// P3.142: accept map[string]string from filters; expose sensitive-page helper.
// P3.143: replytocom / unapproved+moderation-hash → wp_robots_no_robots (default-filters.php).
func frontRobotsMetaTag(c *wp.Context) string {
	robots := frontRobotsDirectives(c)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_robots", robots); v != nil {
			robots = coerceRobotsFilterValue(v)
		}
	}
	content := formatRobotsDirectives(robots)
	if content == "" {
		return ""
	}
	return "<meta name='robots' content='" + html.EscapeString(content) + "' />\n"
}

// coerceRobotsFilterValue normalizes wp_robots filter return values (P3.142).
func coerceRobotsFilterValue(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		return t
	case map[string]bool:
		out := map[string]any{}
		for k, val := range t {
			out[k] = val
		}
		return out
	case map[string]string:
		out := map[string]any{}
		for k, val := range t {
			out[k] = val
		}
		return out
	default:
		return map[string]any{}
	}
}

// frontRobotsDirectives builds the default wp_robots directive map (P3.90 / P3.104 / P3.142 / P3.143).
// Mirrors the default_filters.php callback order:
// wp_robots_noindex → noindex_embeds → noindex_search → max_image_preview_large,
// plus conditional wp_robots_no_robots for replytocom / unapproved preview (P3.143).
func frontRobotsDirectives(c *wp.Context) map[string]any {
	robots := map[string]any{}
	robots = wpRobotsNoindex(c, robots)
	robots = wpRobotsNoindexEmbeds(c, robots)
	robots = wpRobotsNoindexSearch(c, robots)
	robots = wpRobotsMaxImagePreviewLarge(c, robots)
	// P3.143: default-filters.php conditional add_filter( 'wp_robots', 'wp_robots_no_robots' ).
	if frontRobotsNeedsNoRobots(c) {
		robots = wpRobotsNoRobots(c, robots)
	}
	return robots
}

// frontRobotsNeedsNoRobots mirrors the default-filters.php condition that hooks
// wp_robots_no_robots for comment-reply and unapproved comment preview (P3.143).
func frontRobotsNeedsNoRobots(c *wp.Context) bool {
	if c == nil || c.Req == nil {
		return false
	}
	q := c.Req.URL.Query()
	// Comment reply link: ?replytocom=
	if q.Has("replytocom") {
		return true
	}
	// Unapproved comment preview: ?unapproved= & ?moderation-hash=
	if q.Has("unapproved") && q.Has("moderation-hash") {
		return true
	}
	return false
}

// wpRobotsNoindex mirrors wp_robots_noindex() (P3.142).
func wpRobotsNoindex(c *wp.Context, robots map[string]any) map[string]any {
	if optionTruthy(c, "blog_public") {
		return robots
	}
	return wpRobotsNoRobots(c, robots)
}

// wpRobotsNoindexEmbeds mirrors wp_robots_noindex_embeds() (P3.142).
func wpRobotsNoindexEmbeds(c *wp.Context, robots map[string]any) map[string]any {
	if c != nil && c.Scope != nil && c.Scope.IsEmbed {
		return wpRobotsNoRobots(c, robots)
	}
	return robots
}

// wpRobotsNoindexSearch mirrors wp_robots_noindex_search() (P3.142).
func wpRobotsNoindexSearch(c *wp.Context, robots map[string]any) map[string]any {
	if c != nil && c.Scope != nil && c.Scope.IsSearch {
		return wpRobotsNoRobots(c, robots)
	}
	return robots
}

// wpRobotsNoRobots mirrors wp_robots_no_robots() (P3.104 / P3.142).
func wpRobotsNoRobots(c *wp.Context, robots map[string]any) map[string]any {
	if robots == nil {
		robots = map[string]any{}
	}
	robots["noindex"] = true
	if optionTruthy(c, "blog_public") {
		robots["follow"] = true
	} else {
		robots["nofollow"] = true
	}
	return robots
}

// wpRobotsSensitivePage mirrors wp_robots_sensitive_page() (P3.142).
func wpRobotsSensitivePage(robots map[string]any) map[string]any {
	if robots == nil {
		robots = map[string]any{}
	}
	robots["noindex"] = true
	robots["noarchive"] = true
	return robots
}

// wpRobotsMaxImagePreviewLarge mirrors wp_robots_max_image_preview_large() (P3.142).
func wpRobotsMaxImagePreviewLarge(c *wp.Context, robots map[string]any) map[string]any {
	if robots == nil {
		robots = map[string]any{}
	}
	if optionTruthy(c, "blog_public") {
		robots["max-image-preview"] = "large"
	}
	return robots
}

// formatRobotsDirectives joins wp_robots directives into a meta content string (P3.90 / P3.104).
// Order mirrors PHP array insertion from the default filter chain:
// noindex, follow/nofollow, …, then max-image-preview.
func formatRobotsDirectives(robots map[string]any) string {
	if len(robots) == 0 {
		return ""
	}
	order := []string{"noindex", "follow", "nofollow", "noarchive", "nosnippet", "max-image-preview"}
	seen := map[string]bool{}
	var parts []string
	appendDir := func(k string, v any) {
		k = strings.TrimSpace(k)
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		switch t := v.(type) {
		case bool:
			if t {
				parts = append(parts, k)
			}
		case string:
			t = strings.TrimSpace(t)
			if t != "" {
				parts = append(parts, k+":"+t)
			}
		default:
			if v != nil {
				s := strings.TrimSpace(fmt.Sprint(v))
				if s != "" && s != "false" && s != "0" {
					if s == "true" || s == "1" {
						parts = append(parts, k)
					} else {
						parts = append(parts, k+":"+s)
					}
				}
			}
		}
	}
	for _, k := range order {
		if v, ok := robots[k]; ok {
			appendDir(k, v)
		}
	}
	var extras []string
	for k := range robots {
		if !seen[k] {
			extras = append(extras, k)
		}
	}
	sort.Strings(extras)
	for _, k := range extras {
		appendDir(k, robots[k])
	}
	return strings.Join(parts, ", ")
}

// writeFrontHeadLinks emits wp_head discovery links (feeds, api.w.org, oEmbed, canonical).
// Stylesheet <link> tags are omitted: the HTML normalizer strips them, and TT5 enqueues
// styles as inline <style> which we also normalize away (P5.19).
// extraFeedTitle/Href implement feed_links_extra for author/search/category/tag/singular (P3.39 / P3.103 / P3.128).

func hooksFor(c *wp.Context) *wp.Hooks {
	if c == nil {
		return nil
	}
	return c.Hooks
}

// writeFrontDocumentClose fires wp_footer then closes body/html (P3.118).
func writeFrontDocumentClose(b *strings.Builder, c *wp.Context) {
	if b == nil {
		return
	}
	wp.Footer(c)
	b.WriteString("</body>\n</html>\n")
}

// singularCommentsFeedLink builds feed_links_extra for singular post comments (P3.128).
// Mirrors feed_links_extra(): feed_links_extra_show_post_comments_feed +
// comments_open || pings_open || comment_count > 0.
func singularCommentsFeedLink(c *wp.Context, home, blogname, permalink string, post *singularPost) (title, href string) {
	if post == nil || home == "" || permalink == "" || strings.TrimSpace(post.Title) == "" {
		return "", ""
	}
	hooks := hooksFor(c)
	showComments := feedLinksShow(hooks, "feed_links_show_comments_feed", true)
	showPost := feedLinksShow(hooks, "feed_links_extra_show_post_comments_feed", showComments)
	if !showPost {
		return "", ""
	}
	openComments := strings.EqualFold(strings.TrimSpace(post.CommentStatus), "open")
	openPings := strings.EqualFold(strings.TrimSpace(post.PingStatus), "open")
	if !openComments && !openPings && post.CommentCount <= 0 {
		return "", ""
	}
	sep := "&raquo;"
	singleFmt := "%1$s %2$s %3$s Comments Feed"
	if hooks != nil {
		args := map[string]any{
			"separator":   sep,
			"singletitle": singleFmt,
		}
		if v := hooks.ApplyFilters("feed_links_extra_args", args); v != nil {
			if m, ok := v.(map[string]any); ok {
				if s, ok := m["separator"].(string); ok && s != "" {
					sep = s
				}
				if s, ok := m["singletitle"].(string); ok && s != "" {
					singleFmt = s
				}
			}
		}
	}
	title = strings.ReplaceAll(singleFmt, "%1$s", html.EscapeString(blogname))
	title = strings.ReplaceAll(title, "%2$s", sep)
	title = strings.ReplaceAll(title, "%3$s", html.EscapeString(post.Title))
	href = strings.TrimRight(permalink, "/") + "/feed/"
	return title, href
}

// termArchiveFeedLink builds feed_links_extra title/href for category or tag archives (P3.103).
func termArchiveFeedLink(c *wp.Context, blogname, home, taxonomy string, term *termInfo) (title, href string) {
	if term == nil || term.Slug == "" || home == "" {
		return "", ""
	}
	showFilter := "feed_links_extra_show_category_feed"
	kind := " Category Feed"
	basePath := "/category/"
	if taxonomy == "post_tag" || taxonomy == "tag" {
		showFilter = "feed_links_extra_show_tag_feed"
		kind = " Tag Feed"
		basePath = "/tag/"
	}
	show := true
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters(showFilter, show); v != nil {
			switch t := v.(type) {
			case bool:
				show = t
			case int:
				show = t != 0
			case string:
				show = t != "" && t != "0" && !strings.EqualFold(t, "false")
			}
		}
	}
	if !show {
		return "", ""
	}
	name := term.Name
	if name == "" {
		name = term.Slug
	}
	title = html.EscapeString(blogname) + " &raquo; " + html.EscapeString(name) + kind
	href = home + basePath + term.Slug + "/feed/"
	if c != nil && c.Hooks != nil {
		filterName := "category_feed_link"
		if taxonomy == "post_tag" || taxonomy == "tag" {
			filterName = "tag_feed_link"
		}
		if v := c.Hooks.ApplyFilters(filterName, href, "rss2"); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				href = s
			}
		}
	}
	return title, href
}

// feedLinksShow applies a feed_links_* bool filter (P3.117).
func feedLinksShow(hooks *wp.Hooks, tag string, def bool) bool {
	if hooks == nil {
		return def
	}
	show := def
	if v := hooks.ApplyFilters(tag, show); v != nil {
		switch t := v.(type) {
		case bool:
			show = t
		case int:
			show = t != 0
		case string:
			show = t != "" && t != "0" && !strings.EqualFold(t, "false")
		}
	}
	return show
}

// formatFeedLinkTitleAttr builds esc_attr(sprintf(fmt, blogname, sep)) for feed_links (P3.117).
// Named entities in sep (e.g. &raquo;) are preserved unescaped to match prior Go/WP markup.
func formatFeedLinkTitleAttr(format, blogname, sep string) string {
	out := format
	out = strings.ReplaceAll(out, "%1$s", html.EscapeString(blogname))
	out = strings.ReplaceAll(out, "%2$s", sep)
	// Fallback order if format uses plain sprintf %s %s.
	if !strings.Contains(format, "%1$s") {
		out = fmt.Sprintf(format, html.EscapeString(blogname), sep)
	}
	return out
}

func writeFrontHeadLinks(b *strings.Builder, c *wp.Context, home, blogname, permalink, restPath, title, extraFeedTitle, extraFeedHref, shortlink string, siteIconHTML string) {
	hooks := hooksFor(c)
	// P3.119: fire wp_head (enqueue_scripts / print_styles / print_head_scripts).
	wp.Head(c)
	if home == "" || b == nil {
		return
	}
	// P3.137: wp_resource_hints (priority 2); default empty without remote deps.
	b.WriteString(wp.ResourceHintsHTML(c))
	// P3.117: feed_links() show filters + args (separator / titles).
	showPostsFeed := feedLinksShow(hooks, "feed_links_show_posts_feed", true)
	showCommentsFeed := feedLinksShow(hooks, "feed_links_show_comments_feed", true)
	sep := "&raquo;"
	feedTitleFmt := "%1$s %2$s Feed"
	comsTitleFmt := "%1$s %2$s Comments Feed"
	if hooks != nil {
		args := map[string]any{
			"separator": sep,
			"feedtitle": feedTitleFmt,
			"comstitle": comsTitleFmt,
		}
		if v := hooks.ApplyFilters("feed_links_args", args); v != nil {
			if m, ok := v.(map[string]any); ok {
				if s, ok := m["separator"].(string); ok && s != "" {
					sep = s
				}
				if s, ok := m["feedtitle"].(string); ok && s != "" {
					feedTitleFmt = s
				}
				if s, ok := m["comstitle"].(string); ok && s != "" {
					comsTitleFmt = s
				}
			}
		}
	}
	// P3.147: feed_links / feed_links_extra alternates via feed_content_type + EscURL.
	feedType := wp.FeedContentType(c, "")
	if showPostsFeed {
		b.WriteString("<link rel=\"alternate\" type=\"")
		b.WriteString(feedType)
		b.WriteString("\" title=\"")
		b.WriteString(formatFeedLinkTitleAttr(feedTitleFmt, blogname, sep))
		b.WriteString("\" href=\"")
		b.WriteString(wp.EscURL(wp.GetFeedLink(c, home, "")))
		b.WriteString("\" />\n")
	}
	if showCommentsFeed {
		b.WriteString("<link rel=\"alternate\" type=\"")
		b.WriteString(feedType)
		b.WriteString("\" title=\"")
		b.WriteString(formatFeedLinkTitleAttr(comsTitleFmt, blogname, sep))
		b.WriteString("\" href=\"")
		b.WriteString(wp.EscURL(wp.GetFeedLink(c, home, "comments_rss2")))
		b.WriteString("\" />\n")
	}
	if permalink != "" && title != "" && showCommentsFeed {
		b.WriteString("<link rel=\"alternate\" type=\"")
		b.WriteString(feedType)
		b.WriteString("\" title=\"")
		b.WriteString(html.EscapeString(blogname))
		b.WriteString(" ")
		b.WriteString(sep)
		b.WriteString(" ")
		b.WriteString(html.EscapeString(title))
		b.WriteString(" Comments Feed\" href=\"")
		b.WriteString(wp.EscURL(strings.TrimRight(permalink, "/") + "/feed/"))
		b.WriteString("\" />\n")
	}
	if extraFeedTitle != "" && extraFeedHref != "" {
		b.WriteString("<link rel=\"alternate\" type=\"")
		b.WriteString(feedType)
		b.WriteString("\" title=\"")
		b.WriteString(extraFeedTitle) // may include &#8220; entities like WP
		b.WriteString("\" href=\"")
		b.WriteString(wp.EscURL(extraFeedHref))
		b.WriteString("\" />\n")
	}
	// P3.20 / P3.92 / P3.146: wp_oembed_add_discovery_links (EscURL hrefs + embeddable gate).
	// Front head only passes a singular permalink; embeddable=true mirrors is_post_embeddable.
	b.WriteString(wp.OEmbedDiscoveryLinksEmbeddable(hooks, home, permalink, permalink != ""))
	b.WriteString("<link rel=\"https://api.w.org/\" href=\"")
	b.WriteString(html.EscapeString(home + "/wp-json/"))
	b.WriteString("\" />\n")
	if restPath != "" {
		b.WriteString("<link rel=\"alternate\" title=\"JSON\" type=\"application/json\" href=\"")
		b.WriteString(html.EscapeString(home + "/wp-json" + restPath))
		b.WriteString("\" />\n")
	}
	// P3.135: rsd_link() — EditURI via site_url( xmlrpc.php?rsd, rpc ).
	b.WriteString(wp.RsdLink(c))
	// P3.121: wp_generator → the_generator( wp_generator_type ).
	b.WriteString(wp.MetaGeneratorXHTMLFor(c))
	b.WriteString("\n")
	// P3.42 / P3.105 / P3.144: rel=canonical via CanonicalTagHTML (EscURL).
	if tag := wp.CanonicalTagHTML(permalink); tag != "" {
		b.WriteString(tag)
	}
	if shortlink != "" {
		b.WriteString("<link rel='shortlink' href='")
		b.WriteString(html.EscapeString(shortlink))
		b.WriteString("' />\n")
	}
	// P3.112: wp_site_icon meta tags (empty when no custom icon).
	if siteIconHTML != "" {
		b.WriteString(siteIconHTML)
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
	// P3.133: shared RestOutputLinkHeader (get_rest_url + rest_url filter + Add).
	wp.RestOutputLinkHeader(w, c)
}

// setFrontSingularLinks mirrors WP Link headers for singular views (P5.17).
// X-Pingback is intentionally omitted: XML-RPC/pingbacks are out of scope (P3.41 / P3.98 / P3.131).
func setFrontSingularLinks(w http.ResponseWriter, c *wp.Context, post *singularPost) {
	if post == nil {
		return
	}
	// P3.133: api.w.org discovery via rest_output_link_header.
	wp.RestOutputLinkHeader(w, c)
	restType := "posts"
	if post.Type == "page" {
		restType = "pages"
	}
	if post.Type == "post" || post.Type == "page" {
		if alt := wp.RestURL(c, fmt.Sprintf("/wp/v2/%s/%d", restType, post.ID)); alt != "" {
			w.Header().Add("Link", fmt.Sprintf(
				`<%s>; rel="alternate"; title="JSON"; type="application/json"`,
				wp.SanitizeURL(alt)))
		}
	}
	// P3.106 / P3.130: shortlink Link header via wp_shortlink_header (query context).
	wp.ShortlinkHeader(w, c, post.ID)
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
	// P3.128: feed_links_extra singular comments feed gates.
	CommentStatus string
	PingStatus    string
	CommentCount  int64
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
		CommentStatus:    p.CommentStatus,
		PingStatus:       p.PingStatus,
		CommentCount:     feedCommentCount(c, p.ID),
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
		Paged:        paged,
		MaxPages:     maxPages,
		DebugDisplay: wp.IsDebugDisplay(c),
		GetAttachmentURL: func(id int64) string {
			return wp.GetAttachmentURL(c, id)
		},
		GetAttachmentDimensions: func(id int64) (width, height int) {
			return wp.AttachmentDimensions(c, id)
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
					Label:  p.Title,
					URL:    wp.PagePermalink(c, p),
					ID:     p.ID,
					Parent: p.Parent,
				})
			}
			return out
		},
		CurrentPageID: func() int64 {
			// Queried page ID when rendering a singular page (P5.75).
			if c.Scope != nil && c.Scope.IsPage && c.Scope.QueriedObjectID > 0 {
				return c.Scope.QueriedObjectID
			}
			return 0
		}(),
		FrontPageID: func() int64 {
			v := optionString(c, "page_on_front", "0")
			n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			return n
		}(),
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
			// P5.85: non-inherit query block args (search/author/order/sticky/exclude/format).
			if !q.Inherit {
				if q.Search != "" {
					search = q.Search
				}
				if q.Author > 0 {
					authorID = q.Author
				}
			}
			// Sticky posts are prepended on page 1 only (WP main query / P3.26).
			stickyFirst := postType == "post" && search == "" && year == 0 && authorID == 0 && tagSlug == "" && catSlug == "" && paged <= 1
			ignoreSticky := false
			postNotIn := append([]int64(nil), q.Exclude...)
			var postIn []int64
			switch q.Sticky {
			case "only":
				stickyFirst = false
				ignoreSticky = true
				postIn = wp.StickyPostIDs(c)
				if len(postIn) == 0 {
					postIn = []int64{0}
				}
			case "exclude":
				stickyFirst = false
				ignoreSticky = true
				postNotIn = append(postNotIn, wp.StickyPostIDs(c)...)
			case "ignore":
				stickyFirst = false
				ignoreSticky = true
			}
			order := q.Order
			orderBy := q.OrderBy
			if order == "" {
				order = "desc"
			}
			if orderBy == "" {
				orderBy = "date"
			}
			var formatIn []string
			formatStandard := false
			for _, f := range q.Format {
				f = strings.TrimSpace(strings.ToLower(f))
				if f == "" {
					continue
				}
				if f == "standard" {
					formatStandard = true
					continue
				}
				if !strings.HasPrefix(f, "post-format-") {
					f = "post-format-" + f
				}
				formatIn = append(formatIn, f)
			}
			qa := wp.QueryArgs{
				PostType:           postType,
				PostStatus:         "publish",
				PostsPerPage:       limit,
				Offset:             offset,
				Search:             search,
				Year:               year,
				MonthNum:           month,
				Day:                day,
				Author:             authorID,
				Tag:                tagSlug,
				Category:           catSlug,
				StickyFirst:        stickyFirst,
				IgnoreSticky:       ignoreSticky,
				PostIn:             postIn,
				PostNotIn:          postNotIn,
				Order:              order,
				OrderBy:            orderBy,
				PostFormatIn:       formatIn,
				PostFormatStandard: formatStandard,
				CategoryIn:         q.CategoryIn,
				TagIn:              q.TagIn,
				ParentIn:           q.Parents,
			}
			posts, err := wp.GetPosts(c, qa)
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
					Excerpt:          wp.GetTheExcerpt(c, &p),
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
		// P5.98 / P2.108: footnotes content sanitized like wp_kses_post (+ hooks).
		SanitizeHTML: func(html string) string { return wp.KsesPostFiltered(c, html) },
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
			// P3.111: theme_mod custom_logo, then site_logo option.
			id := wp.CustomLogoID(c)
			if id <= 0 {
				return ""
			}
			return wp.GetAttachmentURL(c, id)
		},
		// P5.120: attachment ID for site-logo width→height aspect.
		SiteLogoID: func() int64 {
			return wp.CustomLogoID(c)
		},
		HomeURL: func() string {
			return strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		},
		IsFrontPage: func() bool {
			// Front block templates set this via env flag when rendering home.
			return c != nil && c.Scope != nil && c.Scope.IsFrontPage
		},
		IsHome: func() bool {
			return c != nil && c.Scope != nil && c.Scope.IsHome
		},
		IsPaged: func() bool {
			return c != nil && c.Scope != nil && c.Scope.Paged > 1
		},
		QueriedObjectID: func() int64 {
			if c == nil || c.Scope == nil {
				return 0
			}
			return c.Scope.QueriedObjectID
		},
		PageForPostsID: func() int64 {
			raw := optionString(c, "page_for_posts", "0")
			id, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
			return id
		},
		// P5.74: core/loginout resolvers.
		// P6.52: login_form_top/middle/bottom via ApplyFilter.
		ApplyFilter: func(tag string, value any, args ...any) any {
			if c == nil || c.Hooks == nil {
				return value
			}
			return c.Hooks.ApplyFilters(tag, value, args...)
		},
		// P5.131: wp_hash for core/legacy-widget instance hash_equals.
		HashNonce: func(data string) string {
			if c == nil || c.Auth == nil {
				return ""
			}
			return c.Auth.Hash(data, "auth")
		},
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
			var postAuthor int64
			if p, err := wp.GetPostByID(c, postID); err == nil && p != nil {
				postAuthor = p.Author
			}
			for _, cm := range rows {
				nicename := ""
				if cm.UserID > 0 {
					if u, err := c.GetUserBy("id", cm.UserID); err == nil && u != nil {
						nicename = u.Nicename
					}
				}
				out = append(out, blocks.CommentItem{
					ID:           cm.ID,
					Parent:       cm.Parent,
					Author:       cm.Author,
					AuthorEmail:  cm.AuthorEmail,
					AuthorURL:    cm.AuthorURL,
					Content:      wp.KsesPostFiltered(c, cm.Content),
					Date:         cm.Date,
					UserID:       cm.UserID,
					UserNicename: nicename,
					PostAuthor:   postAuthor,
					Type:         cm.Type,
				})
			}
			return out
		},
		CommentOrder: optionString(c, "comment_order", "asc"),
		AdjacentPost: func(postID int64, previous bool, taxonomy string) *blocks.QueryPost {
			// P3.122 / P5.97: get_adjacent_post + in_same_term + join/where/sort filters.
			adj, err := wp.GetAdjacentPost(c, postID, previous, taxonomy)
			if err != nil || adj == nil {
				return nil
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			return &blocks.QueryPost{
				ID:    adj.ID,
				Title: adj.Title,
				Link:  wp.AdjacentPostPermalink(home, adj),
				Date:  adj.Date,
			}
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
		// P5.42 / P5.122: widget-style dynamic blocks.
		ListArchives: func(archiveType string) []blocks.ArchiveItem {
			if c == nil || c.DB == nil {
				return nil
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			archiveType = strings.TrimSpace(archiveType)
			if archiveType == "" {
				archiveType = "monthly"
			}
			if archiveType == "yearly" {
				q, args, err := c.DB.Prepare(
					"SELECT YEAR(post_date) AS y, COUNT(*) AS n FROM "+c.DB.Posts()+
						" WHERE post_type = %s AND post_status = %s "+
						"GROUP BY YEAR(post_date) ORDER BY post_date DESC LIMIT %d",
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
					if y <= 0 {
						continue
					}
					out = append(out, blocks.ArchiveItem{
						Label: strconv.Itoa(y),
						URL:   fmt.Sprintf("%s/%04d/", home, y),
						Count: int(rowInt64DB(row["n"])),
					})
				}
				return out
			}
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
		HasPublishedPosts: func() bool {
			if c == nil || c.DB == nil {
				return false
			}
			q, args, err := c.DB.Prepare(
				"SELECT 1 FROM "+c.DB.Posts()+" WHERE post_type = %s AND post_status = %s LIMIT 1",
				"post", "publish",
			)
			if err != nil {
				return false
			}
			v, err := c.DB.GetVar(q, args...)
			return err == nil && v != nil && fmt.Sprint(v) != "" && fmt.Sprint(v) != "<nil>"
		},
		// P5.124: html5 search-element opt-in for core/search landmark wrapper.
		ThemeSupports: func(feature string, args ...any) bool {
			return wp.CurrentThemeSupports(c, feature, args...)
		},
		ListTagCloud: func(taxonomy string, number int) []blocks.TagCloudItem {
			if c == nil || c.DB == nil {
				return nil
			}
			taxonomy = strings.TrimSpace(taxonomy)
			if taxonomy == "" {
				taxonomy = "post_tag"
			}
			if number <= 0 {
				number = 45
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			q, args, err := c.DB.Prepare(
				"SELECT t.name, t.slug, tt.count FROM "+c.DB.Terms()+" t "+
					"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
					"WHERE tt.taxonomy = %s AND tt.count > 0 ORDER BY tt.count DESC, t.name ASC LIMIT %d",
				taxonomy, number,
			)
			if err != nil {
				return nil
			}
			rows, err := c.DB.GetResults(q, args...)
			if err != nil {
				return nil
			}
			basePath := "/tag/"
			if taxonomy == "category" {
				basePath = "/category/"
			}
			out := make([]blocks.TagCloudItem, 0, len(rows))
			for _, row := range rows {
				slug := rowStringDB(row["slug"])
				out = append(out, blocks.TagCloudItem{
					Name:  rowStringDB(row["name"]),
					URL:   home + basePath + slug + "/",
					Count: int(rowInt64DB(row["count"])),
				})
			}
			return out
		},
		// P5.81: core/categories term list.
		ListCategories: func(taxonomy string) []blocks.CategoryItem {
			if c == nil || c.DB == nil {
				return nil
			}
			taxonomy = strings.TrimSpace(taxonomy)
			if taxonomy == "" {
				taxonomy = "category"
			}
			home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
			q, args, err := c.DB.Prepare(
				"SELECT t.term_id, t.name, t.slug, tt.count, tt.parent FROM "+c.DB.Terms()+" t "+
					"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
					"WHERE tt.taxonomy = %s ORDER BY t.name ASC LIMIT %d",
				taxonomy, 200,
			)
			if err != nil {
				return nil
			}
			rows, err := c.DB.GetResults(q, args...)
			if err != nil {
				return nil
			}
			prefix := "/category/"
			if taxonomy == "post_tag" {
				prefix = "/tag/"
			}
			out := make([]blocks.CategoryItem, 0, len(rows))
			for _, row := range rows {
				slug := rowStringDB(row["slug"])
				out = append(out, blocks.CategoryItem{
					ID:     rowInt64DB(row["term_id"]),
					Name:   rowStringDB(row["name"]),
					URL:    home + prefix + slug + "/",
					Count:  int(rowInt64DB(row["count"])),
					Parent: rowInt64DB(row["parent"]),
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
					ID:          cm.ID,
					Author:      cm.Author,
					AuthorURL:   cm.AuthorURL,
					AuthorEmail: cm.AuthorEmail,
					Content:     strings.TrimSpace(stripTagsSimple(cm.Content)),
					Date:        cm.Date,
					PostID:      cm.PostID,
					PostTitle:   title,
					PostURL:     link + "#comment-" + strconv.FormatInt(cm.ID, 10),
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
