package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestPostsListAnon(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?per_page=5", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatalf("json array: %v body=%s", err, rec.Body.String())
	}
	if len(posts) == 0 || len(posts) > 5 {
		t.Fatalf("len=%d", len(posts))
	}
	p := posts[0]
	if p.ID <= 0 || p.Slug == "" || p.Status != "publish" || p.Type != "post" {
		t.Fatalf("unexpected post: %+v", p)
	}
	if p.Title.Rendered == "" {
		t.Fatal("expected title.rendered")
	}
	if p.Date == "" || !containsT(p.Date) {
		t.Fatalf("date=%q", p.Date)
	}
	if p.Link == "" {
		t.Fatal("expected link")
	}
}

func TestPostSingle(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var post rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if post.ID != 1 || post.Type != "post" || post.Slug != "hello-world" {
		t.Fatalf("unexpected post: %+v", post)
	}
	if post.Title.Rendered == "" || post.Link == "" || post.Date == "" {
		t.Fatalf("missing fields: %+v", post)
	}
	if !containsT(post.Date) {
		t.Fatalf("date=%q", post.Date)
	}
	if post.GUID.Rendered == "" {
		t.Fatal("expected guid.rendered")
	}
	if post.Modified == "" || post.ModifiedGMT == "" {
		t.Fatalf("expected modified fields: %+v", post)
	}
	if post.Author <= 0 {
		t.Fatalf("author=%d", post.Author)
	}
	if post.CommentStatus == "" || post.PingStatus == "" {
		t.Fatalf("comment/ping status missing: %+v", post)
	}
	if post.Format == "" {
		t.Fatal("expected format")
	}
	if post.Meta == nil {
		t.Fatal("expected meta object")
	}
	if post.Categories == nil || post.Tags == nil {
		t.Fatal("expected categories/tags arrays")
	}
	if len(post.Categories) == 0 {
		t.Fatal("hello-world should have a category")
	}
}

func TestPostsOrderbyQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	for _, tc := range []struct {
		q    string
		cmp  func(a, b rest.Post) bool // true if a should come before or equal b
		name string
	}{
		{
			q:    "orderby=title&order=asc&per_page=5",
			name: "title asc",
			cmp: func(a, b rest.Post) bool {
				return strings.ToLower(a.Title.Rendered) <= strings.ToLower(b.Title.Rendered)
			},
		},
		{
			q:    "orderby=id&order=asc&per_page=5",
			name: "id asc",
			cmp:  func(a, b rest.Post) bool { return a.ID <= b.ID },
		},
		{
			q:    "orderby=slug&order=asc&per_page=5",
			name: "slug asc",
			cmp:  func(a, b rest.Post) bool { return a.Slug <= b.Slug },
		},
		{
			q:    "orderby=date&order=desc&per_page=5",
			name: "date desc",
			cmp:  func(a, b rest.Post) bool { return a.Date >= b.Date },
		},
	} {
		req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?"+tc.q, nil)
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, rc)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", tc.name, rec.Code, rec.Body.String())
		}
		var posts []rest.Post
		if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
			t.Fatalf("%s decode: %v", tc.name, err)
		}
		if len(posts) < 2 {
			t.Fatalf("%s need ≥2 posts, got %d", tc.name, len(posts))
		}
		for i := 1; i < len(posts); i++ {
			if !tc.cmp(posts[i-1], posts[i]) {
				t.Fatalf("%s order broken at %d: %#v then %#v", tc.name, i, posts[i-1], posts[i])
			}
		}
	}
}

func TestPostsSearchQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=Hello&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("search=Hello status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected search hits for Hello")
	}
	found := false
	for _, p := range hits {
		if strings.Contains(strings.ToLower(p.Title.Rendered), "hello") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Hello in titles, got %+v", hits)
	}
	total := rec.Header().Get("X-WP-Total")
	if total == "" || total == "0" {
		t.Fatalf("expected X-WP-Total > 0, got %q", total)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=zzznomatchterm999&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("miss status=%d", rec.Code)
	}
	var miss []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("expected empty miss, got %+v", miss)
	}
	if rec.Header().Get("X-WP-Total") != "0" {
		t.Fatalf("expected X-WP-Total=0, got %q", rec.Header().Get("X-WP-Total"))
	}
}

func TestPostsAuthorQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Fixture posts are authored by admin (ID 1).
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?author=1&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("author=1 status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected posts for author=1")
	}
	for _, p := range hits {
		if p.Author != 1 {
			t.Fatalf("expected author=1, got %d on post %d", p.Author, p.ID)
		}
	}
	total := rec.Header().Get("X-WP-Total")
	if total == "" || total == "0" {
		t.Fatalf("expected X-WP-Total > 0, got %q", total)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?author=99999&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("author miss status=%d", rec.Code)
	}
	var miss []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("expected empty for unknown author, got %+v", miss)
	}
	if rec.Header().Get("X-WP-Total") != "0" {
		t.Fatalf("expected X-WP-Total=0, got %q", rec.Header().Get("X-WP-Total"))
	}
}

func TestPostsStickyQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Fixture sticky_posts = [13] (Fixture Post 01).
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?sticky=true&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("sticky=true status=%d body=%s", rec.Code, rec.Body.String())
	}
	var stickyPosts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &stickyPosts); err != nil {
		t.Fatal(err)
	}
	if len(stickyPosts) != 1 || stickyPosts[0].ID != 13 || !stickyPosts[0].Sticky {
		t.Fatalf("sticky=true got %+v", stickyPosts)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?sticky=false&per_page=20", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("sticky=false status=%d body=%s", rec.Code, rec.Body.String())
	}
	var nonSticky []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &nonSticky); err != nil {
		t.Fatal(err)
	}
	for _, p := range nonSticky {
		if p.ID == 13 || p.Sticky {
			t.Fatalf("sticky=false included sticky post %+v", p)
		}
	}
}

func TestPostSingleMissing(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/999999991", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPostFeaturedMediaLink(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	postID := int64(13)
	_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "_thumbnail_id", "")
	if _, _, err := c.Meta.AddMetadata(wp.MetaPost, postID, "_thumbnail_id", "5", true); err != nil {
		t.Fatalf("set thumbnail: %v", err)
	}
	t.Cleanup(func() {
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "_thumbnail_id", "")
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/13", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var post rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatal(err)
	}
	if post.FeaturedMedia != 5 {
		t.Fatalf("featured_media=%d want 5", post.FeaturedMedia)
	}
	if post.Links["wp:featuredmedia"] == nil {
		t.Fatalf("expected wp:featuredmedia link: %+v", post.Links)
	}
}

func containsT(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == 'T' {
			return true
		}
	}
	return false
}
