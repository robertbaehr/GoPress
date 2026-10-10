package server

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/blocks"
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

	// redirect_canonical: enforce trailing slash on pretty permalinks (P3.15).
	if f.tryTrailingSlashRedirect(w, r, c, parsed) {
		return
	}

	if isFeedVars(parsed.QueryVars) {
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

	// Category archives: list posts in term (shim until full archive templates).
	if cat := parsed.QueryVars["category_name"]; cat != "" {
		f.renderTermArchive(w, c, "category", cat)
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
	case "feed", "rss", "rss2", "atom":
		return true
	default:
		return false
	}
}

func (f *Front) renderFeed(w http.ResponseWriter, r *http.Request, c *wp.Context, feed string) {
	switch strings.ToLower(feed) {
	case "atom":
		f.renderFeedAtom(w, r, c)
	default:
		f.renderFeedRSS2(w, c)
	}
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

// renderSearch serves /?s= front search results (P3.21).
func (f *Front) renderSearch(w http.ResponseWriter, c *wp.Context, term string) {
	if c != nil && c.Scope != nil {
		c.Scope.IsSearch = true
		if term == "" {
			term = c.Scope.SearchQuery
		} else {
			c.Scope.SearchQuery = term
		}
	}
	slugs := wp.SearchTemplateSlugs()
	if tpl, ok := wp.ResolveBlockTemplate(c, slugs...); ok && strings.TrimSpace(tpl) != "" {
		f.renderSearchBlockTemplate(w, c, tpl, term)
		return
	}
	f.renderSearchShim(w, c, term)
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
	b.WriteString("<meta name='robots' content='max-image-preview:large' />\n")
	b.WriteString("<title>")
	b.WriteString(title)
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "")
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
	b.WriteString("<meta name='robots' content='max-image-preview:large' />\n")
	b.WriteString("<title>")
	b.WriteString(pageTitle)
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "")
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
	b.WriteString("<meta name='robots' content='max-image-preview:large' />\n")
	b.WriteString("<title>")
	b.WriteString(pageTitle)
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "")
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
	b.WriteString("<meta name='robots' content='max-image-preview:large' />\n")
	b.WriteString("<title>")
	b.WriteString(pageTitle)
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "")
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
	if stylesheet != "" {
		bodyClasses += " wp-theme-" + stylesheet
	}
	bodyClasses = wp.BodyClassWithAdminBar(c, bodyClasses)
	safe = ensureMainSkipTarget(safe)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en-US\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n")
	b.WriteString("<meta name='robots' content='max-image-preview:large' />\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "")
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
		ClassList:        wp.GetPostClass(c, post.ID),
	})
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
	b.WriteString("<meta name='robots' content='max-image-preview:large' />\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString(" &#8211; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n")
	perm := singularPermalink(home, post)
	restPath := "/wp/v2/posts/" + strconv.FormatInt(post.ID, 10)
	if post.Type == "page" {
		restPath = "/wp/v2/pages/" + strconv.FormatInt(post.ID, 10)
	}
	writeFrontHeadLinks(&b, home, blogname, perm, restPath, post.Title)
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

	setFrontSingularLinks(w, c, post)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderSingularShim(w http.ResponseWriter, c *wp.Context, post *singularPost) {
	blogname := optionString(c, "blogname", "GoPress")
	content := post.Content
	if post.Type == "attachment" && content == "" {
		if url := wp.GetAttachmentURL(c, post.ID); url != "" {
			content = `<p class="attachment"><a href="` + html.EscapeString(url) + `">` +
				html.EscapeString(post.Title) + `</a></p>`
		}
	}
	// the_content: blocks → shortcodes → texturize (P5.11).
	content = wp.TheContent(c, content)
	safeContent := wp.ProcessInteractivityHTML(wp.KsesPost(content))

	bodyClass := "singular"
	if post.Type == "attachment" {
		bodyClass = "singular attachment"
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
	if post.FeaturedImageURL != "" {
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
	if post.Type == "post" || post.Type == "page" {
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

func (f *Front) renderFeedRSS2(w http.ResponseWriter, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	desc := optionString(c, "blogdescription", "")
	posts, err := latestFeedPosts(c, 10)
	if err != nil {
		http.Error(w, "feed: "+err.Error()+"\n", http.StatusInternalServerError)
		return
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
	b.WriteString("\t>\n\n")
	b.WriteString("<channel>\n")
	b.WriteString("\t<title>")
	b.WriteString(xmlEscape(blogname))
	b.WriteString("</title>\n")
	b.WriteString("\t<atom:link href=\"")
	b.WriteString(xmlEscape(home + "/feed/"))
	b.WriteString("\" rel=\"self\" type=\"application/rss+xml\" />\n")
	b.WriteString("\t<link>")
	b.WriteString(xmlEscape(home))
	b.WriteString("</link>\n")
	b.WriteString("\t<description>")
	b.WriteString(xmlEscape(desc))
	b.WriteString("</description>\n")
	b.WriteString("\t<lastBuildDate>")
	b.WriteString(xmlEscape(feedBuildDate(posts)))
	b.WriteString("</lastBuildDate>\n")
	b.WriteString("\t<language>en-US</language>\n")
	b.WriteString("\t<sy:updatePeriod>\n")
	b.WriteString("\thourly\t</sy:updatePeriod>\n")
	b.WriteString("\t<sy:updateFrequency>\n")
	b.WriteString("\t1\t</sy:updateFrequency>\n")
	b.WriteString("\t<generator>https://wordpress.org/?v=7.2-alpha-64260</generator>\n")
	for i, p := range posts {
		link := postPermalink(home, p.Post)
		guid := home + "/?p=" + strconv.FormatInt(p.ID, 10)
		if p.GUID != "" {
			guid = p.GUID
		}
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
			anchor := "#respond"
			if p.CommentCount > 0 {
				anchor = "#comments"
			}
			b.WriteString("\t\t\t\t\t<comments>")
			b.WriteString(xmlEscape(strings.TrimRight(link, "/") + "/" + anchor))
			b.WriteString("</comments>\n")
			b.WriteString("\t\t\n")
		}
		b.WriteString("\t\t<dc:creator><![CDATA[")
		b.WriteString(p.AuthorName)
		b.WriteString("]]></dc:creator>\n")
		b.WriteString("\t\t<pubDate>")
		b.WriteString(xmlEscape(feedRFC822(p.DateGMT, p.Date)))
		b.WriteString("</pubDate>\n")
		for _, cat := range p.Categories {
			b.WriteString("\t\t\t\t<category><![CDATA[")
			b.WriteString(cat)
			b.WriteString("]]></category>\n")
		}
		b.WriteString("\t\t<guid isPermaLink=\"false\">")
		b.WriteString(xmlEscape(guid))
		b.WriteString("</guid>\n\n")
		b.WriteString("\t\t\t\t\t<description><![CDATA[")
		b.WriteString(p.ExcerptRSS)
		b.WriteString("]]></description>\n")
		b.WriteString("\t\t\t\t\t\t\t\t\t\t")
		if p.ContentRSS != "" {
			b.WriteString("<content:encoded><![CDATA[")
			if !strings.HasPrefix(p.ContentRSS, "\n") {
				b.WriteString("\n")
			}
			b.WriteString(p.ContentRSS)
			if !strings.HasSuffix(p.ContentRSS, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("]]></content:encoded>\n")
		} else {
			b.WriteString("<content:encoded><![CDATA[\n]]></content:encoded>\n")
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
	posts, err := latestFeedPosts(c, 10)
	if err != nil {
		http.Error(w, "feed: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}

	self := feedSelfLink(r, c)
	atomID := home + "/feed/atom/"

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><feed` + "\n")
	b.WriteString("\txmlns=\"http://www.w3.org/2005/Atom\"\n")
	b.WriteString("\txml:lang=\"en-US\"\n")
	b.WriteString("\txmlns:thr=\"http://purl.org/syndication/thread/1.0\"\n")
	b.WriteString("\t>\n")
	b.WriteString("\t<title type=\"text\">")
	b.WriteString(xmlEscape(blogname))
	b.WriteString("</title>\n")
	b.WriteString("\t<subtitle type=\"text\">")
	b.WriteString(xmlEscape(desc))
	b.WriteString("</subtitle>\n\n")
	b.WriteString("\t<updated>")
	if len(posts) > 0 {
		b.WriteString(xmlEscape(feedAtomDate(posts[0].ModifiedGMT, posts[0].Modified)))
	} else {
		b.WriteString(xmlEscape(feedAtomDate("", "")))
	}
	b.WriteString("</updated>\n\n")
	b.WriteString("\t<link rel=\"alternate\" type=\"text/html\" href=\"")
	b.WriteString(xmlEscape(home))
	b.WriteString("\" />\n")
	b.WriteString("\t<id>")
	b.WriteString(xmlEscape(atomID))
	b.WriteString("</id>\n")
	b.WriteString("\t<link rel=\"self\" type=\"application/atom+xml\" href=\"")
	b.WriteString(xmlEscape(self))
	b.WriteString("\" />\n\n")
	b.WriteString("\t<generator uri=\"https://wordpress.org/\" version=\"7.2-alpha-64260\">WordPress</generator>\n")

	for _, p := range posts {
		link := postPermalink(home, p.Post)
		guid := home + "/?p=" + strconv.FormatInt(p.ID, 10)
		if p.GUID != "" {
			guid = p.GUID
		}
		b.WriteString("\t<entry>\n")
		b.WriteString("\t\t<author>\n")
		b.WriteString("\t\t\t<name>")
		b.WriteString(xmlEscape(p.AuthorName))
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
		b.WriteString(xmlEscape(feedAtomDate(p.ModifiedGMT, p.Modified)))
		b.WriteString("</updated>\n")
		b.WriteString("\t\t<published>")
		b.WriteString(xmlEscape(feedAtomDate(p.DateGMT, p.Date)))
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
			commentsAnchor := "#respond"
			if p.CommentCount > 0 {
				commentsAnchor = "#comments"
			}
			b.WriteString("\t\t\t\t\t<link rel=\"replies\" type=\"text/html\" href=\"")
			b.WriteString(xmlEscape(strings.TrimRight(link, "/") + "/" + commentsAnchor))
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
		b.WriteString("\t\t\t</entry>\n")
	}
	b.WriteString("</feed>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "application/atom+xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

type feedPost struct {
	wp.Post
	AuthorName   string
	AuthorURL    string
	Categories   []string
	CommentCount int64
	ExcerptRSS   string
	ContentRSS   string
}

func latestFeedPosts(c *wp.Context, limit int) ([]feedPost, error) {
	if limit <= 0 {
		limit = 10
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostType: "post", PostStatus: "publish", PostsPerPage: limit})
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
		out = append(out, fp)
	}
	return out, nil
}

func feedBuildDate(posts []feedPost) string {
	if len(posts) == 0 {
		return time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 +0000")
	}
	return feedRFC822(posts[0].DateGMT, posts[0].Date)
}

func feedRFC822(gmt, local string) string {
	for _, src := range []string{gmt, local} {
		if src == "" {
			continue
		}
		if t, err := time.Parse("2006-01-02 15:04:05", src); err == nil {
			return t.UTC().Format("Mon, 02 Jan 2006 15:04:05 +0000")
		}
	}
	return time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 +0000")
}

func feedAuthor(c *wp.Context, authorID int64) (name, url string) {
	if c == nil || authorID <= 0 {
		return "", ""
	}
	u, err := c.GetUserBy("id", authorID)
	if err != nil || u == nil {
		return "", ""
	}
	if u.DisplayName != "" {
		name = u.DisplayName
	} else {
		name = u.Login
	}
	return name, u.URL
}

func feedAtomDate(gmt, local string) string {
	for _, src := range []string{gmt, local} {
		if src == "" {
			continue
		}
		if t, err := time.Parse("2006-01-02 15:04:05", src); err == nil {
			return t.UTC().Format("2006-01-02T15:04:05Z")
		}
	}
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func feedSelfLink(r *http.Request, c *wp.Context) string {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	if r == nil {
		return home + "/feed/atom/"
	}
	// Prefer absolute URL matching the request (WP self_link).
	if r.URL != nil {
		path := r.URL.RequestURI()
		if path == "" {
			path = "/"
		}
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			return path
		}
		return home + path
	}
	return home + "/feed/atom/"
}

func feedCategoryNames(c *wp.Context, postID int64) []string {
	out := []string{}
	if c == nil || c.DB == nil || postID <= 0 {
		return out
	}
	q, args, err := c.DB.Prepare(
		"SELECT t.name FROM "+c.DB.Terms()+" t "+
			"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
			"INNER JOIN "+c.DB.TermRelationships()+" tr ON tr.term_taxonomy_id = tt.term_taxonomy_id "+
			"WHERE tr.object_id = %d AND tt.taxonomy = %s ORDER BY t.name ASC",
		postID, "category",
	)
	if err != nil {
		return out
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return out
	}
	for _, row := range rows {
		name := ""
		switch v := row["name"].(type) {
		case string:
			name = v
		case []byte:
			name = string(v)
		}
		if name != "" {
			out = append(out, name)
		}
	}
	return out
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
	http.Redirect(w, r, target, http.StatusMovedPermanently)
	return true
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
	if parsed == nil || parsed.IsHome {
		return false
	}
	if isSingularVars(parsed.QueryVars) || isFeedVars(parsed.QueryVars) {
		return true
	}
	for _, k := range []string{"category_name", "tag", "year", "monthnum", "author_name"} {
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
	http.Redirect(w, r, target, http.StatusMovedPermanently)
	return true
}

func (f *Front) renderTermArchive(w http.ResponseWriter, c *wp.Context, taxonomy, slug string) {
	blogname := optionString(c, "blogname", "GoPress")
	posts, err := latestPosts(c, 10)
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(slug))
	b.WriteString(" &#8212; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "archive")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(taxonomy))
	b.WriteString(": ")
	b.WriteString(html.EscapeString(slug))
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

func (f *Front) render404(w http.ResponseWriter, c *wp.Context) {
	// P3.16: prefer block theme templates/404.html when present.
	if tpl, ok := wp.ResolveBlockTemplate(c, "404"); ok && strings.TrimSpace(tpl) != "" {
		f.render404BlockTemplate(w, c, tpl)
		return
	}
	f.render404Shim(w, c)
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
	b.WriteString("<meta name='robots' content='max-image-preview:large' />\n")
	b.WriteString("<title>Page not found &#8211; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n")
	writeFrontHeadLinks(&b, home, blogname, "", "", "")
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

// writeFrontHeadLinks emits wp_head discovery links (feeds, api.w.org, oEmbed, canonical).
// Stylesheet <link> tags are omitted: the HTML normalizer strips them, and TT5 enqueues
// styles as inline <style> which we also normalize away (P5.19).
func writeFrontHeadLinks(b *strings.Builder, home, blogname, permalink, restPath, title string) {
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
	b.WriteString("<meta name=\"generator\" content=\"WordPress 7.2-alpha-64260\" />\n")
	if permalink != "" {
		b.WriteString("<link rel=\"canonical\" href=\"")
		b.WriteString(html.EscapeString(permalink))
		b.WriteString("\" />\n")
		// shortlink uses ?p=ID when restPath ends with /ID
		if i := strings.LastIndex(restPath, "/"); i >= 0 && i+1 < len(restPath) {
			b.WriteString("<link rel='shortlink' href='")
			b.WriteString(html.EscapeString(home + "/?p=" + restPath[i+1:]))
			b.WriteString("' />\n")
		}
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
func setFrontSingularLinks(w http.ResponseWriter, c *wp.Context, post *singularPost) {
	siteURL := strings.TrimRight(optionString(c, "siteurl", ""), "/")
	home := strings.TrimRight(optionString(c, "home", siteURL), "/")
	if home == "" {
		home = siteURL
	}
	if home == "" || post == nil {
		return
	}
	w.Header().Add("Link", fmt.Sprintf(`<%s/wp-json/>; rel="https://api.w.org/"`, home))
	restType := "posts"
	if post.Type == "page" {
		restType = "pages"
	}
	if post.Type == "post" || post.Type == "page" {
		w.Header().Add("Link", fmt.Sprintf(
			`<%s/wp-json/wp/v2/%s/%d>; rel="alternate"; title="JSON"; type="application/json"`,
			home, restType, post.ID))
		w.Header().Add("Link", fmt.Sprintf(`<%s/?p=%d>; rel=shortlink`, home, post.ID))
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
	return &blocks.Env{
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
			// Inherit tag archive filters (P3.24).
			if q.Inherit && c != nil && c.Scope != nil && c.Scope.IsTag {
				tagSlug = c.Scope.TagSlug
			}
			posts, err := wp.GetPosts(c, wp.QueryArgs{
				PostType:     postType,
				PostStatus:   "publish",
				PostsPerPage: limit,
				Offset:       q.Offset,
				Search:       search,
				Year:         year,
				MonthNum:     month,
				Day:          day,
				Author:       authorID,
				Tag:          tagSlug,
				// Sticky posts are prepended for front query loops (inherit or not).
				StickyFirst: postType == "post" && search == "" && year == 0 && authorID == 0 && tagSlug == "",
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
		HomeURL: func() string {
			return strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		},
		IsFrontPage: func() bool {
			// Front block templates set this via env flag when rendering home.
			return false
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
			p, err := wp.GetPostByID(c, postID)
			if err != nil || p == nil || c == nil || c.DB == nil {
				return ""
			}
			q, args, err := c.DB.Prepare(
				"SELECT display_name FROM "+c.DB.Users()+" WHERE ID = %d LIMIT 1",
				p.Author,
			)
			if err != nil {
				return ""
			}
			row, err := c.DB.GetRow(q, args...)
			if err != nil || row == nil {
				return ""
			}
			return rowStringDB(row["display_name"])
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
	}
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
