package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestCreateUpdateDeleteMenuItem(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	body := `{"title":"E2E Menu Link","url":"https://example.com/e2e","menus":37,"status":"publish","type":"custom"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/menu-items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.ClassicMenuItem
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID <= 0 || created.Title != "E2E Menu Link" || created.URL != "https://example.com/e2e" {
		t.Fatalf("created=%+v", created)
	}
	if created.Menus != 37 {
		t.Fatalf("menus=%d want 37", created.Menus)
	}
	id := created.ID
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", id)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
		dq, dargs, err := c.DB.Prepare("DELETE FROM "+c.DB.TermRelationships()+" WHERE object_id = %d", id)
		if err == nil {
			_, _ = c.DB.Exec(dq, dargs...)
		}
	})

	upd := `{"title":"E2E Menu Link Updated","url":"https://example.com/updated","parent":447}`
	req = httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/menu-items/"+strconv.FormatInt(id, 10), bytes.NewBufferString(upd))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated rest.ClassicMenuItem
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Title != "E2E Menu Link Updated" || updated.URL != "https://example.com/updated" || updated.Parent != 447 {
		t.Fatalf("updated=%+v", updated)
	}

	req = httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/menu-items/"+strconv.FormatInt(id, 10)+"?force=true", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	var del map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &del); err != nil {
		t.Fatal(err)
	}
	if del["deleted"] != true {
		t.Fatalf("delete resp=%#v", del)
	}
}

func TestCreateMenuItemRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/menu-items", bytes.NewBufferString(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDeleteMenuItemRequiresForce(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	body := `{"title":"Temp Delete","url":"/tmp","menus":37}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/menu-items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.ClassicMenuItem
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id := created.ID
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", id)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})

	req = httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/menu-items/"+strconv.FormatInt(id, 10), nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 without force, got %d body=%s", rec.Code, rec.Body.String())
	}
}
