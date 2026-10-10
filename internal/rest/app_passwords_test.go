package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestUsersMeViaApplicationPassword(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	plain := "restAPPPassTEST1234abcdEF"
	hash := wp.FastHash(plain)
	prev := c.Meta.GetMetadata(wp.MetaUser, user.ID, "_application_passwords", true)
	if err := wp.SetApplicationPasswordsForTest(c, user.ID, "", "REST Test", hash); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, user.ID, "_application_passwords", prev, "")
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users/me", nil)
	req.SetBasicAuth("admin", plain)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var me rest.User
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.ID != 1 || me.Slug != "admin" {
		t.Fatalf("unexpected me: %+v", me)
	}
}

func TestUsersMeRejectsBadApplicationPassword(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users/me", nil)
	req.SetBasicAuth("admin", "not-a-valid-application-password")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
