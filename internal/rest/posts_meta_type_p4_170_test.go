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

// P4.170: meta_type[]= → rest_invalid_param / rest_invalid_type.
func TestPostsMetaTypeArrayFormInvalidP4170(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_type[]=NUMERIC&per_page=1", nil)
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
	if body["code"] != "rest_invalid_param" {
		t.Fatalf("code=%v body=%s", body["code"], rec.Body.String())
	}
}

// P4.170: meta_type=bogus → rest_not_in_enum.
func TestPostsMetaTypeEnumInvalidP4170(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_type=FLOAT&per_page=1", nil)
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
	mt, _ := details["meta_type"].(map[string]any)
	if mt["code"] != "rest_not_in_enum" {
		t.Fatalf("expected rest_not_in_enum, got %#v body=%s", mt, rec.Body.String())
	}
}

// P4.170: meta_type=NUMERIC + meta_compare=>= casts for numeric compare.
func TestPostsMetaTypeNumericGteP4170(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	createBody := `{"title":"Meta Type Numeric","content":"Body","status":"publish","slug":"gopress-meta-type-numeric","meta":{"footnotes":"42"}}`
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

	listReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_key=footnotes&meta_value=40&meta_compare=%3E%3D&meta_type=NUMERIC&per_page=10", nil)
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
		t.Fatalf("expected created post for NUMERIC >= filter: %+v", posts)
	}
}
