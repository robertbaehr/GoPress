package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestCreateCategoryRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	body := `{"name":"Nope Cat"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/categories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateCategoryAndTagAsEditor(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	user, err := c.GetUserBy("login", "editor")
	if err != nil || user == nil {
		t.Fatalf("editor: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	createBody := `{"name":"GoPress Write Cat","description":"from REST","slug":"gopress-write-cat"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/categories", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create category status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cat rest.Term
	if err := json.Unmarshal(rec.Body.Bytes(), &cat); err != nil {
		t.Fatal(err)
	}
	if cat.ID <= 0 || cat.Slug != "gopress-write-cat" || cat.Taxonomy != "category" || cat.Name != "GoPress Write Cat" {
		t.Fatalf("created category: %+v", cat)
	}
	catID := cat.ID
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.TermRelationships()+" WHERE term_taxonomy_id IN (SELECT term_taxonomy_id FROM "+c.DB.TermTaxonomy()+" WHERE term_id = %d)", catID)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
		q, args, err = c.DB.Prepare("DELETE FROM "+c.DB.TermTaxonomy()+" WHERE term_id = %d", catID)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
		q, args, err = c.DB.Prepare("DELETE FROM "+c.DB.Terms()+" WHERE term_id = %d", catID)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})

	tagBody := `{"name":"GoPress Write Tag","slug":"gopress-write-tag"}`
	req = httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/tags", bytes.NewBufferString(tagBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create tag status=%d body=%s", rec.Code, rec.Body.String())
	}
	var tag rest.Term
	if err := json.Unmarshal(rec.Body.Bytes(), &tag); err != nil {
		t.Fatal(err)
	}
	if tag.ID <= 0 || tag.Slug != "gopress-write-tag" || tag.Taxonomy != "post_tag" {
		t.Fatalf("created tag: %+v", tag)
	}
	tagID := tag.ID
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.TermTaxonomy()+" WHERE term_id = %d", tagID)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
		q, args, err = c.DB.Prepare("DELETE FROM "+c.DB.Terms()+" WHERE term_id = %d", tagID)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})

	// Subscriber cannot create.
	sub, err := c.GetUserBy("login", "subscriber")
	if err != nil || sub == nil {
		t.Fatalf("subscriber: %v", err)
	}
	subSet, err := c.SetAuthCookie(sub, false, false, 0, "")
	if err != nil {
		t.Fatalf("sub cookie: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/categories", bytes.NewBufferString(`{"name":"No Cap Cat"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: subSet.LoggedInName, Value: subSet.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("subscriber status=%d body=%s", rec.Code, rec.Body.String())
	}
}
