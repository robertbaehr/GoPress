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

// P4.171: meta_compare=FOO rejected; IN / NOT IN accepted as enum.
func TestPostsMetaCompareInEnumP4171(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_compare=FOO&per_page=1", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body["data"].(map[string]any)
	details, _ := data["details"].(map[string]any)
	mc, _ := details["meta_compare"].(map[string]any)
	msg, _ := mc["message"].(string)
	if !strings.Contains(msg, "IN") || !strings.Contains(msg, "NOT IN") {
		t.Fatalf("expected IN enum message, got %q", msg)
	}

	okReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_compare=IN&per_page=1", nil)
	okRec := httptest.NewRecorder()
	okRC := c.ForRequest(okRec, okReq)
	(&rest.Server{}).Handle(okRec, okReq, okRC)
	if okRec.Code == http.StatusBadRequest && strings.Contains(okRec.Body.String(), "rest_not_in_enum") {
		t.Fatalf("IN should be accepted: %s", okRec.Body.String())
	}
}

// P4.171: meta_key + meta_value=a,b + meta_compare=IN matches list.
func TestPostsMetaCompareInFilterP4171(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	createBody := `{"title":"Meta Compare IN","content":"Body","status":"publish","slug":"gopress-meta-compare-in","meta":{"footnotes":"alpha"}}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if created.ID > 0 {
			if err := wp.DeletePost(c, created.ID); err != nil {
				t.Errorf("cleanup DeletePost(%d): %v", created.ID, err)
			}
		}
	})

	hitReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=alpha,beta&meta_compare=IN&per_page=10", nil)
	hitRec := httptest.NewRecorder()
	hc := c.ForRequest(hitRec, hitReq)
	(&rest.Server{}).Handle(hitRec, hitReq, hc)
	if hitRec.Code != http.StatusOK {
		t.Fatalf("hit status=%d body=%s", hitRec.Code, hitRec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(hitRec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range hits {
		if p.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected IN match for created post: %+v", hits)
	}

	missReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=alpha,beta&meta_compare=NOT%20IN&per_page=10", nil)
	missRec := httptest.NewRecorder()
	mc := c.ForRequest(missRec, missReq)
	(&rest.Server{}).Handle(missRec, missReq, mc)
	if missRec.Code != http.StatusOK {
		t.Fatalf("NOT IN status=%d body=%s", missRec.Code, missRec.Body.String())
	}
	var misses []rest.Post
	if err := json.Unmarshal(missRec.Body.Bytes(), &misses); err != nil {
		t.Fatal(err)
	}
	for _, p := range misses {
		if p.ID == created.ID {
			t.Fatalf("expected NOT IN to exclude created post")
		}
	}
}

// P4.171: space-separated IN list accepted.
func TestPostsMetaCompareInSpaceSeparatedP4171(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	createBody := `{"title":"Meta Compare IN Space","content":"Body","status":"publish","slug":"gopress-meta-compare-in-space","meta":{"footnotes":"gamma"}}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if created.ID > 0 {
			if err := wp.DeletePost(c, created.ID); err != nil {
				t.Errorf("cleanup DeletePost(%d): %v", created.ID, err)
			}
		}
	})

	listReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=gamma+delta&meta_compare=IN&per_page=10", nil)
	listRec := httptest.NewRecorder()
	lc := c.ForRequest(listRec, listReq)
	(&rest.Server{}).Handle(listRec, listReq, lc)
	if listRec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	var posts []rest.Post
	if err := json.Unmarshal(listRec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range posts {
		if p.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected space IN match: %+v", posts)
	}
}
