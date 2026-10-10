package wp_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSaveUploadedMedia(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	payload := []byte("gopress-upload-test-bytes")
	up, err := wp.SaveUploadedMedia(c, "hello-upload.txt", bytes.NewReader(payload), int64(len(payload)))
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
	if up.ID <= 0 {
		t.Fatal("expected attachment id")
	}
	if !strings.HasPrefix(up.RelPath, "2024/06/") {
		t.Fatalf("rel=%q", up.RelPath)
	}
	if _, err := os.Stat(up.AbsPath); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(up.AbsPath)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("file content mismatch: %v %q", err, got)
	}
	if filepath.Base(up.AbsPath) == "" {
		t.Fatal("empty basename")
	}

	fileMeta, _ := c.Meta.GetMetadata(wp.MetaPost, up.ID, "_wp_attached_file", true).(string)
	if fileMeta != up.RelPath {
		t.Fatalf("meta=%q want %q", fileMeta, up.RelPath)
	}
}
