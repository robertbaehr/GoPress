package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestSettingsRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/settings", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "rest_forbidden" {
		t.Fatalf("code=%v", body["code"])
	}
}

func TestSettingsAdmin(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/settings", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var settings map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"title", "description", "url", "email", "timezone", "date_format", "time_format",
		"start_of_week", "language", "use_smilies", "default_category", "default_post_format",
		"posts_per_page", "show_on_front", "page_on_front", "page_for_posts",
		"page_for_privacy_policy", "default_ping_status", "default_comment_status",
		"site_logo", "site_icon",
		"blogname", "blogdescription", "siteurl", "home",
	} {
		if _, ok := settings[key]; !ok {
			t.Fatalf("missing %q in %v", key, settings)
		}
	}
	if settings["blogname"] != "GoPress Fixture" || settings["title"] != "GoPress Fixture" {
		t.Fatalf("blogname/title=%v/%v", settings["blogname"], settings["title"])
	}
	if settings["email"] == "" {
		t.Fatalf("expected admin email, got %#v", settings["email"])
	}
}

func TestSettingsUpdateRequiresManageOptions(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/settings", strings.NewReader(`{"title":"Nope"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSettingsUpdateAdmin(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	origName := c.Options.GetOption("blogname", "")
	origPPP := c.Options.GetOption("posts_per_page", 10)
	defer func() {
		_, _ = c.Options.UpdateOption("blogname", origName, "")
		_, _ = c.Options.UpdateOption("posts_per_page", origPPP, "")
	}()

	origEmail := c.Options.GetOption("admin_email", "")
	origCat := c.Options.GetOption("default_category", 1)
	defer func() {
		_, _ = c.Options.UpdateOption("admin_email", origEmail, "")
		_, _ = c.Options.UpdateOption("default_category", origCat, "")
	}()

	body := `{"title":"GoPress Settings Write","posts_per_page":7,"email":"settings@example.com","default_category":2}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var settings map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatal(err)
	}
	if settings["title"] != "GoPress Settings Write" || settings["blogname"] != "GoPress Settings Write" {
		t.Fatalf("title/blogname=%v/%v", settings["title"], settings["blogname"])
	}
	if int(settings["posts_per_page"].(float64)) != 7 {
		t.Fatalf("posts_per_page=%v", settings["posts_per_page"])
	}
	if settings["email"] != "settings@example.com" {
		t.Fatalf("email=%v", settings["email"])
	}
	if int(settings["default_category"].(float64)) != 2 {
		t.Fatalf("default_category=%v", settings["default_category"])
	}
	if got := c.Options.GetOption("blogname", ""); got != "GoPress Settings Write" {
		t.Fatalf("option blogname=%v", got)
	}
}
