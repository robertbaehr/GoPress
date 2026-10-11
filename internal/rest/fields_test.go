package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestRegisterRestFieldGetCallback(t *testing.T) {
	// P9.26: register_rest_field merges get_callback into post responses.
	rest.ResetRestFields()
	t.Cleanup(rest.ResetRestFields)

	rest.RegisterRestField("post", "gopress_flag", rest.RestFieldArgs{
		GetCallback: func(object map[string]any, fieldName string, r *http.Request, objectType string) any {
			if fieldName != "gopress_flag" || objectType != "post" {
				t.Fatalf("callback args field=%q type=%q", fieldName, objectType)
			}
			id, _ := object["id"].(float64)
			return map[string]any{"ok": true, "id": int64(id)}
		},
		Schema: map[string]any{"type": "object"},
	})
	if got := rest.RestFieldNames("post"); len(got) != 1 || got[0] != "gopress_flag" {
		t.Fatalf("names=%v", got)
	}

	c := bootOrSkip(t)
	defer c.DB.Close()
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?per_page=1", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected posts")
	}
	flag, ok := posts[0]["gopress_flag"].(map[string]any)
	if !ok || flag["ok"] != true {
		t.Fatalf("gopress_flag=%v", posts[0]["gopress_flag"])
	}
}

func TestRegisterRestFieldMultiTypeAndNoOp(t *testing.T) {
	rest.ResetRestFields()
	t.Cleanup(rest.ResetRestFields)

	rest.RegisterRestField([]string{"post", "page"}, "gp_x", rest.RestFieldArgs{
		GetCallback: func(object map[string]any, fieldName string, r *http.Request, objectType string) any {
			return objectType + ":" + fieldName
		},
	})
	names := rest.RestFieldNames("page")
	if len(names) != 1 || names[0] != "gp_x" {
		t.Fatalf("page names=%v", names)
	}
	// Without registrations for comment, EncodeWithAdditionalFields is identity.
	in := []rest.Post{{ID: 1, Status: "publish", Type: "post"}}
	out := rest.EncodeWithAdditionalFields("comment", in, nil)
	if _, ok := out.([]rest.Post); !ok {
		t.Fatalf("expected unchanged []Post, got %T", out)
	}
}
