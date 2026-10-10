package wp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestMediaHandleSideloadLocalFile(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	tmpDir := t.TempDir()
	tmp := filepath.Join(tmpDir, "side-me.txt")
	payload := []byte("sideload-bytes")
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	up, err := wp.MediaHandleSideload(c, wp.SideloadFile{
		Name:    "side-me.txt",
		TmpName: tmp,
	}, 1, "Sideload Title")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(up.AbsPath)
		if c.DB != nil {
			q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", up.ID)
			_, _ = c.DB.Exec(q, args...)
			q2, a2, _ := c.DB.Prepare("DELETE FROM "+c.DB.PostMeta()+" WHERE post_id = %d", up.ID)
			_, _ = c.DB.Exec(q2, a2...)
		}
	})
	if up.ID <= 0 || up.Title != "Sideload Title" {
		t.Fatalf("up=%+v", up)
	}
	if !strings.HasPrefix(up.RelPath, "2024/06/") {
		t.Fatalf("rel=%q", up.RelPath)
	}
	got, err := os.ReadFile(up.AbsPath)
	if err != nil || string(got) != string(payload) {
		t.Fatalf("content=%q err=%v", got, err)
	}
	// Temp file should be consumed.
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("tmp should be moved away: %v", err)
	}
}

func TestMediaSideloadImageRejectsRemote(t *testing.T) {
	c := testBootQuery(t)
	_, err := wp.MediaSideloadImage(c, "https://example.com/a.png", 0, "", "html")
	if err == nil || !strings.Contains(err.Error(), "remote download") {
		t.Fatalf("expected remote reject: %v", err)
	}
}

func TestMediaSideloadImageLocalPNG(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	// Minimal 1x1 PNG.
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	src := filepath.Join(t.TempDir(), "dot.png")
	if err := os.WriteFile(src, png, 0o644); err != nil {
		t.Fatal(err)
	}
	html, err := wp.MediaSideloadImage(c, src, 0, "Dot", "html")
	if err != nil {
		t.Fatal(err)
	}
	s, ok := html.(string)
	if !ok || !strings.Contains(s, `<img src="`) || !strings.Contains(s, `alt="Dot"`) {
		t.Fatalf("html=%v", html)
	}
	idAny, err := wp.MediaSideloadImage(c, src, 0, "Dot2", "id")
	if err != nil {
		t.Fatal(err)
	}
	id, ok := idAny.(int64)
	if !ok || id <= 0 {
		t.Fatalf("id=%v", idAny)
	}
	t.Cleanup(func() {
		if c.DB != nil {
			q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", id)
			_, _ = c.DB.Exec(q, args...)
		}
	})
}
