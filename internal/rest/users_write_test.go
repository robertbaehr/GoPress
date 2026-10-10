package rest_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestCreateUpdateUser(t *testing.T) {
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

	login := fmt.Sprintf("gopress_user_%d", user.ID*1000+42)
	email := login + "@example.com"
	createBody := fmt.Sprintf(
		`{"username":%q,"email":%q,"password":"SecretPass1!","name":"GoPress User","roles":["author"],"description":"Created via REST"}`,
		login, email,
	)
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/users", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.User
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID <= 0 || created.Username != login || created.Email != email {
		t.Fatalf("created: %+v", created)
	}
	if len(created.Roles) == 0 || created.Roles[0] != "author" {
		t.Fatalf("roles: %+v", created.Roles)
	}
	id := created.ID
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Users()+" WHERE ID = %d", id)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
		q, args, err = c.DB.Prepare("DELETE FROM "+c.DB.UserMeta()+" WHERE user_id = %d", id)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})

	updateBody := `{"name":"GoPress User Updated","description":"Updated bio"}`
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/wp-json/wp/v2/users/%d", id), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated rest.User
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "GoPress User Updated" || updated.Description != "Updated bio" {
		t.Fatalf("updated: %+v", updated)
	}

	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wp-json/wp/v2/users/%d?context=edit", id), nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateUserRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/users", bytes.NewBufferString(
		`{"username":"nope","email":"nope@example.com","password":"x"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
