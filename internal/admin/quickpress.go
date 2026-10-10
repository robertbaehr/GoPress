package admin

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveQuickDraftSave handles post.php action=post-quickdraft-save (P7.55).
// Creates a draft post from the dashboard Quick Draft widget.
func (h *Handler) serveQuickDraftSave(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed\n", http.StatusMethodNotAllowed)
		return
	}
	if c == nil || c.CurrentUser == nil {
		http.Error(w, "Unauthorized\n", http.StatusUnauthorized)
		return
	}
	if !c.UserCan(c.CurrentUser, "edit_posts") && !c.UserCan(c.CurrentUser, "create_posts") {
		http.Error(w, "Sorry, you are not allowed to create posts.\n", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad form\n", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("post_title"))
	content := strings.TrimSpace(r.FormValue("content"))
	if title == "" && content == "" {
		h.renderDashboard(w, r, c,
			`<div class="notice notice-error"><p>Cannot create a draft post with empty title and content.</p></div>`)
		return
	}

	if content != "" && !strings.Contains(content, "<!-- wp:paragraph -->") {
		// Wrap Quick Draft content in a Paragraph block (WP post-quickdraft-save).
		escaped := strings.NewReplacer("\r\n", "<br />", "\r", "<br />", "\n", "<br />").Replace(content)
		content = "<!-- wp:paragraph -->" + escaped + "<!-- /wp:paragraph -->"
	}
	if title == "" {
		title = "Auto Draft"
	}

	id, err := insertQuickDraft(c, c.CurrentUser.ID, title, content)
	if err != nil {
		h.renderDashboard(w, r, c,
			`<div class="notice notice-error"><p>Draft could not be created: `+
				html.EscapeString(err.Error())+`</p></div>`)
		return
	}

	msg := fmt.Sprintf(
		`<div class="notice notice-success is-dismissible"><p>Draft created successfully. <a href="/wp-admin/post.php?post=%d&amp;action=edit">Edit draft</a></p></div>`,
		id,
	)
	h.renderDashboard(w, r, c, msg)
}

func insertQuickDraft(c *wp.Context, author int64, title, content string) (int64, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	slug := wp.SanitizeTitle(title)
	if slug == "" {
		slug = "draft"
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	guid := home + "/?p=0"
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		author, now, now, content, title, "", "draft",
		"open", "open", "", slug, "", "", now, now,
		"", 0, guid, 0, "post", "", 0,
	)
	if err != nil {
		return 0, err
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	uq, uargs, uerr := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET guid = %s WHERE ID = %d",
		fmt.Sprintf("%s/?p=%d", home, id), id,
	)
	if uerr == nil {
		_, _ = c.DB.Exec(uq, uargs...)
	}
	return id, nil
}
