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

// P4.167: meta_compare=IN rejected; NOT LIKE accepted as enum (P4.169 added BETWEEN).
func TestPostsMetaCompareExpandedEnumP4167(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_compare=IN&per_page=1", nil)
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
	if !strings.Contains(msg, "NOT LIKE") || !strings.Contains(msg, ">=") {
		t.Fatalf("expected expanded enum message, got %q", msg)
	}
}

// P4.167: meta_key + meta_compare=NOT LIKE filters out matching values.
func TestPostsMetaKeyNotLikeFilterP4167(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	createBody := `{"title":"Meta Key NotLike","content":"Body","status":"publish","slug":"gopress-meta-key-notlike","meta":{"footnotes":"mq-marker-p4167-keep"}}`
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

	keepReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=absent-token&meta_compare=NOT%20LIKE&per_page=10", nil)
	keepRec := httptest.NewRecorder()
	kc := c.ForRequest(keepRec, keepReq)
	(&rest.Server{}).Handle(keepRec, keepReq, kc)
	if keepRec.Code != http.StatusOK {
		t.Fatalf("keep status=%d body=%s", keepRec.Code, keepRec.Body.String())
	}
	var keep []rest.Post
	if err := json.Unmarshal(keepRec.Body.Bytes(), &keep); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range keep {
		if p.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected post when NOT LIKE misses: %+v", keep)
	}

	dropReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=mq-marker-p4167&meta_compare=NOT%20LIKE&per_page=10", nil)
	dropRec := httptest.NewRecorder()
	dc := c.ForRequest(dropRec, dropReq)
	(&rest.Server{}).Handle(dropRec, dropReq, dc)
	if dropRec.Code != http.StatusOK {
		t.Fatalf("drop status=%d", dropRec.Code)
	}
	var drop []rest.Post
	if err := json.Unmarshal(dropRec.Body.Bytes(), &drop); err != nil {
		t.Fatal(err)
	}
	for _, p := range drop {
		if p.ID == created.ID {
			t.Fatalf("post should be excluded by NOT LIKE match")
		}
	}
}

// P4.167: meta_compare=>= with numeric meta_value.
func TestPostsMetaKeyGteFilterP4167(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	createBody := `{"title":"Meta Key Gte","content":"Body","status":"publish","slug":"gopress-meta-key-gte","meta":{"footnotes":"42"}}`
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

	listReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=40&meta_compare=%3E%3D&per_page=10", nil)
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
		t.Fatalf("expected created post for >= filter: %+v", posts)
	}
}
