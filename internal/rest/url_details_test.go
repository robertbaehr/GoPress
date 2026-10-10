package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestURLDetailsRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-block-editor/v1/url-details?url=https://example.com", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["code"] != "rest_cannot_view_url_details" {
		t.Fatalf("code=%v", body["code"])
	}
}

func TestURLDetailsRemoteStaysEmpty(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-block-editor/v1/url-details?url=https://example.com/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var details rest.URLDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &details); err != nil {
		t.Fatalf("json: %v", err)
	}
	// Remote fetch omitted; fields stay empty.
	if details.Title != "" || details.Icon != "" || details.Description != "" || details.Image != "" {
		t.Fatalf("unexpected details: %+v", details)
	}
}

func TestURLDetailsSameSiteEnriched(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-block-editor/v1/url-details?url=http://127.0.0.1:8081/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var details rest.URLDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &details); err != nil {
		t.Fatalf("json: %v", err)
	}
	if details.Title == "" {
		t.Fatalf("expected site title, got %+v", details)
	}
	if details.Icon == "" {
		t.Fatalf("expected icon, got %+v", details)
	}
}

func TestURLDetailsLocalPost(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-block-editor/v1/url-details?url=http://127.0.0.1:8081/?p=1", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var details rest.URLDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &details); err != nil {
		t.Fatalf("json: %v", err)
	}
	if !strings.Contains(strings.ToLower(details.Title), "hello") {
		t.Fatalf("expected Hello world title, got %+v", details)
	}
}

func TestURLDetailsInvalidURL(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-block-editor/v1/url-details?url=javascript:alert(1)", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["code"] != "rest_invalid_url" {
		t.Fatalf("code=%v", body["code"])
	}
}

func TestURLDetailsMissingURL(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-block-editor/v1/url-details", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestParseURLDetailsHTML(t *testing.T) {
	html := `<!doctype html><html><head>
<title> Example Title </title>
<link rel="icon" href="/favicon.ico">
<meta name="description" content="A short desc">
<meta property="og:image" content="https://cdn.example/img.png">
</head><body></body></html>`
	d := rest.ParseURLDetailsHTML(html, "https://example.com/")
	if d.Title != "Example Title" {
		t.Fatalf("title=%q", d.Title)
	}
	if d.Icon != "/favicon.ico" {
		t.Fatalf("icon=%q", d.Icon)
	}
	if d.Description != "A short desc" {
		t.Fatalf("desc=%q", d.Description)
	}
	if d.Image != "https://cdn.example/img.png" {
		t.Fatalf("image=%q", d.Image)
	}
}
