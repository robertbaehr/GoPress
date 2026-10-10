package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestBatchRejectsGET(t *testing.T) {
	c := bootOrSkip(t)
	srv := &rest.Server{}
	req := httptest.NewRequest(http.MethodGet, "/wp-json/batch/v1", nil)
	rec := httptest.NewRecorder()
	srv.Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBatchRequiresRequests(t *testing.T) {
	c := bootOrSkip(t)
	srv := &rest.Server{}
	req := httptest.NewRequest(http.MethodPost, "/wp-json/batch/v1", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "requests") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestBatchRejectsGETMethodInRequests(t *testing.T) {
	c := bootOrSkip(t)
	srv := &rest.Server{}
	body := `{"requests":[{"method":"GET","path":"/wp/v2/types"}]}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/batch/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "rest_not_in_enum") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestBatchDispatchesWriteStub(t *testing.T) {
	c := bootOrSkip(t)
	srv := &rest.Server{}
	body := `{"requests":[{"method":"POST","path":"/wp/v2/posts","body":{"title":"x","status":"draft"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/batch/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handle(rec, req, c)
	if rec.Code != http.StatusMultiStatus {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Responses []struct {
			Status  int            `json:"status"`
			Body    map[string]any `json:"body"`
			Headers any            `json:"headers"`
		} `json:"responses"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if len(out.Responses) != 1 {
		t.Fatalf("responses=%d", len(out.Responses))
	}
	if out.Responses[0].Status != http.StatusUnauthorized && out.Responses[0].Status != http.StatusForbidden {
		// Anon create should be rejected; accept 401/403 depending on handler.
		if out.Responses[0].Status < 400 {
			t.Fatalf("expected error status, got %#v", out.Responses[0])
		}
	}
	// P4.70: sub-response headers are enveloped (Allow/Link/… when present).
	switch h := out.Responses[0].Headers.(type) {
	case map[string]any:
		if len(h) == 0 {
			t.Fatalf("expected non-empty header map, got %#v", h)
		}
	case []any:
		// Empty headers array matches PHP empty assoc → [].
	default:
		t.Fatalf("unexpected headers type %#v", out.Responses[0].Headers)
	}
}

func TestBatchRequireAllValidate(t *testing.T) {
	c := bootOrSkip(t)
	srv := &rest.Server{}
	body := `{"validation":"require-all-validate","requests":[{"method":"POST","path":"/wp/v2/types","body":{}}]}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/batch/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handle(rec, req, c)
	if rec.Code != http.StatusMultiStatus {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Failed    string `json:"failed"`
		Responses []any  `json:"responses"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if out.Failed != "validation" {
		t.Fatalf("failed=%q body=%s", out.Failed, rec.Body.String())
	}
	if len(out.Responses) != 1 {
		t.Fatalf("responses=%d", len(out.Responses))
	}
	if out.Responses[0] == nil {
		t.Fatal("expected error envelope, got null")
	}
}
