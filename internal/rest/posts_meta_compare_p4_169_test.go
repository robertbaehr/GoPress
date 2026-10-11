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

// P4.169: meta_compare=XOR rejected; BETWEEN accepted as enum (P4.171 added IN).
func TestPostsMetaCompareBetweenEnumP4169(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_compare=XOR&per_page=1", nil)
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
	if !strings.Contains(msg, "BETWEEN") || !strings.Contains(msg, "NOT BETWEEN") {
		t.Fatalf("expected BETWEEN enum message, got %q", msg)
	}
}

// P4.169: meta_key + meta_value=low,high + meta_compare=BETWEEN matches range.
func TestPostsMetaCompareBetweenFilterP4169(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	createBody := `{"title":"Meta Compare Between","content":"Body","status":"publish","slug":"gopress-meta-compare-between","meta":{"footnotes":"50"}}`
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

	hitReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=40,60&meta_compare=BETWEEN&per_page=10", nil)
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
		t.Fatalf("expected BETWEEN match for created post: %+v", hits)
	}

	missReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=40,60&meta_compare=NOT%20BETWEEN&per_page=10", nil)
	missRec := httptest.NewRecorder()
	mc := c.ForRequest(missRec, missReq)
	(&rest.Server{}).Handle(missRec, missReq, mc)
	if missRec.Code != http.StatusOK {
		t.Fatalf("miss status=%d body=%s", missRec.Code, missRec.Body.String())
	}
	var kept []rest.Post
	if err := json.Unmarshal(missRec.Body.Bytes(), &kept); err != nil {
		t.Fatal(err)
	}
	for _, p := range kept {
		if p.ID == created.ID {
			t.Fatalf("expected NOT BETWEEN to exclude created post")
		}
	}
}

// P4.169: space-separated BETWEEN bounds accepted.
func TestPostsMetaCompareBetweenSpaceBoundsP4169(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	createBody := `{"title":"Meta Compare Between Space","content":"Body","status":"publish","slug":"gopress-meta-compare-between-space","meta":{"footnotes":"45"}}`
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

	// meta_value=40+60 → "40 60" after query decode (space-separated bounds).
	listReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=40+60&meta_compare=BETWEEN&per_page=10", nil)
	listRec := httptest.NewRecorder()
	lc := c.ForRequest(listRec, listReq)
	(&rest.Server{}).Handle(listRec, listReq, lc)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
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
		t.Fatalf("expected space BETWEEN match: %+v", posts)
	}
}
