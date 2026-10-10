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

func TestUsersMeRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users/me", nil)
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
	if body["code"] != "rest_not_logged_in" {
		t.Fatalf("code=%v", body["code"])
	}
}

func TestUsersMeAdmin(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users/me", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Allow"), "POST") {
		t.Fatalf("Allow=%q", rec.Header().Get("Allow"))
	}
	var me rest.User
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.ID != 1 || me.Slug != "admin" || me.Name == "" {
		t.Fatalf("unexpected me: %+v", me)
	}
	if me.Email == "" || me.Username != "admin" {
		t.Fatalf("expected email/username: %+v", me)
	}
	if me.Link == "" {
		t.Fatal("expected link")
	}
	if me.Links == nil || me.AvatarURLs == nil || me.AvatarURLs["96"] == "" {
		t.Fatalf("expected _links/avatar_urls: %+v", me)
	}
	if len(me.Capabilities) == 0 || !me.Capabilities["manage_options"] {
		t.Fatalf("expected capabilities: %+v", me.Capabilities)
	}
	if me.RegisteredDate == "" {
		t.Fatal("expected registered_date")
	}
}

func TestUsersMeUpdate(t *testing.T) {
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
	body := `{"name":"Admin Me Updated","description":"me-desc","first_name":"Ada"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/users/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var me rest.User
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.Name != "Admin Me Updated" || me.Description != "me-desc" {
		t.Fatalf("update not reflected: %+v", me)
	}
	if me.FirstName != "Ada" {
		t.Fatalf("first_name=%q", me.FirstName)
	}
	// Restore fixture user fields before package teardown (shared MySQL).
	n, d := "Admin User", ""
	_ = c.UpdateUser(wp.UserUpdate{ID: user.ID, DisplayName: &n, Description: &d})
	if c.Meta != nil {
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, user.ID, "first_name", "", "")
	}
}

func TestUsersCollectionListUsers(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users?per_page=10", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-WP-Total") == "" {
		t.Fatal("expected X-WP-Total")
	}
	var users []rest.User
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatal(err)
	}
	if len(users) < 1 {
		t.Fatal("expected at least one user")
	}
	foundAdmin := false
	for _, u := range users {
		if u.Slug == "admin" {
			foundAdmin = true
			if len(u.Roles) == 0 {
				t.Fatalf("list_users should include roles: %+v", u)
			}
		}
		if u.Link == "" || u.Links == nil {
			t.Fatalf("expected link/_links: %+v", u)
		}
	}
	if !foundAdmin {
		t.Fatalf("expected admin in collection: %+v", users)
	}
}

func TestUsersCollectionAnonPublished(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users?per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var users []rest.User
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatal(err)
	}
	if len(users) < 1 {
		t.Fatal("expected published authors")
	}
	for _, u := range users {
		if u.Email != "" || u.Username != "" {
			t.Fatalf("anon view must omit email/username: %+v", u)
		}
	}
}
