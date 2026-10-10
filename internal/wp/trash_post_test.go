package wp_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestTrashAndUntrashPost(t *testing.T) {
	cfg := testBootConfig()
	cfg.DBName = "gopress_go" // avoid mutating oracle fixture DB
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		int64(1), "2024-01-01 00:00:00", "2024-01-01 00:00:00", "trash me", "Trash Restore Test",
		"", "publish", "closed", "closed", "", "trash-restore-test", "", "", "2024-01-01 00:00:00", "2024-01-01 00:00:00",
		"", 0, "http://127.0.0.1:8082/?p=0", 0, "post", "", 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	t.Cleanup(func() {
		dq, dargs, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", id)
		_, _ = c.DB.Exec(dq, dargs...)
		if c.Meta != nil {
			_, _ = c.Meta.DeleteMetadata(wp.MetaPost, id, "_wp_trash_meta_status", "")
			_, _ = c.Meta.DeleteMetadata(wp.MetaPost, id, "_wp_trash_meta_time", "")
		}
	})

	if err := wp.TrashPost(c, id); err != nil {
		t.Fatalf("TrashPost: %v", err)
	}
	p, err := wp.GetPostByID(c, id)
	if err != nil || p == nil {
		t.Fatalf("after trash: %v %#v", err, p)
	}
	if p.Status != "trash" {
		t.Fatalf("expected trash, got %q", p.Status)
	}
	if c.Meta != nil {
		got := fmt.Sprint(c.Meta.GetMetadata(wp.MetaPost, id, "_wp_trash_meta_status", true))
		if got != "publish" {
			t.Fatalf("expected trash meta publish, got %q", got)
		}
	}

	if err := wp.UntrashPost(c, id); err != nil {
		t.Fatalf("UntrashPost: %v", err)
	}
	p, err = wp.GetPostByID(c, id)
	if err != nil || p == nil {
		t.Fatalf("after untrash: %v %#v", err, p)
	}
	if p.Status != "publish" {
		t.Fatalf("expected restore to publish, got %q", p.Status)
	}
}
