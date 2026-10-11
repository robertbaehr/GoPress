package admin

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// activityRecentCommentsHTML builds the Recent Comments block for the
// dashboard Activity widget (P7.56 / P7.133 / wp_dashboard_recent_comments).
func activityRecentCommentsHTML(c *wp.Context, limit int) string {
	if limit <= 0 {
		limit = 5
	}
	comments, err := recentActivityComments(c, limit)
	if err != nil || len(comments) == 0 {
		return `<div id="latest-comments" class="activity-block"><h3>Recent Comments</h3>
<p>No comments yet. <a href="/wp-admin/edit-comments.php">View all</a></p>
</div>`
	}
	var b strings.Builder
	b.WriteString(`<div id="latest-comments" class="activity-block table-view-list"><h3>Recent Comments</h3>`)
	b.WriteString(`<ul id="the-comment-list" data-wp-lists="list:comment">`)
	for _, cm := range comments {
		b.WriteString(activityCommentRowHTML(c, cm))
	}
	b.WriteString(`</ul></div>`)
	return b.String()
}

// activityCommentRowHTML renders one dashboard comment row (P7.133 /
// _wp_dashboard_recent_comments_row subset).
func activityCommentRowHTML(c *wp.Context, cm activityComment) string {
	statusClass := "approved"
	pending := false
	if cm.Approved == "0" || cm.Approved == "hold" {
		statusClass = "unapproved"
		pending = true
	}
	var b strings.Builder
	b.WriteString(`<li id="comment-`)
	b.WriteString(fmt.Sprintf("%d", cm.ID))
	b.WriteString(`" class="comment comment-item `)
	b.WriteString(statusClass)
	b.WriteString(`">`)
	b.WriteString(`<div class="dashboard-comment-wrap has-row-actions">`)
	b.WriteString(`<p class="comment-meta">From <cite class="comment-author">`)
	b.WriteString(html.EscapeString(cm.Author))
	b.WriteString(`</cite>`)
	if cm.PostTitle != "" {
		b.WriteString(` on <a href="/wp-admin/post.php?post=`)
		b.WriteString(fmt.Sprintf("%d", cm.PostID))
		b.WriteString(`&amp;action=edit">`)
		b.WriteString(html.EscapeString(cm.PostTitle))
		b.WriteString(`</a>`)
	}
	if pending {
		b.WriteString(` <span class="approve">[Pending]</span>`)
	}
	b.WriteString(`</p>`)
	excerpt, truncated := commentExcerpt(cm.Content, 20)
	b.WriteString(`<blockquote><p>`)
	b.WriteString(html.EscapeString(excerpt))
	if truncated {
		b.WriteString("&hellip;")
	}
	b.WriteString(`</p></blockquote>`)
	b.WriteString(`<p class="row-actions">`)
	b.WriteString(activityCommentRowActions(c, cm, pending))
	b.WriteString(`</p></div></li>`)
	return b.String()
}

func activityCommentRowActions(c *wp.Context, cm activityComment, pending bool) string {
	approveNonce := ""
	delNonce := ""
	if c != nil {
		approveNonce = c.CreateNonce("approve-comment_" + fmt.Sprintf("%d", cm.ID))
		delNonce = c.CreateNonce("delete-comment_" + fmt.Sprintf("%d", cm.ID))
	}
	base := fmt.Sprintf("comment.php?p=%d&c=%d", cm.PostID, cm.ID)
	approveURL := "/wp-admin/" + base + "&action=approvecomment&_wpnonce=" + url.QueryEscape(approveNonce)
	unapproveURL := "/wp-admin/" + base + "&action=unapprovecomment&_wpnonce=" + url.QueryEscape(approveNonce)
	spamURL := "/wp-admin/" + base + "&action=spamcomment&_wpnonce=" + url.QueryEscape(delNonce)
	trashURL := "/wp-admin/" + base + "&action=trashcomment&_wpnonce=" + url.QueryEscape(delNonce)
	editURL := fmt.Sprintf("/wp-admin/comment.php?action=editcomment&c=%d", cm.ID)

	var parts []string
	if pending {
		parts = append(parts, fmt.Sprintf(
			`<span class="approve"><a href="%s" data-wp-lists="dim:the-comment-list:comment-%d:unapproved:e7e7d3:e7e7d3:new=approved" class="vim-a aria-button-if-js" aria-label="Approve this comment">Approve</a></span>`,
			html.EscapeString(approveURL), cm.ID,
		))
	} else {
		parts = append(parts, fmt.Sprintf(
			`<span class="unapprove"><a href="%s" data-wp-lists="dim:the-comment-list:comment-%d:unapproved:e7e7d3:e7e7d3:new=unapproved" class="vim-u aria-button-if-js" aria-label="Unapprove this comment">Unapprove</a></span>`,
			html.EscapeString(unapproveURL), cm.ID,
		))
	}
	parts = append(parts, fmt.Sprintf(
		`<span class="reply hide-if-no-js"> | <button type="button" onclick="window.commentReply && commentReply.open('%d','%d');" class="vim-r button-link hide-if-no-js" aria-label="Reply to this comment">Reply</button></span>`,
		cm.ID, cm.PostID,
	))
	parts = append(parts, fmt.Sprintf(
		`<span class="edit"> | <a href="%s" aria-label="Edit this comment">Edit</a></span>`,
		html.EscapeString(editURL),
	))
	parts = append(parts, fmt.Sprintf(
		`<span class="spam"> | <a href="%s" data-wp-lists="delete:the-comment-list:comment-%d::spam=1" class="vim-s vim-destructive aria-button-if-js" aria-label="Mark this comment as spam">Spam</a></span>`,
		html.EscapeString(spamURL), cm.ID,
	))
	parts = append(parts, fmt.Sprintf(
		`<span class="trash"> | <a href="%s" data-wp-lists="delete:the-comment-list:comment-%d::trash=1" class="delete vim-d vim-destructive aria-button-if-js" aria-label="Move this comment to the Trash">Trash</a></span>`,
		html.EscapeString(trashURL), cm.ID,
	))
	viewClass := "view"
	if pending {
		viewClass = "view hidden"
	}
	parts = append(parts, fmt.Sprintf(
		`<span class="%s"> | <a class="comment-link" href="/wp-admin/edit-comments.php" aria-label="View this comment">View</a></span>`,
		viewClass,
	))
	return strings.Join(parts, "")
}

type activityComment struct {
	ID        int64
	Author    string
	Content   string
	PostID    int64
	PostTitle string
	Approved  string
}

func recentActivityComments(c *wp.Context, limit int) ([]activityComment, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 5
	}
	// Prefer approved comments; pull a wider window then filter spam/trash.
	q, args, err := c.DB.Prepare(
		"SELECT c.comment_ID, c.comment_author, c.comment_content, c.comment_post_ID, c.comment_approved, "+
			"p.post_title FROM "+c.DB.Comments()+" c "+
			"LEFT JOIN "+c.DB.Posts()+" p ON p.ID = c.comment_post_ID "+
			"WHERE c.comment_approved NOT IN ('spam','trash','post-trashed') "+
			"ORDER BY c.comment_date_gmt DESC LIMIT %d",
		limit*3,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]activityComment, 0, limit)
	for _, row := range rows {
		approved := asString(row["comment_approved"])
		// Include approved ("1") and pending ("0"); skip other junk.
		if approved != "1" && approved != "0" && approved != "hold" {
			continue
		}
		out = append(out, activityComment{
			ID:        asInt64(row["comment_ID"]),
			Author:    asString(row["comment_author"]),
			Content:   asString(row["comment_content"]),
			PostID:    asInt64(row["comment_post_ID"]),
			PostTitle: asString(row["post_title"]),
			Approved:  approved,
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func commentExcerpt(content string, words int) (string, bool) {
	text := strings.TrimSpace(wp.WPStripAllTags(content, true))
	if text == "" {
		return "", false
	}
	fields := strings.Fields(text)
	if words <= 0 || len(fields) <= words {
		return text, false
	}
	return strings.Join(fields[:words], " "), true
}
