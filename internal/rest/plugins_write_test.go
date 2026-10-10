package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestPluginsActivateDeactivateViaREST(t *testing.T) {
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
	srv := &rest.Server{}

	// Ensure inactive first.
	_ = plugins.DeactivatePlugin(c, "hello.php")

	put := func(status string) rest.PluginREST {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/wp-json/wp/v2/plugins/hello", bytes.NewBufferString(`{"status":"`+status+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
		rec := httptest.NewRecorder()
		c2 := c.ForRequest(rec, req)
		srv.Handle(rec, req, c2)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var p rest.PluginREST
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatalf("json: %v", err)
		}
		return p
	}

	active := put("active")
	if active.Status != "active" {
		t.Fatalf("activate: %+v", active)
	}
	if !plugins.IsPluginActive(c, "hello.php") {
		t.Fatal("expected hello.php active in options")
	}

	inactive := put("inactive")
	if inactive.Status != "inactive" {
		t.Fatalf("deactivate: %+v", inactive)
	}
	if plugins.IsPluginActive(c, "hello.php") {
		t.Fatal("expected hello.php inactive")
	}
}

func TestPluginsUpdateRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodPut, "/wp-json/wp/v2/plugins/hello", bytes.NewBufferString(`{"status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
