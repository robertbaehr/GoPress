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

func TestCreateFontFaceRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	body := `{"font_face_settings":{"fontFamily":"Test","src":"https://example.com/t.woff2"}}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/font-families/1/font-faces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateAndDeleteFontFace(t *testing.T) {
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

	// Create parent family first.
	famBody := `{"font_family_settings":{"name":"Face Family","slug":"face-family","fontFamily":"Face Family"}}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/font-families", bytes.NewBufferString(famBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("family create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var fam rest.FontFamily
	if err := json.Unmarshal(rec.Body.Bytes(), &fam); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d OR post_parent = %d", fam.ID, fam.ID)
		_, _ = c.DB.Exec(q, args...)
	})

	faceBody := `{"font_face_settings":{"fontFamily":"Face Family","fontStyle":"normal","fontWeight":"400","src":"https://example.com/face.woff2"}}`
	path := "/wp-json/wp/v2/font-families/" + strconv.FormatInt(fam.ID, 10) + "/font-faces"
	req = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(faceBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("face create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var face rest.FontFace
	if err := json.Unmarshal(rec.Body.Bytes(), &face); err != nil {
		t.Fatal(err)
	}
	if face.ID <= 0 || face.Parent != fam.ID {
		t.Fatalf("face: %+v", face)
	}
	if face.FontFaceSettings["src"] != "https://example.com/face.woff2" {
		t.Fatalf("src: %+v", face.FontFaceSettings)
	}

	// Duplicate → 400
	req = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(faceBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("duplicate status=%d body=%s", rec.Code, rec.Body.String())
	}

	idPath := path + "/" + strconv.FormatInt(face.ID, 10)
	req = httptest.NewRequest(http.MethodDelete, idPath, nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("trash status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, idPath+"?force=true", nil)
	req.AddCookie(ck)
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
