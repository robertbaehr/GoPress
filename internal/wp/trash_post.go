// wp_trash_post / wp_untrash_post subset (P7.74).
package wp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	trashMetaStatus = "_wp_trash_meta_status"
	trashMetaTime   = "_wp_trash_meta_time"
)

// TrashPost moves a post to trash, storing the previous status in post meta
// (wp_trash_post subset, P7.74).
func TrashPost(c *Context, id int64) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("wp: TrashPost: invalid args")
	}
	post, err := GetPostByID(c, id)
	if err != nil {
		return err
	}
	if post == nil {
		return fmt.Errorf("wp: TrashPost: post not found")
	}
	if post.Status == "trash" {
		return nil
	}
	prev := post.Status
	if prev == "" {
		prev = "draft"
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET post_status = %s, post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
		"trash", now, now, id,
	)
	if err != nil {
		return err
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		return err
	}
	if c.Meta != nil {
		_, _ = c.Meta.DeleteMetadata(MetaPost, id, trashMetaStatus, "")
		_, _ = c.Meta.DeleteMetadata(MetaPost, id, trashMetaTime, "")
		_, _, _ = c.Meta.AddMetadata(MetaPost, id, trashMetaStatus, prev, true)
		_, _, _ = c.Meta.AddMetadata(MetaPost, id, trashMetaTime, strconv.FormatInt(time.Now().Unix(), 10), true)
	}
	return nil
}

// UntrashPost restores a trashed post to its previous status (wp_untrash_post
// subset, P7.74). Defaults to draft when prior status meta is missing.
func UntrashPost(c *Context, id int64) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("wp: UntrashPost: invalid args")
	}
	post, err := GetPostByID(c, id)
	if err != nil {
		return err
	}
	if post == nil {
		return fmt.Errorf("wp: UntrashPost: post not found")
	}
	if post.Status != "trash" {
		return nil
	}
	prev := "draft"
	if c.Meta != nil {
		if v := c.Meta.GetMetadata(MetaPost, id, trashMetaStatus, true); v != nil {
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
				prev = s
			}
		}
	}
	switch prev {
	case "publish", "draft", "pending", "private", "future":
	default:
		prev = "draft"
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET post_status = %s, post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
		prev, now, now, id,
	)
	if err != nil {
		return err
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		return err
	}
	if c.Meta != nil {
		_, _ = c.Meta.DeleteMetadata(MetaPost, id, trashMetaStatus, "")
		_, _ = c.Meta.DeleteMetadata(MetaPost, id, trashMetaTime, "")
	}
	return nil
}
