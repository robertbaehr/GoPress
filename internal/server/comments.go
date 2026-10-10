package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveCommentsPost handles POST /wp-comments-post.php (P5.8).
func (s *Server) serveCommentsPost(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, "Method Not Allowed\n")
		return
	}
	if c == nil {
		http.Error(w, "GoPress context unavailable\n", http.StatusServiceUnavailable)
		return
	}
	c.AuthenticateRequest()
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form\n", http.StatusBadRequest)
		return
	}

	postID, _ := strconv.ParseInt(strings.TrimSpace(r.Form.Get("comment_post_ID")), 10, 64)
	if postID <= 0 {
		http.Error(w, "Invalid post ID.\n", http.StatusBadRequest)
		return
	}
	post, err := wp.GetPost(c, wp.QueryArgs{P: postID, PostType: "post", PostStatus: "publish"})
	if err != nil {
		http.Error(w, "post: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if post == nil {
		// Also allow commenting on pages.
		post, err = wp.GetPost(c, wp.QueryArgs{P: postID, PostType: "page", PostStatus: "publish"})
		if err != nil {
			http.Error(w, "post: "+err.Error()+"\n", http.StatusInternalServerError)
			return
		}
	}
	if post == nil {
		http.Error(w, "Invalid post ID.\n", http.StatusBadRequest)
		return
	}
	if strings.ToLower(post.CommentStatus) == "closed" {
		http.Error(w, "Comments are closed.\n", http.StatusForbidden)
		return
	}

	author := strings.TrimSpace(r.Form.Get("author"))
	email := strings.TrimSpace(r.Form.Get("email"))
	url := strings.TrimSpace(r.Form.Get("url"))
	content := strings.TrimSpace(r.Form.Get("comment"))
	parent, _ := strconv.ParseInt(strings.TrimSpace(r.Form.Get("comment_parent")), 10, 64)
	if parent < 0 {
		parent = 0
	}

	var userID int64
	if c.CurrentUser != nil {
		userID = c.CurrentUser.ID
		if author == "" {
			author = c.CurrentUser.DisplayName
			if author == "" {
				author = c.CurrentUser.Login
			}
		}
		if email == "" {
			email = c.CurrentUser.Email
		}
	}

	requireNameEmail := optionTruthyServer(c, "require_name_email")
	if c.CurrentUser == nil && requireNameEmail && (author == "" || email == "") {
		http.Error(w, "Name and email are required.\n", http.StatusBadRequest)
		return
	}
	if content == "" {
		http.Error(w, "Comment content is required.\n", http.StatusBadRequest)
		return
	}

	content = wp.KsesData(content)
	approved := wp.ResolveCommentApproval(c, email, c.CurrentUser)

	id, err := wp.InsertComment(c, wp.CommentInsert{
		PostID:      postID,
		Author:      author,
		AuthorEmail: email,
		AuthorURL:   url,
		AuthorIP:    requestIP(r),
		Content:     content,
		Parent:      parent,
		UserID:      userID,
		Agent:       truncate(r.UserAgent(), 255),
		Approved:    approved,
	})
	if err != nil {
		http.Error(w, "comment: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}

	loc := strings.TrimSpace(r.Form.Get("redirect_to"))
	if loc == "" {
		home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		loc = fmt.Sprintf("%s/?p=%d#comment-%d", home, postID, id)
	} else if !strings.Contains(loc, "#comment-") {
		loc = loc + fmt.Sprintf("#comment-%d", id)
	}
	http.Redirect(w, r, loc, http.StatusFound)
}

func requestIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		return host[:i]
	}
	return host
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}

func optionTruthyServer(c *wp.Context, name string) bool {
	if c == nil || c.Options == nil {
		return false
	}
	v := c.Options.GetOption(name, "")
	switch t := v.(type) {
	case bool:
		return t
	case int:
		return t != 0
	case int64:
		return t != 0
	case string:
		s := strings.TrimSpace(t)
		return s != "" && s != "0" && !strings.EqualFold(s, "false")
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		return s != "" && s != "0" && s != "<nil>"
	}
}
