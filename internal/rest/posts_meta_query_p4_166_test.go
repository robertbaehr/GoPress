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

// P4.166: meta_key[]= → rest_invalid_param / rest_invalid_type.
func TestPostsMetaKeyArrayFormInvalidP4166(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	for _, path := range []string{
		"/wp-json/wp/v2/posts?meta_key[]=footnotes&per_page=1",
		"/wp-json/wp/v2/posts?meta_value[]=x&per_page=1",
		"/wp-json/wp/v2/posts?meta_compare[]=EXISTS&per_page=1",
		"/wp-json/wp/v2/posts?meta_query[0][key]=footnotes&per_page=1",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, rc)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["code"] != "rest_invalid_param" {
			t.Fatalf("%s code=%v", path, body["code"])
		}
	}
}

// P4.166: meta_compare=bogus → rest_not_in_enum.
func TestPostsMetaCompareInvalidEnumP4166(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_compare=BETWEEN&per_page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
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
	if mc["code"] != "rest_not_in_enum" {
		t.Fatalf("detail=%v", mc)
	}
}

// P4.166: scalar meta_key+meta_value filters via EXISTS.
func TestPostsMetaKeyValueFilterP4166(t *testing.T) {
	c := bootOrSkip(t)
	// Close after Cleanup so DeletePost still has a live DB handle.
	t.Cleanup(func() { _ = c.DB.Close() })

	createBody := `{"title":"Meta Query Target","content":"Body","status":"publish","slug":"gopress-meta-query-target","meta":{"footnotes":"mq-marker-p4166"}}`
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

	listReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=mq-marker-p4166&per_page=10", nil)
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
		t.Fatalf("expected created post in meta filter results: %+v", posts)
	}

	missReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=no-such-marker&per_page=10", nil)
	missRec := httptest.NewRecorder()
	mc := c.ForRequest(missRec, missReq)
	(&rest.Server{}).Handle(missRec, missReq, mc)
	if missRec.Code != http.StatusOK {
		t.Fatalf("miss status=%d", missRec.Code)
	}
	var miss []rest.Post
	if err := json.Unmarshal(missRec.Body.Bytes(), &miss); err != nil {
		t.Fatal(err)
	}
	for _, p := range miss {
		if p.ID == created.ID {
			t.Fatalf("unexpected match for wrong meta_value")
		}
	}
}
