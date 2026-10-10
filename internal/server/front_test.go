package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func testBoot(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH: "/workspace/",
		AuthKey: "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	return c
}

func TestFrontHomeRendersBlogname(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<title>") || !strings.Contains(body, "GoPress Fixture") {
		t.Fatalf("expected blogname in HTML, got %q", body)
	}
	// P5.16: TT5 home document chrome (body classes + site-blocks wrapper).
	if !strings.Contains(body, `wp-theme-twentytwentyfive`) || !strings.Contains(body, `home blog`) {
		t.Fatalf("expected TT5 home body classes, got %q", body)
	}
	if !strings.Contains(body, `class="wp-site-blocks"`) {
		t.Fatalf("expected wp-site-blocks wrapper, got %q", body)
	}
	if !strings.Contains(body, "wp-block-group") {
		t.Fatalf("expected rendered block template markup, got %q", body)
	}
	// P5.6: home.html includes <!-- wp:template-part {"slug":"header"} /-->;
	// fixture customizes that part in the DB.
	if !strings.Contains(body, "Customized header part") {
		t.Fatalf("expected rendered template part, got %q", body)
	}
	if !strings.Contains(body, `<header class="wp-block-template-part"`) {
		t.Fatalf("expected header template-part wrapper from slug default, got %q", body)
	}
	if !strings.Contains(body, ">Blog</h1>") {
		t.Fatalf("expected hidden-blog-heading pattern text Blog, got %q", body)
	}
	if strings.Contains(body, "<!-- wp:") {
		t.Fatalf("block comments should be rendered away, got %q", body)
	}
	if strings.Contains(body, `<ul class="posts">`) {
		t.Fatalf("shim home should not render when a block template exists")
	}
	// P5.13: home pattern expands core/query + post-template with published posts.
	if !strings.Contains(body, `wp-block-post-template`) {
		t.Fatalf("expected query post-template on home, got %q", body)
	}
	if !strings.Contains(body, "Hello world") && !strings.Contains(body, "Hello World") {
		t.Fatalf("expected a post title from query loop, got %q", body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type=%q", ct)
	}
}

func TestFrontSingularByQueryP(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Hello world!") {
		t.Fatalf("expected title, got %q", body)
	}
	if !strings.Contains(body, "Welcome to WordPress") {
		t.Fatalf("expected content, got %q", body)
	}
	if !strings.Contains(body, "entry-content") && !strings.Contains(body, "wp-site-blocks") {
		t.Fatalf("expected singular content wrapper, got %q", body)
	}
	if strings.Contains(body, "<!-- wp:") {
		t.Fatalf("block comments should be rendered away, got %q", body)
	}
}

func TestFrontSingularOmitsXPingback(t *testing.T) {
	// P3.41: XML-RPC is gone, so singular responses must not advertise X-Pingback.
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if got := rec.Header().Get("X-Pingback"); got != "" {
		t.Fatalf("X-Pingback must not be sent, got %q", got)
	}
	for _, v := range rec.Header().Values("X-Pingback") {
		if strings.TrimSpace(v) != "" {
			t.Fatalf("X-Pingback values=%v", rec.Header().Values("X-Pingback"))
		}
	}
}

func TestFrontSingularOEmbedDiscoveryLinks(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `type="application/json+oembed"`) {
		t.Fatalf("missing oEmbed JSON discovery link: %q", body)
	}
	if !strings.Contains(body, `type="text/xml+oembed"`) {
		t.Fatalf("missing oEmbed XML discovery link: %q", body)
	}
	if !strings.Contains(body, "/wp-json/oembed/1.0/embed?url=") {
		t.Fatalf("missing oEmbed endpoint href: %q", body)
	}
}

func TestFrontSingularByPageID(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?page_id=2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Sample Page") {
		t.Fatalf("expected Sample Page, got %q", rec.Body.String())
	}
}

func TestFrontSingularByPrettyPermalink(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/2024/06/15/hello-world/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Hello world!") {
		t.Fatalf("expected Hello world!, got %q", rec.Body.String())
	}
}

func TestFrontSingular404(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=999999999", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Page not found") {
		t.Fatalf("expected 404 page, got %q", body)
	}
	// P3.16: active block theme should render templates/404.html.
	if !strings.Contains(body, "error404") {
		t.Fatalf("expected error404 body class: %q", body)
	}
	if !strings.Contains(body, "wp-site-blocks") {
		t.Fatalf("expected block theme 404 wrapper: %q", body)
	}
	// P3.35: 404 must include header and footer template parts.
	if !strings.Contains(body, `<header class="wp-block-template-part"`) {
		t.Fatalf("expected header template-part on 404: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, `<footer class="wp-block-template-part"`) {
		t.Fatalf("expected footer template-part on 404: %q", body[:min(600, len(body))])
	}
}

func TestEnsure404TemplateParts(t *testing.T) {
	bare := `<!-- wp:paragraph --><p>oops</p><!-- /wp:paragraph -->`
	got := server.Ensure404TemplateParts(bare)
	if !strings.Contains(got, `"slug":"header"`) || !strings.Contains(got, `"slug":"footer"`) {
		t.Fatalf("expected header/footer injection: %q", got)
	}
	if !strings.Contains(got, "oops") {
		t.Fatalf("expected original content preserved: %q", got)
	}
	full := `<!-- wp:template-part {"slug":"header"} /--><!-- wp:paragraph --><p>x</p><!-- /wp:paragraph --><!-- wp:template-part {"slug":"footer"} /-->`
	if server.Ensure404TemplateParts(full) != full {
		t.Fatalf("should leave complete template alone")
	}
}

func TestFrontAttachmentByQueryID(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?attachment_id=5", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "sample-photo-a") {
		t.Fatalf("expected attachment title/url, got %q", body)
	}
	if !strings.Contains(body, "attachment") || !strings.Contains(body, "type-attachment") {
		t.Fatalf("expected attachment body/class, got %q", body)
	}
}

func TestFrontAttachmentByRewrite(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	// Fixture rewrite_rules include ".?.+?/attachment/([^/]+)/?$" → attachment=$matches[1].
	req := httptest.NewRequest(http.MethodGet, "/files/attachment/sample-photo-a/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "sample-photo-a") {
		t.Fatalf("expected attachment, got %q", rec.Body.String())
	}
}

func TestFrontEmbedByQuery(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1&embed=true", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-WP-embed") != "true" {
		t.Fatalf("missing X-WP-embed header")
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="wp-embed"`) || !strings.Contains(body, "Hello world") {
		t.Fatalf("expected embed markup, got %q", body)
	}
}

func TestFrontEmbedPrettyPath(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/2024/06/15/hello-world/embed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-WP-embed") != "true" {
		t.Fatalf("missing X-WP-embed header")
	}
	if !strings.Contains(rec.Body.String(), "wp-embed-heading") {
		t.Fatalf("expected embed heading, got %q", rec.Body.String())
	}
}

func TestFrontFeedQueryRSS2(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "rss") && !strings.Contains(ct, "xml") {
		t.Fatalf("content-type=%q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<rss version="2.0"`) || !strings.Contains(body, "<channel>") {
		t.Fatalf("expected RSS2 channel, got %q", body)
	}
	if !strings.Contains(body, "xmlns:content=") || !strings.Contains(body, "<atom:link") {
		t.Fatalf("expected WP feed namespaces/atom:link, got %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "<item>") || !strings.Contains(body, "<dc:creator>") {
		t.Fatalf("expected feed items with dc:creator, got %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, "<content:encoded>") {
		t.Fatalf("expected content:encoded, got %q", body[:min(800, len(body))])
	}
}

func TestFrontFeedPrettyPath(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<rss version="2.0"`) || !strings.Contains(body, "xmlns:atom=") {
		t.Fatalf("expected RSS2 with namespaces, got %q", body[:min(400, len(body))])
	}
}

func TestFrontFeedAtom(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=atom", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "atom") {
		t.Fatalf("content-type=%q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<feed`) || !strings.Contains(body, `xmlns="http://www.w3.org/2005/Atom"`) {
		t.Fatalf("expected Atom feed, got %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "<entry>") || !strings.Contains(body, "<author>") {
		t.Fatalf("expected entries with author, got %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, `type="application/atom+xml"`) {
		t.Fatalf("expected self atom link, got %q", body[:min(600, len(body))])
	}
}

func TestFrontFeedAtomPrettyPath(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/atom/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `xmlns="http://www.w3.org/2005/Atom"`) || !strings.Contains(body, "<entry>") {
		t.Fatalf("expected Atom feed via pretty path, got %q", body[:min(400, len(body))])
	}
}

func TestFrontFeedRDF(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rdf", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "rdf") && !strings.Contains(ct, "xml") {
		t.Fatalf("content-type=%q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<rdf:RDF") || !strings.Contains(body, `xmlns="http://purl.org/rss/1.0/"`) {
		t.Fatalf("expected RDF feed, got %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "<rdf:Seq>") || !strings.Contains(body, "<item rdf:about=") {
		t.Fatalf("expected rdf:Seq and items, got %q", body[:min(800, len(body))])
	}
}

func TestFrontFeedRDFPrettyPath(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/rdf/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<rdf:RDF") || !strings.Contains(body, "<channel rdf:about=") {
		t.Fatalf("expected RDF via pretty path, got %q", body[:min(400, len(body))])
	}
}

func TestFrontCommentsFeed(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/comments/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<rss version="2.0"`) {
		t.Fatalf("expected RSS2: %q", body[:min(300, len(body))])
	}
	if !strings.Contains(body, "Comments for ") {
		t.Fatalf("expected comments feed title: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "/comments/feed/") {
		t.Fatalf("expected self comments/feed link: %q", body[:min(500, len(body))])
	}
	if !strings.Contains(body, "Comment on ") || !strings.Contains(body, "#comment-") {
		t.Fatalf("expected comment items: %q", body[:min(800, len(body))])
	}
	// Must not be the posts feed.
	if strings.Contains(body, "xmlns:wfw=") || strings.Contains(body, "xmlns:slash=") {
		t.Fatalf("comments feed should not include wfw/slash namespaces")
	}
	if strings.Contains(body, "<slash:comments>") {
		t.Fatalf("comments feed should not list post slash:comments")
	}
}

func TestFrontCommentsFeedAtom(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/comments/feed/atom/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "atom") {
		t.Fatalf("Content-Type=%q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<feed`) || !strings.Contains(body, `xmlns="http://www.w3.org/2005/Atom"`) {
		t.Fatalf("expected Atom feed: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "Comments for ") {
		t.Fatalf("expected comments title: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "/comments/feed/atom/") {
		t.Fatalf("expected self atom link: %q", body[:min(500, len(body))])
	}
	if !strings.Contains(body, "<entry>") || !strings.Contains(body, "thr:in-reply-to") {
		t.Fatalf("expected comment entries: %q", body[:min(800, len(body))])
	}
	if strings.Contains(body, `<rss`) {
		t.Fatalf("atom comments feed must not be RSS")
	}
}

func TestFrontTrailingSlashCanonical(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/sample-page", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasSuffix(loc, "/sample-page/") {
		t.Fatalf("Location=%q", loc)
	}
	if !strings.HasPrefix(loc, "http://") && !strings.HasPrefix(loc, "https://") {
		t.Fatalf("expected absolute Location, got %q", loc)
	}
}

func TestFrontTrailingSlashCanonicalPreservesQuery(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/sample-page?utm=1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status=%d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasSuffix(loc, "/sample-page/?utm=1") {
		t.Fatalf("Location=%q", loc)
	}
}

func TestFrontTrailingSlashHomePaged(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/page/2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasSuffix(loc, "/page/2/") {
		t.Fatalf("Location=%q", loc)
	}
}

func TestFrontSearchResults(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?s=Hello", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "search") {
		t.Fatalf("expected search body class, got %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "Search results") && !strings.Contains(body, "Search Results") {
		t.Fatalf("expected search heading, got %q", body[:min(600, len(body))])
	}
	// P3.36: theme search.html + curly quotes + search form from pattern.
	if !strings.Contains(body, "\u201CHello\u201D") && !strings.Contains(body, "&#8220;Hello&#8221;") {
		t.Fatalf("expected curly-quoted search term in heading: %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, `role="search"`) || !strings.Contains(body, "wp-block-search") {
		t.Fatalf("expected core/search form from search.html: %q", body[:min(1200, len(body))])
	}
	if !strings.Contains(body, "Hello world") && !strings.Contains(body, "Hello World") {
		t.Fatalf("expected matching post title, got %q", body)
	}
	if strings.Contains(body, "GoPress: front route not implemented") {
		t.Fatal("search should not 501")
	}
}

func TestEnsureSearchTemplateParts(t *testing.T) {
	got := server.EnsureSearchTemplateParts(`<!-- wp:query-title {"type":"search"} /-->`)
	if !strings.Contains(got, `"slug":"header"`) || !strings.Contains(got, `"slug":"footer"`) {
		t.Fatalf("expected header/footer injection: %q", got)
	}
	full := `<!-- wp:template-part {"slug":"header"} /-->` + "\n" +
		`<!-- wp:query-title {"type":"search"} /-->` + "\n" +
		`<!-- wp:template-part {"slug":"footer"} /-->`
	if server.EnsureSearchTemplateParts(full) != strings.TrimSpace(full) {
		t.Fatal("should leave complete template alone")
	}
}

func TestFrontAuthorArchive(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	req := httptest.NewRequest(http.MethodGet, "/author/admin/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "archive") || !strings.Contains(body, "author") {
		t.Fatalf("expected archive author body class: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "Author:") {
		t.Fatalf("expected Author heading: %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, "Hello world") && !strings.Contains(body, "Hello World") {
		t.Fatalf("expected post in author archive: %q", body)
	}
	if !strings.Contains(body, "Posts by") || !strings.Contains(body, "/author/admin/feed/") {
		t.Fatalf("expected author feed_links_extra alternate: %q", body[:min(1200, len(body))])
	}
	if !strings.Contains(body, `/wp-json/wp/v2/users/1`) {
		t.Fatalf("expected author JSON alternate: %q", body[:min(1200, len(body))])
	}
	if strings.Contains(body, "GoPress: front route not implemented") {
		t.Fatal("author archive should not 501")
	}

	req = httptest.NewRequest(http.MethodGet, "/author/nobody-here/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing author want 404, got %d", rec.Code)
	}
}

func TestFrontTagArchive(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	req := httptest.NewRequest(http.MethodGet, "/tag/fixture-tag-01/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "archive") || !strings.Contains(body, "tag") {
		t.Fatalf("expected archive tag body class: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "Tag:") {
		t.Fatalf("expected Tag heading: %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, "Fixture Post") {
		t.Fatalf("expected tagged post: %q", body)
	}
	if strings.Contains(body, "GoPress: front route not implemented") {
		t.Fatal("tag archive should not 501")
	}

	req = httptest.NewRequest(http.MethodGet, "/tag/nobody-here-tag/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing tag want 404, got %d", rec.Code)
	}
}

func TestFrontCategoryArchive(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	req := httptest.NewRequest(http.MethodGet, "/category/uncategorized/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "archive") || !strings.Contains(body, "category") {
		t.Fatalf("expected archive category body class: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "Category:") {
		t.Fatalf("expected Category heading: %q", body[:min(800, len(body))])
	}
	if strings.Contains(body, "GoPress: front route not implemented") {
		t.Fatal("category archive should not 501")
	}

	req = httptest.NewRequest(http.MethodGet, "/category/nobody-here-cat/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing category want 404, got %d", rec.Code)
	}
}

func TestFrontDateArchiveYearMonth(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	req := httptest.NewRequest(http.MethodGet, "/2024/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("year status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "archive") || !strings.Contains(body, "date") {
		t.Fatalf("expected archive date body class: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "Year:") {
		t.Fatalf("expected Year heading: %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, "Hello world") && !strings.Contains(body, "Hello World") {
		t.Fatalf("expected post in year archive: %q", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/2024/06/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("month status=%d body=%q", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Month:") || !strings.Contains(body, "June") {
		t.Fatalf("expected Month: June heading: %q", body[:min(800, len(body))])
	}

	req = httptest.NewRequest(http.MethodGet, "/2019/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("empty year want 404, got %d body=%q", rec.Code, rec.Body.String()[:min(200, rec.Body.Len())])
	}
}

func TestFrontRobotsNoindexWhenNotPublic(t *testing.T) {
	c := testBoot(t)
	orig := c.Options.GetOption("blog_public", "1")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("blog_public", orig, "yes") })
	if _, err := c.Options.UpdateOption("blog_public", "0", "yes"); err != nil {
		t.Fatalf("UpdateOption: %v", err)
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `content='noindex, nofollow'`) {
		t.Fatalf("expected noindex meta, got %q", body)
	}
	if strings.Contains(body, `max-image-preview:large`) {
		t.Fatalf("unexpected max-image-preview when private: %q", body)
	}
}
