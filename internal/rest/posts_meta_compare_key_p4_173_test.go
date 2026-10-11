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

// P4.173: meta_compare_key enum expands to NOT LIKE / != / EXISTS / NOT EXISTS.
func TestPostsMetaCompareKeyEnumP4173(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_compare_key=BETWEEN&per_page=1", nil)
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
	for _, want := range []string{"NOT LIKE", "!=", "EXISTS", "NOT EXISTS"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in enum message %q", want, msg)
		}
	}

	for _, okVal := range []string{"NOT LIKE", "!=", "EXISTS", "NOT EXISTS"} {
		okReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_compare_key="+strings.ReplaceAll(okVal, " ", "%20")+"&per_page=1", nil)
		okRec := httptest.NewRecorder()
		okRC := c.ForRequest(okRec, okReq)
		(&rest.Server{}).Handle(okRec, okReq, okRC)
		if okRec.Code == http.StatusBadRequest && strings.Contains(okRec.Body.String(), "rest_not_in_enum") {
			t.Fatalf("%s should be accepted: %s", okVal, okRec.Body.String())
		}
	}
}

// P4.173: meta_compare_key=NOT LIKE + EXISTS excludes posts with matching key substring.
func TestPostsMetaCompareKeyNotLikeFilterP4173(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	createBody := `{"title":"Meta Compare Key NOT LIKE","content":"Body","status":"publish","slug":"gopress-meta-compare-key-not-like","meta":{"footnotes":"hello"}}`
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

	missReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=note&meta_compare_key=NOT%20LIKE&meta_compare=EXISTS&per_page=100&slug=gopress-meta-compare-key-not-like", nil)
	missRec := httptest.NewRecorder()
	mc := c.ForRequest(missRec, missReq)
	(&rest.Server{}).Handle(missRec, missReq, mc)
	if missRec.Code != http.StatusOK {
		t.Fatalf("miss status=%d body=%s", missRec.Code, missRec.Body.String())
	}
	var misses []rest.Post
	if err := json.Unmarshal(missRec.Body.Bytes(), &misses); err != nil {
		t.Fatal(err)
	}
	for _, p := range misses {
		if p.ID == created.ID {
			t.Fatalf("NOT LIKE should exclude post with footnotes key matching note substring")
		}
	}
}
