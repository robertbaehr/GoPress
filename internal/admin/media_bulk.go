package admin

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveUpload lists media and handles bulk delete on list view (P7.61).
func (h *Handler) serveUpload(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method == http.MethodPost {
		h.handleMediaBulk(w, r, c)
		return
	}
	blogname := optionString(c, "blogname", "GoPress")
	items, err := recentMedia(c, 20)
	if err != nil {
		http.Error(w, "media: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	// P7.42: grid is the WP default; ?mode=list keeps the list table.
	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	if mode != "list" && mode != "grid" {
		mode = mediaLibraryMode(c)
	}
	if mode == "grid" {
		writeHTML(w, http.StatusOK, renderMediaGrid(blogname, items, c, r))
		return
	}
	writeHTML(w, http.StatusOK, renderMediaList(blogname, items, c, r))
}

func (h *Handler) handleMediaBulk(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil {
		http.Error(w, "Sorry, you are not allowed to delete this item.\n", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	action := strings.TrimSpace(r.FormValue("action"))
	if action == "" || action == "-1" {
		action = strings.TrimSpace(r.FormValue("action2"))
	}
	if action != "delete" {
		http.Redirect(w, r, "/wp-admin/upload.php?mode=list", http.StatusFound)
		return
	}
	ids := collectMediaIDs(r)
	deleted := 0
	for _, id := range ids {
		if !c.UserCan(c.CurrentUser, "delete_post", id) && !c.UserCan(c.CurrentUser, "delete_posts") {
			continue
		}
		if err := deleteAdminAttachment(c, id); err == nil {
			deleted++
		}
	}
	loc := "/wp-admin/upload.php?mode=list"
	if deleted > 0 {
		loc += "&deleted=" + strconv.Itoa(deleted)
	}
	http.Redirect(w, r, loc, http.StatusFound)
}

func collectMediaIDs(r *http.Request) []int64 {
	if r == nil {
		return nil
	}
	raw := r.Form["media[]"]
	if len(raw) == 0 {
		raw = r.Form["media"]
	}
	if len(raw) == 0 {
		raw = r.Form["post[]"]
	}
	if len(raw) == 0 {
		raw = r.Form["post"]
	}
	out := make([]int64, 0, len(raw))
	seen := map[int64]bool{}
	for _, s := range raw {
		id, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// deleteAdminAttachment force-deletes an attachment row + common meta (P7.61 stub).
func deleteAdminAttachment(c *wp.Context, id int64) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("invalid attachment")
	}
	if c.Meta != nil {
		for _, key := range []string{
			"_wp_attached_file", "_wp_attachment_metadata", "_wp_attachment_image_alt",
		} {
			_, _ = c.Meta.DeleteMetadata(wp.MetaPost, id, key, "")
		}
	}
	q, args, err := c.DB.Prepare(
		"DELETE FROM "+c.DB.Posts()+" WHERE ID = %d AND post_type = %s",
		id, "attachment",
	)
	if err != nil {
		return err
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("attachment not found")
	}
	return nil
}

func mediaBulkDeletedNotice(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	n := strings.TrimSpace(r.URL.Query().Get("deleted"))
	if n == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div id="message" class="updated notice is-dismissible"><p>`)
	if n == "1" {
		b.WriteString("Media file permanently deleted.")
	} else {
		b.WriteString(html.EscapeString(n))
		b.WriteString(" media files permanently deleted.")
	}
	b.WriteString(`</p></div>` + "\n")
	return b.String()
}
