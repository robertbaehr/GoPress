package admin

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveQuickDraftSave handles post.php action=post-quickdraft-save (P7.55 / P7.134).
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

	// P7.134: verify add-post nonce (wp_dashboard_quick_press / post.php).
	nonce := r.FormValue("_wpnonce")
	if c.VerifyNonce(nonce, "add-post") == 0 {
		h.renderDashboard(w, r, c, quickDraftNotice(c,
			"Unable to submit this form, please refresh and try again.", "error"))
		return
	}

	title := strings.TrimSpace(r.FormValue("post_title"))
	content := strings.TrimSpace(r.FormValue("content"))
	if title == "" && content == "" {
		h.renderDashboard(w, r, c, quickDraftNotice(c,
			"Cannot create a draft post with empty title and content.", "error"))
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
		h.renderDashboard(w, r, c, quickDraftNotice(c,
			"Draft could not be created: "+err.Error(), "error"))
		return
	}

	// P7.134: success notice matches wp_dashboard_quick_press(…, 'success').
	h.renderDashboard(w, r, c, quickDraftNotice(c, "Draft created successfully.", "success"))
	_ = id
}

// quickDraftNotice builds wp_admin_notice markup for Quick Draft feedback (P7.134).
func quickDraftNotice(c *wp.Context, message, noticeType string) string {
	return wp.GetAdminNotice(c, html.EscapeString(message), wp.AdminNoticeArgs{
		Type: noticeType,
		Attributes: map[string]string{
			"role": "alert",
		},
	})
}

// quickDraftFormHTML renders the Quick Draft form + recent drafts (P7.55 / P7.134).
func quickDraftFormHTML(c *wp.Context, noticeHTML string) string {
	nonce := ""
	postID := int64(0)
	if c != nil && c.CurrentUser != nil {
		nonce = c.CreateNonce("add-post")
		postID = ensureQuickDraftAutoDraft(c)
	}
	var b strings.Builder
	b.WriteString(`<form name="post" action="/wp-admin/post.php" method="post" id="quick-press" class="hide-if-no-js">`)
	if strings.TrimSpace(noticeHTML) != "" {
		b.WriteString(noticeHTML)
	}
	b.WriteString(`<div class="input-text-wrap" id="title-wrap"><label for="title">Title</label>`)
	b.WriteString(`<input type="text" name="post_title" id="title" autocomplete="off" /></div>`)
	b.WriteString(`<div class="textarea-wrap" id="description-wrap"><label for="content">Content</label>`)
	b.WriteString(`<textarea name="content" id="content" placeholder="What&#8217;s on your mind?" class="mceEditor" rows="3" cols="15" autocomplete="off"></textarea></div>`)
	b.WriteString(`<p class="submit">`)
	b.WriteString(`<input type="hidden" name="action" id="quickpost-action" value="post-quickdraft-save" />`)
	b.WriteString(`<input type="hidden" name="post_ID" value="`)
	b.WriteString(strconv.FormatInt(postID, 10))
	b.WriteString(`" />`)
	b.WriteString(`<input type="hidden" name="post_type" value="post" />`)
	b.WriteString(`<input type="hidden" id="_wpnonce" name="_wpnonce" value="`)
	b.WriteString(html.EscapeString(nonce))
	b.WriteString(`" />`)
	b.WriteString(`<input type="submit" name="save" id="save-post" class="button button-primary" value="Save Draft" />`)
	b.WriteString(`<br class="clear" /></p></form>`)
	b.WriteString(quickDraftRecentDraftsHTML(c, 4))
	return b.String()
}

// ensureQuickDraftAutoDraft returns/creates the dashboard auto-draft post ID (P7.134).
func ensureQuickDraftAutoDraft(c *wp.Context) int64 {
	if c == nil || c.CurrentUser == nil || c.DB == nil {
		return 0
	}
	uid := c.CurrentUser.ID
	if raw := strings.TrimSpace(wp.GetUserOption(c, "dashboard_quick_press_last_post_id", uid)); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			q, args, err := c.DB.Prepare(
				"SELECT post_status FROM "+c.DB.Posts()+" WHERE ID = %d LIMIT 1", id,
			)
			if err == nil {
				if st, err := c.DB.GetVar(q, args...); err == nil {
					status := ""
					switch v := st.(type) {
					case string:
						status = v
					case []byte:
						status = string(v)
					}
					if status == "auto-draft" {
						return id
					}
				}
			}
		}
	}
	id, err := insertQuickDraftAutoDraft(c, uid)
	if err != nil || id <= 0 {
		return 0
	}
	_ = wp.UpdateUserOption(c, uid, "dashboard_quick_press_last_post_id", strconv.FormatInt(id, 10))
	return id
}

func insertQuickDraftAutoDraft(c *wp.Context, author int64) (int64, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	guid := home + "/?p=0"
	commentStatus := optionString(c, "default_comment_status", "open")
	if commentStatus == "" {
		commentStatus = "open"
	}
	pingStatus := optionString(c, "default_ping_status", "open")
	if pingStatus == "" {
		pingStatus = "open"
	}
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		author, now, now, "", "Auto Draft", "", "auto-draft",
		commentStatus, pingStatus, "", "", "", "", now, now,
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

// quickDraftRecentDraftsHTML lists the current user's recent drafts (P7.134).
func quickDraftRecentDraftsHTML(c *wp.Context, limit int) string {
	if c == nil || c.CurrentUser == nil || c.DB == nil {
		return ""
	}
	if limit <= 0 {
		limit = 4
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_title, post_modified FROM "+c.DB.Posts()+
			" WHERE post_type = %s AND post_status = %s AND post_author = %d"+
			" ORDER BY post_modified DESC LIMIT %d",
		"post", "draft", c.CurrentUser.ID, limit,
	)
	if err != nil {
		return ""
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil || len(rows) == 0 {
		return `<div class="no-drafts"><p>There are no drafts at the moment</p></div>`
	}
	var b strings.Builder
	b.WriteString(`<div class="drafts"><h3>Recent Drafts <a href="/wp-admin/edit.php?post_status=draft&amp;post_type=post">View all drafts</a></h3><ul>`)
	for _, row := range rows {
		id := int64(0)
		switch v := row["ID"].(type) {
		case int64:
			id = v
		case []byte:
			id, _ = strconv.ParseInt(string(v), 10, 64)
		}
		title := strings.TrimSpace(fmt.Sprint(row["post_title"]))
		if title == "" {
			title = "(no title)"
		}
		b.WriteString(`<li><div class="draft-title"><a href="/wp-admin/post.php?post=`)
		b.WriteString(strconv.FormatInt(id, 10))
		b.WriteString(`&amp;action=edit" aria-label="Edit &#8220;`)
		b.WriteString(html.EscapeString(title))
		b.WriteString(`&#8221;">`)
		b.WriteString(html.EscapeString(title))
		b.WriteString(`</a></div></li>`)
	}
	b.WriteString(`</ul></div>`)
	return b.String()
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
	// P7.134: default comment/ping status from options (get_default_comment_status subset).
	commentStatus := optionString(c, "default_comment_status", "open")
	if commentStatus == "" {
		commentStatus = "open"
	}
	pingStatus := optionString(c, "default_ping_status", "open")
	if pingStatus == "" {
		pingStatus = "open"
	}
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		author, now, now, content, title, "", "draft",
		commentStatus, pingStatus, "", slug, "", "", now, now,
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
