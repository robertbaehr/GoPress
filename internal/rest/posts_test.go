package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
