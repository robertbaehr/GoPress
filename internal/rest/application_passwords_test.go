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

func TestApplicationPasswordsCRUDForMe(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	prev := c.Meta.GetMetadata(wp.MetaUser, user.ID, "_application_passwords", true)
	t.Cleanup(func() {
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, user.ID, "_application_passwords", prev, "")
	})
	_, _ = wp.DeleteAllApplicationPasswords(c, user.ID)

	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	// Create
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/users/me/application-passwords",
		strings.NewReader(`{"name":"GoPress CLI"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	uuid, _ := created["uuid"].(string)
	plain, _ := created["password"].(string)
	if uuid == "" || plain == "" {
		t.Fatalf("expected uuid+password: %+v", created)
	}
	if created["name"] != "GoPress CLI" {
		t.Fatalf("name=%v", created["name"])
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users/me/application-passwords", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range list {
		if item["uuid"] == uuid && item["name"] == "GoPress CLI" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created item missing from list=%v", list)
	}

	// Get item
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users/me/application-passwords/"+uuid, nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Rename
	req = httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/users/me/application-passwords/"+uuid,
		strings.NewReader(`{"name":"Renamed App"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Auth with new password still works (hash unchanged on rename)
	compact := strings.ReplaceAll(plain, " ", "")
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users/me", nil)
	req.SetBasicAuth("admin", compact)
	rec = httptest.NewRecorder()
	c2 := bootOrSkip(t)
	t.Cleanup(func() { _ = c2.DB.Close() })
	c2 = c2.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c2)
	if rec.Code != http.StatusOK {
		t.Fatalf("basic auth after rename status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Delete item
	req = httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/users/me/application-passwords/"+uuid, nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	var del map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &del); err != nil {
		t.Fatal(err)
	}
	if del["deleted"] != true {
		t.Fatalf("delete resp=%v", del)
	}
}

func TestApplicationPasswordsRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users/me/application-passwords", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
