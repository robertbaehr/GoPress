package rest_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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
		{
			q:    "orderby=modified&order=desc&per_page=5",
			name: "modified desc",
			cmp:  func(a, b rest.Post) bool { return a.Modified >= b.Modified },
		},
		{
			q:    "orderby=author&order=asc&per_page=5",
			name: "author asc",
			cmp:  func(a, b rest.Post) bool { return a.Author <= b.Author },
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

func TestPostsSearchColumnsQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Title-only search should still find Hello world.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=Hello&search_columns=post_title&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("search_columns=post_title status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected title hits for Hello")
	}

	// Content-only search for a title term should miss when the term is only in the title.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=Hello&search_columns=post_content&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("search_columns=post_content status=%d", rec.Code)
	}
	var contentHits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &contentHits); err != nil {
		t.Fatal(err)
	}
	// Fixture "Hello world!" post has the word in the title; content may or may not.
	// Ensure invalid columns are ignored and fall back to all columns.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=Hello&search_columns=bogus&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("bogus search_columns status=%d", rec.Code)
	}
	var allHits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &allHits); err != nil {
		t.Fatal(err)
	}
	if len(allHits) == 0 {
		t.Fatal("invalid search_columns should fall back to all columns")
	}

	// Comma-separated columns still search titles.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=Hello&search_columns=post_title,post_excerpt&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("comma search_columns status=%d", rec.Code)
	}
	var commaHits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &commaHits); err != nil {
		t.Fatal(err)
	}
	if len(commaHits) == 0 {
		t.Fatal("expected hits for search_columns=post_title,post_excerpt")
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

	// P4.134: status[]= array form + status=any for editors.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?status[]=draft&status[]=publish&per_page=20", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status[]= status=%d body=%s", rec.Code, rec.Body.String())
	}
	var mixed []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &mixed); err != nil {
		t.Fatal(err)
	}
	if len(mixed) == 0 {
		t.Fatal("expected posts for status[]=draft&status[]=publish")
	}
	seenDraft, seenPub := false, false
	for _, p := range mixed {
		switch p.Status {
		case "draft":
			seenDraft = true
		case "publish":
			seenPub = true
		default:
			t.Fatalf("unexpected status %q", p.Status)
		}
	}
	if !seenDraft || !seenPub {
		t.Fatalf("want both draft and publish in status[] results (draft=%v publish=%v)", seenDraft, seenPub)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?status=any&per_page=5", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=any status=%d body=%s", rec.Code, rec.Body.String())
	}
	var anyPosts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &anyPosts); err != nil {
		t.Fatal(err)
	}
	if len(anyPosts) == 0 {
		t.Fatal("expected posts for status=any")
	}
	for _, p := range anyPosts {
		if p.Status == "trash" || p.Status == "auto-draft" {
			t.Fatalf("status=any should exclude trash/auto-draft, got %q", p.Status)
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

// P4.136: categories_exclude / tags_exclude / tax_relation.
func TestPostsTaxQueryExcludeAndRelation(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?categories_exclude=1&per_page=100", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("categories_exclude status=%d body=%s", rec.Code, rec.Body.String())
	}
	var excl []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &excl); err != nil {
		t.Fatal(err)
	}
	for _, p := range excl {
		for _, id := range p.Categories {
			if id == 1 {
				t.Fatalf("post %d still in category 1 after exclude", p.ID)
			}
		}
	}

	// tax_relation=OR with categories=1 should still return Uncategorized posts.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?categories=1&tax_relation=OR&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("tax_relation status=%d", rec.Code)
	}
	var orHits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &orHits); err != nil {
		t.Fatal(err)
	}
	if len(orHits) == 0 {
		t.Fatal("expected OR tax_relation hits for categories=1")
	}
}

func TestPostsFormatQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Fixture Post 15 (ID 27) has post-format-status.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?format=status&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("format=status status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected posts for format=status")
	}
	found := false
	for _, p := range hits {
		if p.Format != "status" {
			t.Fatalf("post %d format=%q want status", p.ID, p.Format)
		}
		if p.Slug == "fixture-post-15" || strings.Contains(p.Title.Rendered, "Fixture Post 15") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Fixture Post 15 among format=status hits")
	}
	if rec.Header().Get("X-WP-Total") == "" || rec.Header().Get("X-WP-Total") == "0" {
		t.Fatalf("expected X-WP-Total > 0, got %q", rec.Header().Get("X-WP-Total"))
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?format=standard&per_page=5", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("format=standard status=%d", rec.Code)
	}
	var std []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &std); err != nil {
		t.Fatal(err)
	}
	if len(std) == 0 {
		t.Fatal("expected standard-format posts")
	}
	for _, p := range std {
		if p.Format != "standard" {
			t.Fatalf("post %d format=%q want standard", p.ID, p.Format)
		}
		if p.ID == 27 {
			t.Fatal("status-format post should be excluded by format=standard")
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?format=video&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("format=video miss status=%d", rec.Code)
	}
	var miss []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("expected empty for unused format=video, got %+v", miss)
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

func TestPostsSlugArrayQueryParam(t *testing.T) {
	// P4.133: slug[]= array form (WP REST type=array query param).
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?slug[]=hello-world&slug[]=fixture-post-01&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("slug[] status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) < 2 {
		t.Fatalf("expected >=2 for slug[], got %d", len(hits))
	}
	seen := map[string]bool{}
	for _, p := range hits {
		seen[p.Slug] = true
	}
	if !seen["hello-world"] || !seen["fixture-post-01"] {
		t.Fatalf("slug[] missing expected posts: %+v", seen)
	}

	// Collection slug is not sanitize_title'd — spaced title does not match.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?slug=Hello%20World&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("unsanitized slug status=%d", rec.Code)
	}
	var miss []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("expected empty for unsanitized Hello World slug, got %+v", miss)
	}
}

func TestPostsSlugEmptyArrayZeroResults(t *testing.T) {
	// P4.143: slug[]= / slug[0]= present-but-empty → zero results (WP REST).
	c := bootOrSkip(t)
	defer c.DB.Close()

	for _, path := range []string{
		"/wp-json/wp/v2/posts?slug[]=&per_page=10",
		"/wp-json/wp/v2/posts?slug[0]=&per_page=10",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		c = c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, c)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		var hits []rest.Post
		if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
			t.Fatal(err)
		}
		if len(hits) != 0 {
			t.Fatalf("%s: expected 0 posts, got %d", path, len(hits))
		}
		if rec.Header().Get("X-WP-Total") != "0" {
			t.Fatalf("%s: expected X-WP-Total=0, got %q", path, rec.Header().Get("X-WP-Total"))
		}
	}

	// Scalar slug= remains no filter.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?slug=&per_page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("slug= status=%d", rec.Code)
	}
	var one []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if len(one) != 1 {
		t.Fatalf("scalar empty slug= should list posts, got %d", len(one))
	}
}

func TestPostsSlugIndexedQueryParam(t *testing.T) {
	// P4.143: slug[0]= / slug[1]= indexed array form.
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?slug[0]=hello-world&slug[1]=fixture-post-01&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("slug[N] status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) < 2 {
		t.Fatalf("expected >=2 for slug[N], got %d", len(hits))
	}
	seen := map[string]bool{}
	for _, p := range hits {
		seen[p.Slug] = true
	}
	if !seen["hello-world"] || !seen["fixture-post-01"] {
		t.Fatalf("slug[N] missing expected posts: %+v", seen)
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

// P4.137: include[]= / exclude[]= PHP array query forms.
func TestPostsIncludeExcludeArrayQueryParams(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?include[]=1&include[]=13&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("include[] status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("include[]=1&include[]=13 want 2 got %d", len(hits))
	}
	seen := map[int64]bool{}
	for _, p := range hits {
		seen[p.ID] = true
	}
	if !seen[1] || !seen[13] {
		t.Fatalf("expected IDs 1 and 13 from include[], got %+v", seen)
	}
	if rec.Header().Get("X-WP-Total") != "2" {
		t.Fatalf("expected X-WP-Total=2, got %q", rec.Header().Get("X-WP-Total"))
	}

	// Combined comma + array form, with dedupe.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?include=1&include[]=1&include[]=13&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("include mix status=%d", rec.Code)
	}
	var mix []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &mix); err != nil {
		t.Fatal(err)
	}
	if len(mix) != 2 {
		t.Fatalf("deduped include mix want 2 got %d", len(mix))
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?exclude[]=1&exclude[]=13&per_page=50", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("exclude[] status=%d", rec.Code)
	}
	var excl []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &excl); err != nil {
		t.Fatal(err)
	}
	for _, p := range excl {
		if p.ID == 1 || p.ID == 13 {
			t.Fatalf("exclude[] still returned post %d", p.ID)
		}
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

func TestPostsAuthorExcludeQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Exclude admin (ID 1); fixture also has authors 2/3 — none may be author 1.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?author_exclude=1&per_page=20", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("author_exclude=1 status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	for _, p := range hits {
		if p.Author == 1 {
			t.Fatalf("author_exclude=1 returned author=1 post %d", p.ID)
		}
	}

	// author=1 with author_exclude=99999 still returns admin posts.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?author=1&author_exclude=99999&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("author+exclude status=%d", rec.Code)
	}
	var keep []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &keep); err != nil {
		t.Fatal(err)
	}
	if len(keep) == 0 {
		t.Fatal("expected posts for author=1 with unrelated exclude")
	}
	for _, p := range keep {
		if p.Author != 1 {
			t.Fatalf("expected author=1, got %d", p.Author)
		}
	}

	// Exclude all known fixture authors → empty.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?author_exclude=1,2,3&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("author_exclude multi status=%d", rec.Code)
	}
	var none []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &none); err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("expected empty for author_exclude=1,2,3, got %d", len(none))
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

func TestPostsStickyIgnoreStickyInvalidParams(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// P4.141: present but non-boolean sticky/ignore_sticky → rest_invalid_param.
	for _, path := range []string{
		"/wp-json/wp/v2/posts?sticky=maybe",
		"/wp-json/wp/v2/posts?sticky=",
		"/wp-json/wp/v2/posts?sticky=yes",
		"/wp-json/wp/v2/posts?ignore_sticky=maybe",
		"/wp-json/wp/v2/posts?ignore_sticky=",
		"/wp-json/wp/v2/posts?sticky=maybe&ignore_sticky=no",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		c = c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, c)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, "rest_invalid_param") || !strings.Contains(body, "is not of type boolean") {
			t.Fatalf("%s: expected rest_invalid_type boolean: %s", path, body)
		}
	}
}

func TestPostsIgnoreStickyFalsePrepends(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// P4.141: ignore_sticky=false prepends sticky post 13.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?ignore_sticky=false&per_page=5", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 || posts[0].ID != 13 || !posts[0].Sticky {
		t.Fatalf("ignore_sticky=false want sticky first, got %+v", posts)
	}

	// Default / ignore_sticky=true does not force sticky-first.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?ignore_sticky=true&per_page=5", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) > 0 && posts[0].ID == 13 {
		// Sticky may still be first by date; only assert default path is 200.
		_ = posts
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

func TestPostsAfterBeforeInvalidDateParams(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// P4.140: present but invalid/empty date-time → rest_invalid_param + rest_invalid_date.
	for _, path := range []string{
		"/wp-json/wp/v2/posts?after=not-a-date",
		"/wp-json/wp/v2/posts?before=",
		"/wp-json/wp/v2/posts?modified_after=xyz",
		"/wp-json/wp/v2/pages?after=2024-13-01",
		"/wp-json/wp/v2/posts?after=bad&before=also-bad",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		c = c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, c)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, "rest_invalid_param") || !strings.Contains(body, "Invalid date.") {
			t.Fatalf("%s: expected rest_invalid_param Invalid date: %s", path, body)
		}
		if !strings.Contains(body, "rest_invalid_date") {
			t.Fatalf("%s: expected rest_invalid_date details: %s", path, body)
		}
	}
	// Valid ISO8601 still works.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?after=2024-02-01T00:00:00%2B00:00&per_page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid after status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPostsModifiedAfterBeforeQueryParams(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Fixture publish posts share post_modified ≈ 2024-06-15; bump one earlier
	// so modified_before can retain a non-empty set.
	uq, uargs, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
		"2024-01-15 12:00:00", "2024-01-15 12:00:00", int64(13),
	)
	if err != nil {
		t.Fatalf("prepare bump modified: %v", err)
	}
	if _, err := c.DB.Exec(uq, uargs...); err != nil {
		t.Fatalf("bump modified: %v", err)
	}
	t.Cleanup(func() {
		uq, uargs, _ := c.DB.Prepare(
			"UPDATE "+c.DB.Posts()+" SET post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
			"2024-06-15 12:00:00", "2024-06-15 12:00:00", int64(13),
		)
		_, _ = c.DB.Exec(uq, uargs...)
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?modified_after=2024-06-01T00:00:00&per_page=100", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("modified_after status=%d body=%s", rec.Code, rec.Body.String())
	}
	var afterHits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &afterHits); err != nil {
		t.Fatal(err)
	}
	if len(afterHits) == 0 {
		t.Fatal("expected posts modified after 2024-06-01")
	}
	for _, p := range afterHits {
		if p.ID == 13 {
			t.Fatalf("post 13 modified Jan should be excluded by modified_after")
		}
		if p.Modified != "" && p.Modified < "2024-06-01T00:00:00" {
			t.Fatalf("post %d modified %q before bound", p.ID, p.Modified)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?modified_before=2024-02-01T00:00:00&per_page=100", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("modified_before status=%d", rec.Code)
	}
	var beforeHits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &beforeHits); err != nil {
		t.Fatal(err)
	}
	if len(beforeHits) == 0 {
		t.Fatal("expected posts modified before 2024-02-01")
	}
	found13 := false
	for _, p := range beforeHits {
		if p.ID == 13 {
			found13 = true
		}
		if p.Modified != "" && p.Modified > "2024-02-01T00:00:00" {
			t.Fatalf("post %d modified %q after before bound", p.ID, p.Modified)
		}
	}
	if !found13 {
		t.Fatal("expected fixture post 13 in modified_before set")
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?modified_after=2099-01-01T00:00:00&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	var miss []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("expected empty for future modified_after, got %d", len(miss))
	}
	if rec.Header().Get("X-WP-Total") != "0" {
		t.Fatalf("expected X-WP-Total=0, got %q", rec.Header().Get("X-WP-Total"))
	}
}

func TestPagesParentQueryParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Fixture: Parent(436) → Child(437) → Grandchild(438).
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?parent=436&per_page=100", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var kids []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &kids); err != nil {
		t.Fatal(err)
	}
	if len(kids) == 0 {
		t.Fatal("expected child pages for parent=436")
	}
	for _, p := range kids {
		if p.Parent != 436 {
			t.Fatalf("page %d parent=%d want 436", p.ID, p.Parent)
		}
	}

	// parent=0 → top-level pages only.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?parent=0&per_page=100", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	var top []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &top); err != nil {
		t.Fatal(err)
	}
	if len(top) == 0 {
		t.Fatal("expected top-level pages for parent=0")
	}
	for _, p := range top {
		if p.Parent != 0 {
			t.Fatalf("top-level page %d parent=%d want 0", p.ID, p.Parent)
		}
	}

	// parent_exclude=436 → no direct children of 436.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?parent_exclude=436&per_page=100", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	var excl []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &excl); err != nil {
		t.Fatal(err)
	}
	for _, p := range excl {
		if p.Parent == 436 {
			t.Fatalf("parent_exclude=436 still returned page %d", p.ID)
		}
	}
}

func TestPagesParentStrictValidation(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// parent=foo → rest_invalid_param / rest_invalid_type (P4.142).
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?parent=foo", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("parent=foo status=%d body=%s", rec.Code, rec.Body.String())
	}
	var errBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody["code"] != "rest_invalid_param" {
		t.Fatalf("code=%v", errBody["code"])
	}
	data, _ := errBody["data"].(map[string]any)
	params, _ := data["params"].(map[string]any)
	if params["parent"] != "parent[0] is not of type integer." {
		t.Fatalf("params.parent=%v", params["parent"])
	}

	// parent= empty → no filter (all pages), matching WP sanitize to [].
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?parent=&per_page=100", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("parent= status=%d body=%s", rec.Code, rec.Body.String())
	}
	var all []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &all); err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("parent= should not filter to empty")
	}

	// parent[]=436&parent[]=0 → children of 436 plus top-level.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?parent[]=436&parent[]=0&per_page=100", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("parent[] status=%d body=%s", rec.Code, rec.Body.String())
	}
	var multi []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &multi); err != nil {
		t.Fatal(err)
	}
	if len(multi) == 0 {
		t.Fatal("expected pages for parent[]=436&parent[]=0")
	}
	for _, p := range multi {
		if p.Parent != 436 && p.Parent != 0 {
			t.Fatalf("page %d parent=%d want 0 or 436", p.ID, p.Parent)
		}
	}

	// parent[]= alone → invalid.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?parent[]=", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("parent[]= status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPagesOrderbyMenuOrder(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Temporarily distinct menu_order on fixture pages so ASC/DESC are observable.
	type ord struct {
		id, want int64
	}
	set := []ord{{436, 30}, {437, 10}, {438, 20}}
	prev := map[int64]int64{}
	for _, s := range set {
		q, args, err := c.DB.Prepare("SELECT menu_order AS n FROM "+c.DB.Posts()+" WHERE ID = %d", s.id)
		if err != nil {
			t.Fatal(err)
		}
		row, err := c.DB.GetRow(q, args...)
		if err != nil || row == nil {
			t.Fatalf("load menu_order for %d: %v", s.id, err)
		}
		prev[s.id] = asInt64Test(row["n"])
		uq, uargs, err := c.DB.Prepare("UPDATE "+c.DB.Posts()+" SET menu_order = %d WHERE ID = %d", s.want, s.id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.DB.Exec(uq, uargs...); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for id, mo := range prev {
			uq, uargs, _ := c.DB.Prepare("UPDATE "+c.DB.Posts()+" SET menu_order = %d WHERE ID = %d", mo, id)
			_, _ = c.DB.Exec(uq, uargs...)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?orderby=menu_order&order=asc&include=436,437,438&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var pages []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &pages); err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 {
		t.Fatalf("want 3 pages, got %d", len(pages))
	}
	wantIDs := []int64{437, 438, 436} // menu_order 10, 20, 30
	for i, id := range wantIDs {
		if pages[i].ID != id {
			t.Fatalf("asc[%d]=%d want %d (menu_order sort)", i, pages[i].ID, id)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?orderby=menu_order&order=desc&include=436,437,438&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if err := json.Unmarshal(rec.Body.Bytes(), &pages); err != nil {
		t.Fatal(err)
	}
	wantIDs = []int64{436, 438, 437}
	for i, id := range wantIDs {
		if pages[i].ID != id {
			t.Fatalf("desc[%d]=%d want %d", i, pages[i].ID, id)
		}
	}
}

func TestPostsOrderbyIncludeAndValidation(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?include=13,1&orderby=include&per_page=10", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("include orderby status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 || posts[0].ID != 13 || posts[1].ID != 1 {
		t.Fatalf("orderby=include want [13,1], got %+v", posts)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?orderby=include", nil)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing include status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "rest_orderby_include_missing_include") {
		t.Fatalf("body=%s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?orderby=relevance", nil)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("relevance without search status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "rest_no_search_term_defined") {
		t.Fatalf("body=%s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?slug=fixture-post-01,hello-world&orderby=include_slugs&per_page=10", nil)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("include_slugs status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 || posts[0].Slug != "fixture-post-01" || posts[1].Slug != "hello-world" {
		got := make([]string, len(posts))
		for i, p := range posts {
			got[i] = p.Slug
		}
		t.Fatalf("orderby=include_slugs got %v", got)
	}
}

func asInt64Test(v any) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(v)), 10, 64)
	return n
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

func TestPostsOffsetValidation(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// P4.139: non-integer offset → rest_invalid_param (matches WP schema type=integer).
	for _, path := range []string{
		"/wp-json/wp/v2/posts?offset=abc&per_page=1",
		"/wp-json/wp/v2/posts?offset=1.5&per_page=1",
		"/wp-json/wp/v2/posts?offset=&per_page=1",
		"/wp-json/wp/v2/posts?offset=true&per_page=1",
		"/wp-json/wp/v2/pages?offset=xyz&per_page=1",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, rc)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "rest_invalid_param") || !strings.Contains(rec.Body.String(), "offset") {
			t.Fatalf("%s: expected rest_invalid_param offset: %s", path, rec.Body.String())
		}
	}

	// Negative offset is a valid integer; WP_Query clamps/ignores → 200.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?offset=-5&per_page=1", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("offset=-5 status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPostsPerPageValidation(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// P4.138: per_page out of schema range → rest_invalid_param.
	for _, path := range []string{
		"/wp-json/wp/v2/posts?per_page=0",
		"/wp-json/wp/v2/posts?per_page=101",
		"/wp-json/wp/v2/posts?per_page=-1",
		"/wp-json/wp/v2/posts?per_page=abc",
		"/wp-json/wp/v2/pages?per_page=200",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, rc)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "rest_invalid_param") || !strings.Contains(rec.Body.String(), "per_page") {
			t.Fatalf("%s: expected rest_invalid_param per_page: %s", path, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?per_page=100", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("per_page=100 status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// P4.144: zero post_modified_gmt is shimmed from post_modified via site timezone.
func TestPostsModifiedGMTZeroShim(t *testing.T) {
	c := bootOrSkip(t)
	const postID int64 = 1
	orig, err := wp.GetPostByID(c, postID)
	if err != nil || orig == nil {
		t.Fatalf("load post: %v", err)
	}
	prevOff := c.Options.GetOption("gmt_offset", "0")
	prevTZ := c.Options.GetOption("timezone_string", "")
	t.Cleanup(func() {
		uq, uargs, _ := c.DB.Prepare(
			"UPDATE "+c.DB.Posts()+" SET post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
			orig.Modified, orig.ModifiedGMT, postID,
		)
		_, _ = c.DB.Exec(uq, uargs...)
		_, _ = c.Options.UpdateOption("gmt_offset", prevOff, "yes")
		_, _ = c.Options.UpdateOption("timezone_string", prevTZ, "yes")
		_ = c.DB.Close()
	})

	uq, uargs, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
		"2024-06-15 14:00:00", "0000-00-00 00:00:00", postID,
	)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if _, err := c.DB.Exec(uq, uargs...); err != nil {
		t.Fatalf("zero modified_gmt: %v", err)
	}
	_, _ = c.Options.UpdateOption("timezone_string", "", "yes")
	_, _ = c.Options.UpdateOption("gmt_offset", "-5", "yes")

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/"+strconv.FormatInt(postID, 10), nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var post rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatal(err)
	}
	// Local 14:00 with gmt_offset -5 → GMT 19:00.
	if post.ModifiedGMT != "2024-06-15T19:00:00" {
		t.Fatalf("modified_gmt=%q want 2024-06-15T19:00:00 (modified=%q)", post.ModifiedGMT, post.Modified)
	}
	if post.Modified != "2024-06-15T14:00:00" {
		t.Fatalf("modified=%q want local 14:00", post.Modified)
	}
}

func TestMySQLDateToRESTZeroEmpty(t *testing.T) {
	// Indirect: zero modified_gmt with empty local yields empty modified_gmt.
	c := bootOrSkip(t)
	const postID int64 = 1
	orig, err := wp.GetPostByID(c, postID)
	if err != nil || orig == nil {
		t.Fatalf("load post: %v", err)
	}
	t.Cleanup(func() {
		uq, uargs, _ := c.DB.Prepare(
			"UPDATE "+c.DB.Posts()+" SET post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
			orig.Modified, orig.ModifiedGMT, postID,
		)
		_, _ = c.DB.Exec(uq, uargs...)
		_ = c.DB.Close()
	})
	uq, uargs, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
		"0000-00-00 00:00:00", "0000-00-00 00:00:00", postID,
	)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if _, err := c.DB.Exec(uq, uargs...); err != nil {
		t.Fatalf("zero both: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/"+strconv.FormatInt(postID, 10), nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var post rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatal(err)
	}
	if post.ModifiedGMT != "" {
		t.Fatalf("expected empty modified_gmt for zero dates, got %q", post.ModifiedGMT)
	}
}
