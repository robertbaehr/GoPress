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

// P4.172: meta_compare_key=FOO rejected; LIKE accepted as enum.
func TestPostsMetaCompareKeyEnumP4172(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_compare_key=FOO&per_page=1", nil)
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
	mc, _ := details["meta_compare_key"].(map[string]any)
	msg, _ := mc["message"].(string)
	if !strings.Contains(msg, "LIKE") || !strings.Contains(msg, "=") {
		t.Fatalf("expected compare_key enum message, got %q", msg)
	}

	okReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_compare_key=LIKE&per_page=1", nil)
	okRec := httptest.NewRecorder()
	okRC := c.ForRequest(okRec, okReq)
	(&rest.Server{}).Handle(okRec, okReq, okRC)
	if okRec.Code == http.StatusBadRequest && strings.Contains(okRec.Body.String(), "rest_not_in_enum") {
		t.Fatalf("LIKE should be accepted: %s", okRec.Body.String())
	}
}

// P4.172: meta_key substring + meta_compare_key=LIKE + EXISTS matches.
func TestPostsMetaCompareKeyLikeFilterP4172(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	createBody := `{"title":"Meta Compare Key LIKE","content":"Body","status":"publish","slug":"gopress-meta-compare-key-like","meta":{"footnotes":"hello"}}`
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

	hitReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=note&meta_compare_key=LIKE&meta_compare=EXISTS&per_page=10", nil)
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
		t.Fatalf("expected LIKE key match for created post: %+v", hits)
	}
}
