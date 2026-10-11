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

func TestFrontHeadFeedLinksShowFilters(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("feed_links_show_posts_feed", func(value any, _ ...any) any {
		return false
	}, 10, 1)
	c.Hooks.AddFilter("feed_links_show_comments_feed", func(value any, _ ...any) any {
		return false
	}, 10, 1)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, " &raquo; Feed\"") {
		t.Fatalf("posts feed should be hidden: %q", body[:min(800, len(body))])
	}
	if strings.Contains(body, " &raquo; Comments Feed\"") {
		t.Fatalf("comments feed should be hidden: %q", body[:min(800, len(body))])
	}
	// Discovery links still present.
	if !strings.Contains(body, `rel="https://api.w.org/"`) {
		t.Fatalf("expected api.w.org link: %q", body[:min(800, len(body))])
	}
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

func TestFrontSingularOmitsTrackbackRDF(t *testing.T) {
	// P3.46: trackback_rdf() / xmlns:trackback must not appear on singular HTML.
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	for _, path := range []string{"/?p=1", "/sample-page/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d", path, rec.Code)
		}
		body := rec.Body.String()
		for _, bad := range []string{
			`xmlns:trackback=`,
			`trackback:ping=`,
			`<rdf:RDF xmlns:rdf=`,
		} {
			if strings.Contains(body, bad) {
				t.Fatalf("%s must omit trackback RDF marker %q", path, bad)
			}
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

func TestFront404SetsScopeAndSearch(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=999999999", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
	// Scope is request-local via ForRequest; re-check body markers instead.
	body := rec.Body.String()
	if !strings.Contains(body, "wp-block-search") {
		t.Fatalf("expected search on 404: %q", body[:min(500, len(body))])
	}
	if !strings.Contains(body, "Page not found &#8211;") {
		t.Fatalf("expected document title en-dash: %q", body[:min(400, len(body))])
	}
}

func TestFront404DocumentTitlePartsFilter(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("document_title_parts", func(value any, _ ...any) any {
		m, _ := value.(map[string]string)
		if m == nil {
			m = map[string]string{}
		}
		m["title"] = "Missing page"
		return m
	}, 10, 1)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=999999998", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "<title>Missing page &#8211;") {
		t.Fatalf("expected filtered title: %q", rec.Body.String()[:min(300, rec.Body.Len())])
	}
}

func TestFrontAttachmentByQueryID(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?attachment_id=5", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	// P3.109: fixture has wp_attachment_pages_enabled=0 → 301 to file URL.
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "sample-photo-a") || !strings.Contains(loc, "/wp-content/uploads/") {
		t.Fatalf("Location=%q", loc)
	}
	if rec.Header().Get("X-Redirect-By") != "WordPress" {
		t.Fatalf("X-Redirect-By=%q", rec.Header().Get("X-Redirect-By"))
	}
}

func TestFrontAttachmentByRewrite(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	// Fixture rewrite_rules include ".?.+?/attachment/([^/]+)/?$" → attachment=$matches[1].
	req := httptest.NewRequest(http.MethodGet, "/files/attachment/sample-photo-a/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "sample-photo-a.jpg") {
		t.Fatalf("Location=%q", loc)
	}
}

func TestFrontAttachmentPageWhenEnabled(t *testing.T) {
	c := testBoot(t)
	if c.Options != nil {
		if _, err := c.Options.UpdateOption("wp_attachment_pages_enabled", "1", "yes"); err != nil {
			t.Fatalf("enable attachment pages: %v", err)
		}
		t.Cleanup(func() { _, _ = c.Options.UpdateOption("wp_attachment_pages_enabled", "0", "yes") })
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?attachment_id=5", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="attachment"`) || !strings.Contains(body, "type-attachment") {
		t.Fatalf("expected attachment page, got %q", body)
	}
	if !strings.Contains(body, "<img ") && !strings.Contains(body, "sample-photo-a") {
		t.Fatalf("expected medium image or title link, got %q", body)
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
	// P3.50: atom:link rel=self mirrors the request URI (WP get_self_link).
	if !strings.Contains(body, `rel="self"`) || !strings.Contains(body, "/?feed=rss2") {
		t.Fatalf("expected self atom:link with request URI, got %q", body[:min(500, len(body))])
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
	if !strings.Contains(body, `rel="self"`) || !strings.Contains(body, "/feed/") {
		t.Fatalf("expected self atom:link for /feed/, got %q", body[:min(500, len(body))])
	}
}

func TestFrontFeedLastBuildDateStable(t *testing.T) {
	// P3.55: lastBuildDate from newest post — stable across requests (no wall clock).
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	extract := func() string {
		req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
		body := rec.Body.String()
		const open, close = "<lastBuildDate>", "</lastBuildDate>"
		i := strings.Index(body, open)
		j := strings.Index(body, close)
		if i < 0 || j < i {
			t.Fatalf("missing lastBuildDate in %q", body[:min(400, len(body))])
		}
		return body[i+len(open) : j]
	}
	a, b := extract(), extract()
	if a == "" {
		t.Fatal("empty lastBuildDate")
	}
	if a != b {
		t.Fatalf("unstable lastBuildDate: %q vs %q", a, b)
	}
	if !strings.Contains(a, "2024") {
		t.Fatalf("expected fixture post year in lastBuildDate, got %q", a)
	}
}

func TestFrontFeedTTLOptional(t *testing.T) {
	// P3.57: <ttl> only when rss_ttl is a positive integer.
	c := testBoot(t)
	_, _ = c.Options.DeleteOption("rss_ttl")
	t.Cleanup(func() { _, _ = c.Options.DeleteOption("rss_ttl") })
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "<ttl>") {
		t.Fatalf("unexpected ttl without option: %q", rec.Body.String()[:min(500, rec.Body.Len())])
	}

	if _, err := c.Options.UpdateOption("rss_ttl", "90", "yes"); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<ttl>90</ttl>") {
		t.Fatalf("missing ttl: %q", rec.Body.String()[:min(600, rec.Body.Len())])
	}
}

func TestFrontFeedRSS2SiteIconImage(t *testing.T) {
	// P3.52: channel <image> when site_icon attachment is set (rss2_site_icon).
	c := testBoot(t)
	orig := c.Options.GetOption("site_icon", "0")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("site_icon", orig, "yes") })
	if _, err := c.Options.UpdateOption("site_icon", "5", "yes"); err != nil {
		t.Fatalf("UpdateOption: %v", err)
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<image>",
		"<url>",
		"sample-photo-a.jpg",
		"<width>32</width>",
		"<height>32</height>",
		"</image>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in feed: %q", want, body[:min(800, len(body))])
		}
	}
}

func TestFrontFeedRSS2OmitsImageWithoutSiteIcon(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if strings.Contains(body, "<image>") {
		t.Fatalf("unexpected channel image without site_icon: %q", body[:min(600, len(body))])
	}
}

func TestFrontFeedOmitsPubSubHubbub(t *testing.T) {
	// P3.51: feeds must not advertise PubSubHubbub hub endpoints.
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	for _, path := range []string{"/?feed=rss2", "/feed/", "/?feed=atom", "/feed/atom/", "/comments/feed/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d", path, rec.Code)
		}
		body := strings.ToLower(rec.Body.String())
		if strings.Contains(body, `rel="hub"`) || strings.Contains(body, "rel='hub'") {
			t.Fatalf("%s must not emit rel=hub: %q", path, rec.Body.String()[:min(500, rec.Body.Len())])
		}
		if strings.Contains(body, "pubsubhubbub") || strings.Contains(body, "pub.sub") {
			t.Fatalf("%s must not mention pubsubhubbub: %q", path, rec.Body.String()[:min(500, rec.Body.Len())])
		}
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
	// P3.58: Atom rel=self mirrors request URI (/?feed=atom).
	if !strings.Contains(body, `rel="self"`) || !strings.Contains(body, "/?feed=atom") {
		t.Fatalf("expected self link with request URI, got %q", body[:min(600, len(body))])
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
	if !strings.Contains(body, `rel="self"`) || !strings.Contains(body, "/feed/atom/") {
		t.Fatalf("expected self link for /feed/atom/, got %q", body[:min(500, len(body))])
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
	if !strings.Contains(body, `xmlns:admin="http://webns.net/mvcb/"`) {
		t.Fatalf("expected admin xmlns for generatorAgent: %q", body[:min(500, len(body))])
	}
	if !strings.Contains(body, `<admin:generatorAgent rdf:resource="https://wordpress.org/?v=`) {
		t.Fatalf("expected RDF generator (P3.56): %q", body[:min(600, len(body))])
	}
	if !strings.Contains(body, "<rdf:Seq>") || !strings.Contains(body, "<item rdf:about=") {
		t.Fatalf("expected rdf:Seq and items, got %q", body[:min(800, len(body))])
	}
}

func TestFrontFeedRDFUpdateFiltersAndNamespaces(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("rss_update_period", func(value any, _ ...any) any { return "daily" }, 10, 1)
	c.Hooks.AddFilter("rss_update_frequency", func(value any, _ ...any) any { return 3 }, 10, 1)
	c.Hooks.AddFilter("wp_feed_namespaces", func(value any, args ...any) any {
		prev, _ := value.(string)
		if len(args) > 0 && args[0] == "rdf" {
			return prev + "\txmlns:taxo=\"http://purl.org/rss/1.0/modules/taxonomy/\"\n"
		}
		return value
	}, 10, 2)
	sawPre, sawHeader, sawItem := false, false, false
	c.Hooks.AddAction("rss_tag_pre", func(args ...any) {
		if len(args) > 0 && args[0] == "rdf" {
			sawPre = true
		}
	}, 10, 1)
	c.Hooks.AddAction("rdf_header", func(args ...any) { sawHeader = true }, 10, 0)
	c.Hooks.AddAction("rdf_item", func(args ...any) { sawItem = true }, 10, 0)

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rdf", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<sy:updatePeriod>\n\tdaily\n\t</sy:updatePeriod>",
		"<sy:updateFrequency>\n\t3\n\t</sy:updateFrequency>",
		`xmlns:taxo="http://purl.org/rss/1.0/modules/taxonomy/"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body[:min(900, len(body))])
		}
	}
	if !sawPre || !sawHeader || !sawItem {
		t.Fatalf("hooks pre=%v header=%v item=%v", sawPre, sawHeader, sawItem)
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
	// P3.58: comments Atom self uses feedSelfLink (request URI).
	if !strings.Contains(body, `rel="self"`) || !strings.Contains(body, `href="`) {
		t.Fatalf("expected rel=self href: %q", body[:min(500, len(body))])
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

func TestFrontCollapseDoubleSlash(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/sample-page//", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if strings.Contains(loc, "//sample") || strings.Contains(strings.TrimPrefix(loc, "http://"), "//") {
		// Allow scheme:// but not path //.
		pathPart := loc
		if i := strings.Index(loc, "://"); i >= 0 {
			pathPart = loc[i+3:]
			if j := strings.Index(pathPart, "/"); j >= 0 {
				pathPart = pathPart[j:]
			}
		}
		if strings.Contains(pathPart, "//") {
			t.Fatalf("Location still has // in path: %q", loc)
		}
	}
	if !strings.Contains(loc, "/sample-page") {
		t.Fatalf("Location=%q", loc)
	}
}

func TestFrontStripTrailingSlashWhenStructureOmits(t *testing.T) {
	c := testBoot(t)
	_, _ = c.Options.UpdateOption("permalink_structure", "/%year%/%monthnum%/%day%/%postname%", "yes")
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("permalink_structure", "/%year%/%monthnum%/%day%/%postname%/", "yes")
	})
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/sample-page/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if strings.HasSuffix(loc, "/sample-page/") || !strings.HasSuffix(loc, "/sample-page") {
		t.Fatalf("expected strip trailing slash, Location=%q", loc)
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

func TestFrontSearchRobotsNoindex(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?s=Hello", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	// P3.104: search uses wp_robots_no_robots → noindex, follow + max-image-preview:large.
	want := `content='noindex, follow, max-image-preview:large'`
	if !strings.Contains(body, want) {
		t.Fatalf("expected search robots %q in %q", want, body)
	}
}

func TestFrontRobotsFilter(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("wp_robots", func(value any, _ ...any) any {
		m, _ := value.(map[string]any)
		if m == nil {
			m = map[string]any{}
		}
		m["noarchive"] = true
		return m
	}, 10, 1)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "noarchive") || !strings.Contains(body, "max-image-preview:large") {
		t.Fatalf("expected wp_robots filter output: %q", body)
	}
}

func TestFrontSearchDocumentTitlePartsFilter(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("document_title_parts", func(value any, _ ...any) any {
		m, _ := value.(map[string]string)
		if m == nil {
			m = map[string]string{}
		}
		m["title"] = "Looking for Hello"
		return m
	}, 10, 1)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?s=Hello", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<title>Looking for Hello &#8211;") {
		t.Fatalf("expected filtered title: %q", rec.Body.String()[:min(400, rec.Body.Len())])
	}
}

func TestFrontSearchTemplateContentFilter(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("search_template_content", func(value any, _ ...any) any {
		return `<!-- wp:template-part {"slug":"header"} /-->` +
			`<!-- wp:paragraph --><p class="search-filter-marker">Custom search shell</p><!-- /wp:paragraph -->` +
			`<!-- wp:template-part {"slug":"footer"} /-->`
	}, 10, 1)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?s=Hello", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "search-filter-marker") {
		t.Fatalf("expected search_template_content filter: %q", rec.Body.String()[:min(800, rec.Body.Len())])
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

func TestFrontAuthorArchiveDocumentTitleParts(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("document_title_parts", func(value any, _ ...any) any {
		parts, _ := value.(map[string]string)
		if parts == nil {
			parts = map[string]string{}
		}
		parts["title"] = "Custom Author Title"
		parts["site"] = "Custom Site"
		return parts
	}, 10, 1)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/author/admin/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Custom Author Title") || !strings.Contains(body, "Custom Site") {
		t.Fatalf("expected document_title_parts: %q", body[:min(600, len(body))])
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
	if !strings.Contains(body, "tag-fixture-tag-01") {
		t.Fatalf("expected tag slug body class: %q", body[:min(500, len(body))])
	}
	if strings.Contains(body, "GoPress: front route not implemented") {
		t.Fatal("tag archive should not 501")
	}
	// P3.103: feed_links_extra tag alternate.
	if !strings.Contains(body, "Tag Feed") || !strings.Contains(body, "/tag/fixture-tag-01/feed/") {
		t.Fatalf("expected tag feed_links_extra alternate: %q", body[:min(1200, len(body))])
	}

	req = httptest.NewRequest(http.MethodGet, "/tag/nobody-here-tag/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing tag want 404, got %d", rec.Code)
	}
}

func TestFrontTagArchiveDocumentTitleFilter(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("document_title_parts", func(value any, _ ...any) any {
		parts, _ := value.(map[string]string)
		if parts == nil {
			parts = map[string]string{}
		}
		parts["title"] = "Tags Only"
		parts["site"] = "GoPress Test"
		return parts
	}, 10, 1)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/tag/fixture-tag-01/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Tags Only") || !strings.Contains(body, "GoPress Test") {
		t.Fatalf("document_title_parts not applied: %q", body[:min(600, len(body))])
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
	if !strings.Contains(body, "category-uncategorized") && !strings.Contains(body, "category-1") {
		t.Fatalf("expected category slug/id body class: %q", body[:min(500, len(body))])
	}
	// P3.103: feed_links_extra category alternate.
	if !strings.Contains(body, "Category Feed") || !strings.Contains(body, "/category/uncategorized/feed/") {
		t.Fatalf("expected category feed_links_extra alternate: %q", body[:min(1200, len(body))])
	}

	req = httptest.NewRequest(http.MethodGet, "/category/nobody-here-cat/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing category want 404, got %d", rec.Code)
	}
}

func TestFrontCategoryArchiveDocumentTitleFilter(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("document_title_parts", func(value any, _ ...any) any {
		parts, _ := value.(map[string]string)
		if parts == nil {
			parts = map[string]string{}
		}
		parts["title"] = "Cats Only"
		parts["site"] = "GoPress Test"
		return parts
	}, 10, 1)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/category/uncategorized/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Cats Only") || !strings.Contains(body, "GoPress Test") {
		t.Fatalf("document_title_parts not applied: %q", body[:min(600, len(body))])
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
	if !strings.Contains(body, "date-year") || !strings.Contains(body, "date-2024") {
		t.Fatalf("expected date-year body class: %q", body[:min(500, len(body))])
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
	if !strings.Contains(body, "date-month") || !strings.Contains(body, "date-202406") {
		t.Fatalf("expected date-month body class: %q", body[:min(500, len(body))])
	}

	req = httptest.NewRequest(http.MethodGet, "/2019/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("empty year want 404, got %d body=%q", rec.Code, rec.Body.String()[:min(200, rec.Body.Len())])
	}
}

func TestFrontDateArchiveDocumentTitleFilter(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("document_title_parts", func(value any, _ ...any) any {
		parts, _ := value.(map[string]string)
		if parts == nil {
			parts = map[string]string{}
		}
		parts["title"] = "Dates Only"
		parts["site"] = "GoPress Test"
		return parts
	}, 10, 1)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/2024/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Dates Only") || !strings.Contains(body, "GoPress Test") {
		t.Fatalf("document_title_parts not applied: %q", body[:min(600, len(body))])
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
