package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestEditTagsCategoriesStub(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/edit-tags.php?taxonomy=category", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/edit-tags.php") {
		t.Fatal("edit-tags.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Categories") {
		t.Fatalf("missing Categories heading: %q", body)
	}
	if !strings.Contains(body, "uncategorized") {
		t.Fatalf("expected uncategorized term: %q", body)
	}
	if !strings.Contains(body, "wp-list-table") {
		t.Fatalf("expected list table: %q", body)
	}
	if !strings.Contains(body, `id="addtag"`) || !strings.Contains(body, "Add New Category") {
		t.Fatalf("expected add category form: %q", body)
	}
	if !strings.Contains(body, `name="tag-name"`) || !strings.Contains(body, `_wpnonce_add-tag`) {
		t.Fatalf("expected tag-name + nonce fields: %q", body)
	}
}

func TestEditTagsTagsStub(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "editor")
	if err != nil || user == nil {
		t.Fatalf("editor: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/edit-tags.php?taxonomy=post_tag", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/edit-tags.php") {
		t.Fatal("edit-tags.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<h1") || !strings.Contains(body, "Tags") {
		t.Fatalf("missing Tags heading: %q", body)
	}
	if !strings.Contains(body, `id="addtag"`) || !strings.Contains(body, "Add New Tag") {
		t.Fatalf("expected add tag form: %q", body)
	}
}

func TestEditTagsDeleteCategory(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	probe := httptest.NewRequest(http.MethodGet, "/wp-admin/edit-tags.php?taxonomy=category", nil)
	probe.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	probeRec := httptest.NewRecorder()
	pc := c.ForRequest(probeRec, probe)
	pc.AuthenticateRequest()
	addNonce := pc.CreateNonce("add-tag")
	name := "gopress-p759-del"
	form := "action=add-tag&taxonomy=category&tag-name=" + name + "&slug=&description=bye&_wpnonce_add-tag=" + addNonce
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/edit-tags.php", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	reqC := c.ForRequest(rec, req)
	reqC.AuthenticateRequest()
	if !(&admin.Handler{}).Handle(rec, req, reqC, "/wp-admin/edit-tags.php") {
		t.Fatal("add not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("add status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Resolve term id.
	q, args, err := c.DB.Prepare("SELECT term_id FROM "+c.DB.Terms()+" WHERE slug = %s LIMIT 1", name)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("term missing: %v", err)
	}
	var id int64
	switch v := row["term_id"].(type) {
	case int64:
		id = v
	case []byte:
		fmt.Sscan(string(v), &id)
	}
	if id <= 0 {
		t.Fatal("bad term id")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/wp-admin/edit-tags.php?taxonomy=category", nil)
	listReq.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	listRec := httptest.NewRecorder()
	listC := c.ForRequest(listRec, listReq)
	listC.AuthenticateRequest()
	(&admin.Handler{}).Handle(listRec, listReq, listC, "/wp-admin/edit-tags.php")
	body := listRec.Body.String()
	if !strings.Contains(body, "delete-tag") || !strings.Contains(body, "action=delete") {
		t.Fatalf("expected delete row action: %q", body)
	}

	delNonce := listC.CreateNonce("delete-tag_" + fmt.Sprintf("%d", id))
	delURL := fmt.Sprintf("/wp-admin/edit-tags.php?taxonomy=category&action=delete&tag_ID=%d&_wpnonce=%s", id, delNonce)
	delReq := httptest.NewRequest(http.MethodGet, delURL, nil)
	delReq.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	delRec := httptest.NewRecorder()
	delC := c.ForRequest(delRec, delReq)
	delC.AuthenticateRequest()
	if !(&admin.Handler{}).Handle(delRec, delReq, delC, "/wp-admin/edit-tags.php") {
		t.Fatal("delete not handled")
	}
	if delRec.Code != http.StatusFound {
		t.Fatalf("delete status=%d body=%s", delRec.Code, delRec.Body.String())
	}
	loc := delRec.Header().Get("Location")
	if !strings.Contains(loc, "message=2") {
		t.Fatalf("location=%q", loc)
	}

	row, _ = c.DB.GetRow(q, args...)
	if row != nil {
		t.Fatal("term still present after delete")
	}

	delReq = httptest.NewRequest(http.MethodGet, loc, nil)
	delReq.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	delRec = httptest.NewRecorder()
	delC = c.ForRequest(delRec, delReq)
	(&admin.Handler{}).Handle(delRec, delReq, delC, "/wp-admin/edit-tags.php")
	body = delRec.Body.String()
	if !strings.Contains(body, "Category deleted.") {
		t.Fatalf("expected deleted notice: %q", body)
	}
	if strings.Contains(body, name) {
		t.Fatalf("deleted term still listed: %q", body)
	}
}

func TestEditTagsAddCategory(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	probe := httptest.NewRequest(http.MethodGet, "/wp-admin/edit-tags.php?taxonomy=category", nil)
	probe.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	probeRec := httptest.NewRecorder()
	pc := c.ForRequest(probeRec, probe)
	pc.AuthenticateRequest()
	nonce := pc.CreateNonce("add-tag")
	if nonce == "" {
		t.Fatal("empty nonce")
	}
	name := "gopress-p758-cat"
	form := "action=add-tag&taxonomy=category&tag-name=" + name + "&slug=&description=hello&_wpnonce_add-tag=" + nonce
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/edit-tags.php", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	reqC := c.ForRequest(rec, req)
	reqC.AuthenticateRequest()
	if !(&admin.Handler{}).Handle(rec, req, reqC, "/wp-admin/edit-tags.php") {
		t.Fatal("edit-tags.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "message=1") {
		t.Fatalf("location=%q", loc)
	}
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("SELECT term_id FROM "+c.DB.Terms()+" WHERE slug = %s LIMIT 1", name)
		if err != nil {
			return
		}
		row, _ := c.DB.GetRow(q, args...)
		if row == nil {
			return
		}
		id := int64(0)
		switch v := row["term_id"].(type) {
		case int64:
			id = v
		case []byte:
			fmt.Sscan(string(v), &id)
		}
		if id > 0 {
			dq, dargs, _ := c.DB.Prepare("DELETE FROM "+c.DB.TermTaxonomy()+" WHERE term_id = %d", id)
			_, _ = c.DB.Exec(dq, dargs...)
			dq, dargs, _ = c.DB.Prepare("DELETE FROM "+c.DB.Terms()+" WHERE term_id = %d", id)
			_, _ = c.DB.Exec(dq, dargs...)
		}
	})

	req = httptest.NewRequest(http.MethodGet, loc, nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	reqC = c.ForRequest(rec, req)
	(&admin.Handler{}).Handle(rec, req, reqC, "/wp-admin/edit-tags.php")
	body := rec.Body.String()
	if !strings.Contains(body, name) || !strings.Contains(body, "Category added.") {
		t.Fatalf("expected new category listed: %q", body)
	}
}
