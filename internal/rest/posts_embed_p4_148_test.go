package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestPostsEmbedReplies(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1?_embed=replies", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	embedded, ok := body["_embedded"].(map[string]any)
	if !ok {
		t.Fatalf("expected _embedded: keys=%v", mapKeys(body))
	}
	replies, ok := embedded["replies"].([]any)
	if !ok || len(replies) != 1 {
		t.Fatalf("expected replies[[…]]: %#v", embedded["replies"])
	}
	list, ok := replies[0].([]any)
	if !ok || len(list) == 0 {
		t.Fatalf("expected non-empty comment list: %#v", replies[0])
	}
	cm, ok := list[0].(map[string]any)
	if !ok {
		t.Fatalf("comment type: %#v", list[0])
	}
	for _, key := range []string{"id", "author_name", "content", "link", "type", "_links"} {
		if _, has := cm[key]; !has {
			t.Fatalf("embed comment missing %s: keys=%v", key, mapKeys(cm))
		}
	}
	for _, omit := range []string{"post", "status", "date_gmt", "meta"} {
		if _, has := cm[omit]; has {
			t.Fatalf("embed comment should omit %s: keys=%v", omit, mapKeys(cm))
		}
	}
	if _, ok := embedded["author"]; ok {
		t.Fatalf("replies-only filter should omit author: keys=%v", mapKeys(embedded))
	}
}

func TestPostsEmbedAllIncludesReplies(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1?_embed=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	embedded, ok := body["_embedded"].(map[string]any)
	if !ok {
		t.Fatalf("expected _embedded: %#v", body["_embedded"])
	}
	if _, ok := embedded["replies"]; !ok {
		t.Fatalf("full _embed should include replies: keys=%v", mapKeys(embedded))
	}
	if _, ok := embedded["author"]; !ok {
		t.Fatalf("full _embed should include author: keys=%v", mapKeys(embedded))
	}
}

func TestPagesEmbedUpParent(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages/438?_embed=up", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	embedded, ok := body["_embedded"].(map[string]any)
	if !ok {
		t.Fatalf("expected _embedded: keys=%v", mapKeys(body))
	}
	up, ok := embedded["up"].([]any)
	if !ok || len(up) != 1 {
		t.Fatalf("expected up[parent]: %#v", embedded["up"])
	}
	parent, ok := up[0].(map[string]any)
	if !ok {
		t.Fatalf("parent type: %#v", up[0])
	}
	if id, _ := parent["id"].(float64); int64(id) != 437 {
		t.Fatalf("expected parent id 437, got %#v", parent["id"])
	}
	if _, has := parent["content"]; has {
		t.Fatalf("embed parent should omit content: keys=%v", mapKeys(parent))
	}
}
