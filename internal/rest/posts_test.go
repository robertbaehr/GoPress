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

func TestPostsStatusQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Anonymous cannot request draft.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?status=draft&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anon draft status=%d body=%s", rec.Code, rec.Body.String())
	}

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?status=draft&per_page=10", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("editor draft status=%d body=%s", rec.Code, rec.Body.String())
	}
	var drafts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &drafts); err != nil {
		t.Fatal(err)
	}
	if len(drafts) == 0 {
		t.Fatal("expected draft posts for editor")
	}
	for _, p := range drafts {
		if p.Status != "draft" {
			t.Fatalf("expected status=draft, got %q on post %d", p.Status, p.ID)
		}
	}
	if rec.Header().Get("X-WP-Total") == "" || rec.Header().Get("X-WP-Total") == "0" {
		t.Fatalf("expected X-WP-Total > 0, got %q", rec.Header().Get("X-WP-Total"))
	}

	// Default status=publish still works without auth.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?status=publish&per_page=5", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("publish status=%d", rec.Code)
	}
	var pubs []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &pubs); err != nil {
		t.Fatal(err)
	}
	for _, p := range pubs {
		if p.Status != "publish" {
			t.Fatalf("expected publish, got %q", p.Status)
		}
	}
}

func TestPostsTagsQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Fixture Tag 01 (term_id 5) is on Fixture Post 01.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?tags=5&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("tags=5 status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected posts for tags=5")
	}
	found := false
	for _, p := range hits {
		has := false
		for _, id := range p.Tags {
			if id == 5 {
				has = true
				break
			}
		}
		if !has {
			t.Fatalf("post %d missing tag 5: %+v", p.ID, p.Tags)
		}
		if p.Slug == "fixture-post-01" || strings.Contains(p.Title.Rendered, "Fixture Post 01") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Fixture Post 01 among tags=5 hits")
	}
	if rec.Header().Get("X-WP-Total") == "" || rec.Header().Get("X-WP-Total") == "0" {
		t.Fatalf("expected X-WP-Total > 0, got %q", rec.Header().Get("X-WP-Total"))
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?tags=99999&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("tags miss status=%d", rec.Code)
	}
	var miss []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("expected empty for unknown tag, got %+v", miss)
	}
	if rec.Header().Get("X-WP-Total") != "0" {
		t.Fatalf("expected X-WP-Total=0, got %q", rec.Header().Get("X-WP-Total"))
	}
}

func TestPostsCategoriesQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Uncategorized is term_id 1; Hello world is in that category.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?categories=1&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("categories=1 status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected posts for categories=1")
	}
	foundHello := false
	for _, p := range hits {
		has := false
		for _, id := range p.Categories {
			if id == 1 {
				has = true
				break
			}
		}
		if !has {
			t.Fatalf("post %d missing category 1: %+v", p.ID, p.Categories)
		}
		if p.Slug == "hello-world" || strings.Contains(p.Title.Rendered, "Hello world") {
			foundHello = true
		}
	}
	if !foundHello {
		t.Fatalf("expected Hello world among categories=1 hits")
	}
	if rec.Header().Get("X-WP-Total") == "" || rec.Header().Get("X-WP-Total") == "0" {
		t.Fatalf("expected X-WP-Total > 0, got %q", rec.Header().Get("X-WP-Total"))
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?categories=99999&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("categories miss status=%d", rec.Code)
	}
	var miss []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("expected empty for unknown category, got %+v", miss)
	}
	if rec.Header().Get("X-WP-Total") != "0" {
		t.Fatalf("expected X-WP-Total=0, got %q", rec.Header().Get("X-WP-Total"))
	}
}

func TestPostsSlugQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?slug=hello-world&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("slug=hello-world status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 post for hello-world, got %d (%+v)", len(hits), hits)
	}
	if hits[0].Slug != "hello-world" {
		t.Fatalf("expected slug hello-world, got %q", hits[0].Slug)
	}
	if rec.Header().Get("X-WP-Total") != "1" {
		t.Fatalf("expected X-WP-Total=1, got %q", rec.Header().Get("X-WP-Total"))
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?slug=hello-world,fixture-post-01&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("slug multi status=%d body=%s", rec.Code, rec.Body.String())
	}
	var multi []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &multi); err != nil {
		t.Fatal(err)
	}
	if len(multi) < 2 {
		t.Fatalf("expected >=2 posts for multi slug, got %d", len(multi))
	}
	seen := map[string]bool{}
	for _, p := range multi {
		seen[p.Slug] = true
	}
	if !seen["hello-world"] || !seen["fixture-post-01"] {
		t.Fatalf("expected both slugs in multi result: %+v", seen)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?slug=no-such-slug-xyz&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("slug miss status=%d", rec.Code)
	}
	var miss []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("expected empty for unknown slug, got %+v", miss)
	}
	if rec.Header().Get("X-WP-Total") != "0" {
		t.Fatalf("expected X-WP-Total=0, got %q", rec.Header().Get("X-WP-Total"))
	}
}

func TestPostsIncludeExcludeQueryParams(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Hello world is ID 1; Fixture Post 01 is ID 13.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?include=1&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("include=1 status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != 1 {
		t.Fatalf("include=1 got %+v", hits)
	}
	if rec.Header().Get("X-WP-Total") != "1" {
		t.Fatalf("expected X-WP-Total=1, got %q", rec.Header().Get("X-WP-Total"))
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?include=1,13&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("include multi status=%d", rec.Code)
	}
	var multi []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &multi); err != nil {
		t.Fatal(err)
	}
	if len(multi) != 2 {
		t.Fatalf("expected 2 for include=1,13 got %d", len(multi))
	}
	seen := map[int64]bool{}
	for _, p := range multi {
		seen[p.ID] = true
	}
	if !seen[1] || !seen[13] {
		t.Fatalf("expected IDs 1 and 13, got %+v", seen)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?exclude=1&per_page=20", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("exclude=1 status=%d", rec.Code)
	}
	var excl []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &excl); err != nil {
		t.Fatal(err)
	}
	for _, p := range excl {
		if p.ID == 1 {
			t.Fatalf("exclude=1 still returned post 1")
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?include=99999&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("include miss status=%d", rec.Code)
	}
	var miss []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("expected empty for unknown include, got %+v", miss)
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

func TestPostsAfterBeforeQueryParams(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Fixture dates span 2024-02-01 .. later; after mid-February should drop early posts.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?after=2024-02-03T00:00:00&per_page=100", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("after status=%d body=%s", rec.Code, rec.Body.String())
	}
	var afterHits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &afterHits); err != nil {
		t.Fatal(err)
	}
	if len(afterHits) == 0 {
		t.Fatal("expected posts after 2024-02-03")
	}
	for _, p := range afterHits {
		if p.Date != "" && p.Date < "2024-02-03T00:00:00" {
			t.Fatalf("post %d date %q before after bound", p.ID, p.Date)
		}
	}
	afterTotal := rec.Header().Get("X-WP-Total")

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?before=2024-02-02T00:00:00&per_page=100", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("before status=%d", rec.Code)
	}
	var beforeHits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &beforeHits); err != nil {
		t.Fatal(err)
	}
	if len(beforeHits) == 0 {
		t.Fatal("expected posts before 2024-02-02")
	}
	for _, p := range beforeHits {
		if p.Date != "" && p.Date > "2024-02-02T00:00:00" {
			t.Fatalf("post %d date %q after before bound", p.ID, p.Date)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?after=2024-02-01T00:00:00&before=2024-02-02T23:59:59&per_page=100", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("range status=%d", rec.Code)
	}
	var rangeHits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &rangeHits); err != nil {
		t.Fatal(err)
	}
	if len(rangeHits) == 0 {
		t.Fatal("expected posts in Feb 1-2 range")
	}
	if afterTotal == "" || afterTotal == "0" {
		t.Fatalf("expected after X-WP-Total > 0, got %q", afterTotal)
	}

	// Far-future after → empty.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?after=2099-01-01T00:00:00&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	var miss []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("expected empty for future after, got %d", len(miss))
	}
	if rec.Header().Get("X-WP-Total") != "0" {
		t.Fatalf("expected X-WP-Total=0, got %q", rec.Header().Get("X-WP-Total"))
	}
}

func TestPostsOffset(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Baseline first page.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?per_page=2&orderby=id&order=asc", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("baseline status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page1 []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &page1); err != nil {
		t.Fatal(err)
	}
	if len(page1) < 2 {
		t.Fatalf("need ≥2 posts for offset test, got %d", len(page1))
	}
	total := rec.Header().Get("X-WP-Total")

	// offset=1 should skip the first item (takes precedence over page).
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?per_page=2&orderby=id&order=asc&offset=1&page=1", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("offset status=%d", rec.Code)
	}
	var offsetHits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &offsetHits); err != nil {
		t.Fatal(err)
	}
	if len(offsetHits) == 0 {
		t.Fatal("expected posts with offset=1")
	}
	if offsetHits[0].ID == page1[0].ID {
		t.Fatalf("offset=1 should skip first post id=%d", page1[0].ID)
	}
	if offsetHits[0].ID != page1[1].ID {
		t.Fatalf("offset=1 first=%d want %d", offsetHits[0].ID, page1[1].ID)
	}
	if rec.Header().Get("X-WP-Total") != total {
		t.Fatalf("offset should not change X-WP-Total: got %q want %q", rec.Header().Get("X-WP-Total"), total)
	}

	// Large offset → empty collection, total unchanged.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?per_page=5&offset=99999", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	var empty []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty for huge offset, got %d", len(empty))
	}
}
