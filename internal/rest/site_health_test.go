package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestSiteHealthRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-site-health/v1/tests/https-status", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSiteHealthTestStub(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-site-health/v1/tests/background-updates", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got rest.SiteHealthTest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got.Status != "good" || got.Test != "background_updates" {
		t.Fatalf("unexpected: %+v", got)
	}
	if got.Badge["label"] != "Security" {
		t.Fatalf("badge: %+v", got.Badge)
	}
}

func TestSiteHealthAllKnownTests(t *testing.T) {
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

	for _, name := range []string{
		"background-updates",
		"loopback-requests",
		"https-status",
		"dotorg-communication",
		"authorization-header",
		"page-cache",
	} {
		req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-site-health/v1/tests/"+name, nil)
		req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, rc)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", name, rec.Code, rec.Body.String())
		}
		var got rest.SiteHealthTest
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s json: %v", name, err)
		}
		if got.Status != "good" || got.Test == "" || got.Label == "" {
			t.Fatalf("%s unexpected: %+v", name, got)
		}
	}
}

func TestSiteHealthTestsCollection(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-site-health/v1/tests", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got []rest.SiteHealthTest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(got) < 6 {
		t.Fatalf("expected >=6 tests, got %d", len(got))
	}
}

func TestSiteHealthUnknownTest404(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-site-health/v1/tests/not-a-real-test", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSiteHealthDirectorySizes(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp-site-health/v1/directory-sizes", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	total, ok := got["total_size"].(map[string]any)
	if !ok || total["debug"] == nil {
		t.Fatalf("expected total_size.debug, got %#v", got["total_size"])
	}
}
