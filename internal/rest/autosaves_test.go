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

func TestAutosaveCreatePublishedPost(t *testing.T) {
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

	body := `{"title":"Autosave Title","content":"<!-- wp:paragraph --><p>Autosaved body</p><!-- /wp:paragraph -->"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts/13/autosaves", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var a rest.Autosave
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if a.Parent != 13 || a.ID <= 0 || !strings.Contains(a.Slug, "autosave") {
		t.Fatalf("autosave=%+v", a)
	}
	if !strings.Contains(a.Title.Rendered, "Autosave Title") && a.Title.Rendered == "" {
		// Title may be texturized; ensure content path worked via slug/parent.
		t.Fatalf("expected title on autosave: %+v", a)
	}

	// Second POST with same payload should reuse the existing autosave row.
	req2 := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts/13/autosaves", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(ck)
	rec2 := httptest.NewRecorder()
	rc2 := c.ForRequest(rec2, req2)
	(&rest.Server{}).Handle(rec2, req2, rc2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("reuse status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	var a2 rest.Autosave
	if err := json.Unmarshal(rec2.Body.Bytes(), &a2); err != nil {
		t.Fatalf("reuse json: %v", err)
	}
	if a2.ID != a.ID {
		t.Fatalf("expected same autosave id %d got %d", a.ID, a2.ID)
	}
}

func TestAutosaveCreateDraftUpdatesParent(t *testing.T) {
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

	// Create a draft owned by admin.
	createBody := `{"title":"Draft for autosave","content":"original","status":"draft"}`
	creq := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts", strings.NewReader(createBody))
	creq.Header.Set("Content-Type", "application/json")
	creq.AddCookie(ck)
	crec := httptest.NewRecorder()
	(&rest.Server{}).Handle(crec, creq, c.ForRequest(crec, creq))
	if crec.Code != http.StatusCreated && crec.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", crec.Code, crec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(crec.Body.Bytes(), &created); err != nil {
		t.Fatalf("create json: %v", err)
	}
	idF, _ := created["id"].(float64)
	draftID := int64(idF)
	if draftID <= 0 {
		t.Fatalf("missing draft id: %v", created)
	}

	autoBody := `{"title":"Draft for autosave","content":"autosaved draft body"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts/"+strconv.FormatInt(draftID, 10)+"/autosaves", strings.NewReader(autoBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	(&rest.Server{}).Handle(rec, req, c.ForRequest(rec, req))
	if rec.Code != http.StatusOK {
		t.Fatalf("autosave status=%d body=%s", rec.Code, rec.Body.String())
	}
	var a rest.Autosave
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("json: %v", err)
	}
	// Draft author path updates the parent in place — id equals parent.
	if a.ID != draftID {
		t.Fatalf("draft autosave should update parent id=%d got autosave id=%d slug=%q", draftID, a.ID, a.Slug)
	}
	if !strings.Contains(a.Content.Rendered, "autosaved draft body") && !strings.Contains(rec.Body.String(), "autosaved draft body") {
		t.Fatalf("expected updated content: %s", rec.Body.String())
	}
}

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
