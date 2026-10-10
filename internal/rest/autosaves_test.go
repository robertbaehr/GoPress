package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestAutosavesListStub(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/13/autosaves", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var autos []rest.Autosave
	if err := json.Unmarshal(rec.Body.Bytes(), &autos); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if len(autos) < 1 {
		t.Fatalf("expected fixture autosave for post 13, got %d", len(autos))
	}
	a := autos[0]
	if a.Parent != 13 || a.ID <= 0 || !strings.Contains(a.Slug, "autosave") {
		t.Fatalf("autosave=%+v", a)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/13/autosaves/"+strconv.FormatInt(a.ID, 10), nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("item status=%d body=%s", rec.Code, rec.Body.String())
	}
}
