package wp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestDieDefaultHTML(t *testing.T) {
	res := wp.Die(nil, nil, "Nope", "", wp.DieArgs{Response: 403})
	if res.Handler != wp.DieHandlerDefault {
		t.Fatalf("handler=%q", res.Handler)
	}
	if res.Status != 403 {
		t.Fatalf("status=%d", res.Status)
	}
	body := string(res.Body)
	if !strings.Contains(body, "wp-die-message") || !strings.Contains(body, "Nope") {
		t.Fatalf("body=%q", body)
	}
	if !strings.Contains(body, "WordPress \u203a Error") {
		t.Fatalf("title missing: %q", body)
	}
}

func TestDieJSONAndAjax(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")
	res := wp.Die(nil, req, "bad", "", 400)
	if res.Handler != wp.DieHandlerJSON {
		t.Fatalf("handler=%q", res.Handler)
	}
	var payload map[string]any
	if err := json.Unmarshal(res.Body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["code"] != "wp_die" || payload["message"] != "bad" {
		t.Fatalf("payload=%v", payload)
	}
	data, _ := payload["data"].(map[string]any)
	if int(data["status"].(float64)) != 400 {
		t.Fatalf("status data=%v", data)
	}

	c := wp.NewContext(nil)
	c.Scope.SetAjax(true)
	ajax := wp.Die(c, nil, "0", "", nil)
	if ajax.Handler != wp.DieHandlerAjax || string(ajax.Body) != "0" || ajax.Status != 200 {
		t.Fatalf("ajax=%+v body=%q", ajax, ajax.Body)
	}
}

func TestDieWPErrorAndFilter(t *testing.T) {
	err := wp.NewWPError("rest_forbidden", "Nope", map[string]any{"status": 403, "title": "Forbidden"})
	err.Add("rest_forbidden", "Also", nil)
	res := wp.Die(nil, nil, err, "", nil)
	if res.Status != 403 || res.Code != "rest_forbidden" {
		t.Fatalf("status=%d code=%q", res.Status, res.Code)
	}
	if res.Title != "Forbidden" {
		t.Fatalf("title=%q", res.Title)
	}
	if !strings.Contains(string(res.Body), "Also") {
		t.Fatalf("additional error missing: %q", res.Body)
	}

	c := wp.NewContext(nil)
	c.Hooks = wp.NewHooks()
	c.Hooks.AddFilter("wp_die_handler", func(value any, _ ...any) any { return "json" }, 10, 1)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	forced := wp.Die(c, req, "x", "", 500)
	if forced.Handler != wp.DieHandlerJSON {
		t.Fatalf("filter handler=%q", forced.Handler)
	}
}

func TestWriteDie(t *testing.T) {
	rec := httptest.NewRecorder()
	wp.WriteDie(rec, wp.Die(nil, nil, "x", "T", 418))
	if rec.Code != 418 {
		t.Fatalf("code=%d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("ct=%q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Cache-Control") == "" {
		t.Fatal("expected nocache headers")
	}
}
