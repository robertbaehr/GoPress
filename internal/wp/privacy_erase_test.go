package wp_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestErasePersonalDataAnonymizesComments(t *testing.T) {
	c := testBootQuery(t)
	email := fmt.Sprintf("erase-%d@example.com", time.Now().UnixNano())

	posts, err := wp.GetPosts(c, wp.QueryArgs{PostsPerPage: 1, PostStatus: "publish"})
	if err != nil || len(posts) == 0 {
		t.Skip("no posts for comment insert")
	}
	postID := posts[0].ID
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Comments()+
			" (comment_post_ID, comment_author, comment_author_email, comment_author_url, comment_author_IP, "+
			"comment_date, comment_date_gmt, comment_content, comment_karma, comment_approved, comment_agent, "+
			"comment_type, comment_parent, user_id) VALUES (%d,%s,%s,%s,%s,%s,%s,%s,%d,%s,%s,%s,%d,%d)",
		postID, "Erase Me", email, "https://example.com", "127.0.0.1",
		now, now, "please erase", 0, "1", "GoPressTest", "comment", 0, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	ins, err := c.DB.Exec(q, args...)
	if err != nil {
		t.Fatal(err)
	}
	cid, _ := ins.LastInsertId()
	t.Cleanup(func() {
		dq, dargs, _ := c.DB.Prepare("DELETE FROM "+c.DB.Comments()+" WHERE comment_ID = %d", cid)
		_, _ = c.DB.Exec(dq, dargs...)
	})

	res, err := wp.ErasePersonalData(c, email)
	if err != nil {
		t.Fatal(err)
	}
	if res.CommentsAnonymized < 1 {
		t.Fatalf("expected comments anonymized, got %+v", res)
	}

	rq, rargs, err := c.DB.Prepare(
		"SELECT comment_author, comment_author_email, comment_author_url FROM "+c.DB.Comments()+" WHERE comment_ID = %d",
		cid,
	)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(rq, rargs...)
	if err != nil || row == nil {
		t.Fatalf("reload: %v", err)
	}
	if fmt.Sprint(row["comment_author"]) != "Anonymous" {
		t.Fatalf("author=%v", row["comment_author"])
	}
	if fmt.Sprint(row["comment_author_email"]) != "" {
		t.Fatalf("email=%v", row["comment_author_email"])
	}
}

func TestErasePersonalDataRejectsBadEmail(t *testing.T) {
	c := testBootQuery(t)
	if _, err := wp.ErasePersonalData(c, "not-an-email"); err == nil {
		t.Fatal("expected error")
	}
}
