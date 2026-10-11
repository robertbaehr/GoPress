package wp_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestMediaHandleUploadWithPostDataAndParentDate(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	parent, err := wp.GetPostByID(c, 1)
	if err != nil || parent == nil {
		t.Fatalf("parent post: %v", err)
	}
	wantDate := parent.Date

	payload := []byte("gopress-p841-upload")
	up, err := wp.MediaHandleUploadWith(c, "my_cool-file.txt", bytes.NewReader(payload), int64(len(payload)), 1, &wp.MediaUploadPostData{
		Title:   "Override Title",
		Excerpt: "Caption excerpt",
	})
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
	if up.Title != "Override Title" {
		t.Fatalf("title=%q want Override Title", up.Title)
	}
	q, args, err := c.DB.Prepare(
		"SELECT post_title, post_excerpt, post_date, post_parent FROM "+c.DB.Posts()+" WHERE ID = %d",
		up.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("row: %v", err)
	}
	if fmt.Sprint(row["post_title"]) != "Override Title" {
		t.Fatalf("post_title=%v", row["post_title"])
	}
	if fmt.Sprint(row["post_excerpt"]) != "Caption excerpt" {
		t.Fatalf("post_excerpt=%v", row["post_excerpt"])
	}
	if fmt.Sprint(row["post_parent"]) != "1" {
		t.Fatalf("post_parent=%v", row["post_parent"])
	}
	gotDate := fmt.Sprint(row["post_date"])
	if wantDate != "" && gotDate != wantDate {
		t.Fatalf("post_date=%q want parent date %q", gotDate, wantDate)
	}
}

func TestMediaHandleUploadSanitizesBasenameTitle(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	payload := []byte("basename-title")
	up, err := wp.MediaHandleUpload(c, "hello_world-file.txt", bytes.NewReader(payload), int64(len(payload)), 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(up.AbsPath)
		if c.DB != nil {
			q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", up.ID)
			_, _ = c.DB.Exec(q, args...)
		}
	})
	if up.Title != "hello world file" {
		t.Fatalf("title=%q want sanitized basename", up.Title)
	}
}
