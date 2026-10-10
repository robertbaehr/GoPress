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

func TestCreateFontFamilyRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	body := `{"font_family_settings":{"name":"Test","slug":"test","fontFamily":"Test, sans-serif"}}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/font-families", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateAndDeleteFontFamily(t *testing.T) {
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

	body := `{"font_family_settings":{"name":"GoPress Sans","slug":"gopress-sans","fontFamily":"\"GoPress Sans\", sans-serif","preview":""}}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/font-families", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.FontFamily
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID <= 0 {
		t.Fatalf("missing id: %+v", created)
	}
	if created.FontFamilySettings["slug"] != "gopress-sans" {
		t.Fatalf("slug: %+v", created.FontFamilySettings)
	}
	if created.FontFamilySettings["fontFamily"] != `"GoPress Sans", sans-serif` {
		t.Fatalf("fontFamily: %+v", created.FontFamilySettings)
	}
	t.Cleanup(func() {
		q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", created.ID)
		_, _ = c.DB.Exec(q, args...)
	})

	// Duplicate slug → 400
	req = httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/font-families", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("duplicate status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Delete without force → 501
	idStr := strconv.FormatInt(created.ID, 10)
	req = httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/font-families/"+idStr, nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("trash status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Force delete → 200
	req = httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/font-families/"+idStr+"?force=true", nil)
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
		t.Fatalf("deleted payload: %+v", del)
	}
}

func TestCreateFontFamilyStringifiedSettings(t *testing.T) {
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
	inner := `{"name":"String Sans","slug":"string-sans","fontFamily":"String Sans"}`
	raw, _ := json.Marshal(map[string]any{"font_family_settings": inner})
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/font-families", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.FontFamily
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", created.ID)
		_, _ = c.DB.Exec(q, args...)
	})
}
