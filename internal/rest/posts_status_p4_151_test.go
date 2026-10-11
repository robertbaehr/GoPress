package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.151: status query wraps forbidden as rest_invalid_param; editors get enum validate.
func TestPostsStatusQueryInvalidParamP4151(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?status=notastatus", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("anon notastatus status=%d body=%s", rec.Code, rec.Body.String())
	}
	var anon map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &anon); err != nil {
		t.Fatal(err)
	}
	if anon["code"] != "rest_invalid_param" {
		t.Fatalf("code=%v", anon["code"])
	}
	if !strings.Contains(rec.Body.String(), `"rest_forbidden_status"`) {
		t.Fatalf("expected forbidden details: %s", rec.Body.String())
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

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?status=notastatus&per_page=1", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("editor notastatus status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "rest_not_in_enum") {
		t.Fatalf("expected rest_not_in_enum: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "status[0] is not one of") {
		t.Fatalf("expected enum message: %s", rec.Body.String())
	}
}
