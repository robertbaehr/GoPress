package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestUserEditScreen(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/user-edit.php?user_id=1", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="your-profile"`,
		`name="display_name"`,
		`name="email"`,
		`name="description"`,
		`user-edit-php`,
		`GoPress user-edit stub`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}

	// Users list links to user-edit.
	req = httptest.NewRequest(http.MethodGet, "/wp-admin/users.php", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `/wp-admin/user-edit.php?user_id=`) {
		t.Fatalf("users list missing edit links: %q", rec.Body.String())
	}
}

func TestUserEditUpdate(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	original := user.DisplayName
	t.Cleanup(func() {
		_ = c.UpdateDisplayName(user.ID, original)
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/user-edit.php?user_id="+strconv.FormatInt(user.ID, 10), nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status=%d", rec.Code)
	}
	body := rec.Body.String()
	nonceStart := strings.Index(body, `name="_wpnonce" value="`)
	if nonceStart < 0 {
		t.Fatal("nonce not found")
	}
	nonceStart += len(`name="_wpnonce" value="`)
	nonceEnd := strings.Index(body[nonceStart:], `"`)
	nonce := body[nonceStart : nonceStart+nonceEnd]

	form := url.Values{}
	form.Set("_wpnonce", nonce)
	form.Set("user_id", strconv.FormatInt(user.ID, 10))
	form.Set("display_name", "Admin Edited")
	form.Set("email", user.Email)
	form.Set("url", "https://example.com/admin-edited")
	form.Set("description", "Edited via user-edit")

	req = httptest.NewRequest(http.MethodPost, "/wp-admin/user-edit.php?user_id="+strconv.FormatInt(user.ID, 10), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "User updated.") {
		t.Fatalf("missing updated notice: %q", rec.Body.String())
	}
	reloaded, err := c.GetUserBy("id", user.ID)
	if err != nil || reloaded == nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.DisplayName != "Admin Edited" {
		t.Fatalf("display_name=%q", reloaded.DisplayName)
	}
}
