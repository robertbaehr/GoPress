package admin

import (
	"fmt"
	"html"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// activityRecentCommentsHTML builds the Recent Comments block for the
// dashboard Activity widget (P7.56 / wp_dashboard_recent_comments).
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
		b.WriteString(`<li id="comment-`)
		b.WriteString(fmt.Sprintf("%d", cm.ID))
		b.WriteString(`" class="comment comment-item">`)
		b.WriteString(`<div class="dashboard-comment-wrap">`)
		b.WriteString(`<p class="comment-meta">From <strong class="comment-author">`)
		b.WriteString(html.EscapeString(cm.Author))
		b.WriteString(`</strong>`)
		if cm.PostTitle != "" {
			b.WriteString(` on <a href="/wp-admin/post.php?post=`)
			b.WriteString(fmt.Sprintf("%d", cm.PostID))
			b.WriteString(`&amp;action=edit">`)
			b.WriteString(html.EscapeString(cm.PostTitle))
			b.WriteString(`</a>`)
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
		b.WriteString(`<span class="edit"><a href="/wp-admin/comment.php?action=editcomment&amp;c=`)
		b.WriteString(fmt.Sprintf("%d", cm.ID))
		b.WriteString(`">Edit</a></span>`)
		b.WriteString(` | <span class="view"><a href="/wp-admin/edit-comments.php">View all</a></span>`)
		b.WriteString(`</p></div></li>`)
	}
	b.WriteString(`</ul></div>`)
	return b.String()
}

type activityComment struct {
	ID        int64
	Author    string
	Content   string
	PostID    int64
	PostTitle string
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
